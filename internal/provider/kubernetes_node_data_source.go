// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS-IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	turboclient "github.com/IBM/turbonomic-go-client"
)

var (
	_ datasource.DataSource              = &kubernetesNodeDataSource{}
	_ datasource.DataSourceWithConfigure = &kubernetesNodeDataSource{}
)

// KubernetesNodeEntityModel is the full data source state for turbonomic_kubernetes_node.
type KubernetesNodeEntityModel struct {
	// Input fields
	Cluster types.String `tfsdk:"cluster"`
	Name    types.String `tfsdk:"name"`

	// Output fields
	EntityUUID  types.String `tfsdk:"entity_uuid"`
	ActionType  types.String `tfsdk:"action_type"`
	ActionState types.String `tfsdk:"action_state"`
	ActionMode  types.String `tfsdk:"action_mode"`
}

type kubernetesNodeDataSource struct {
	client *turboclient.Client
}

func NewKubernetesNodeDataSource() datasource.DataSource {
	return &kubernetesNodeDataSource{}
}

func (d *kubernetesNodeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_node"
}

func (d *kubernetesNodeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves Turbonomic optimization recommendations for a Kubernetes node. " +
			"Returns the pending action type — PROVISION (add capacity), SUSPEND (remove underutilized node), " +
			"or RECONFIGURE (fix a NotReady node). " +
			"Kubernetes nodes are modeled as VirtualMachine entities in Turbonomic; this data source " +
			"filters to k8s-discovered VMs only.",
		Attributes: map[string]schema.Attribute{
			// ── Input ──────────────────────────────────────────────────────────
			"cluster": schema.StringAttribute{
				MarkdownDescription: "Name of the Kubernetes cluster as it appears in Turbonomic (e.g. `\"Kubernetes-mycluster\"`). " +
					"Used to filter node VMs discovered by a specific cluster and exclude cloud-provider VMs with the same hostname.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Kubernetes node (e.g. `\"ip-10-0-1-42.ec2.internal\"`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			// ── Output ─────────────────────────────────────────────────────────
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the node (VirtualMachine) entity.",
				Computed:            true,
			},
			"action_type": schema.StringAttribute{
				MarkdownDescription: "Type of the pending action: `PROVISION`, `SUSPEND`, `RECONFIGURE`, or empty when no action exists.",
				Computed:            true,
			},
			"action_state": schema.StringAttribute{
				MarkdownDescription: "State of the pending action (e.g. `READY`).",
				Computed:            true,
			},
			"action_mode": schema.StringAttribute{
				MarkdownDescription: "Mode of the pending action (e.g. `MANUAL`, `RECOMMEND`).",
				Computed:            true,
			},
		},
	}
}

func (d *kubernetesNodeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected data-source configure type",
			fmt.Sprintf("expected: *providerData, got: %T. please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = data.Client.(*turboclient.Client)
}

func (d *kubernetesNodeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state KubernetesNodeEntityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		setDefaultsKubernetesNodeToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 1. Search for the k8s node (VirtualMachine with Kubernetes discoveredBy) ──
	entityArgs := []EntityOption{
		WithEntityName(state.Name.ValueString()),
		WithEntityType("VirtualMachine"),
	}
	entities, errDiag := GetEntitiesByName(d.client, entityArgs...)

	var errDetail string
	if errDiag != nil {
		errDetail = errDiag.Detail()
	} else if len(entities) == 0 {
		errDetail = fmt.Sprintf("node %q not found", state.Name.ValueString())
	}

	if errDetail == "" {
		// Filter by cluster (discoveredBy.DisplayName) to exclude cloud VMs with the same hostname.
		// Also filter to Kubernetes-discovered VMs only (discoveredBy.Type == "Kubernetes").
		cluster := state.Cluster.ValueString()
		filtered := entities[:0]
		for _, e := range entities {
			if e.DiscoveredBy.DisplayName == cluster && e.DiscoveredBy.Type == "Kubernetes" {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			errDetail = fmt.Sprintf("k8s node %q not found in cluster %q", state.Name.ValueString(), cluster)
		} else {
			entities = filtered
		}
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an entity", errDetail)
		setDefaultsKubernetesNodeToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	entity := entities[0]
	state.EntityUUID = types.StringValue(entity.UUID)
	tflog.Debug(ctx, fmt.Sprintf("kubernetes node entity found: %s", entity.UUID))

	// ── 2. Fetch PROVISION, SUSPEND, RECONFIGURE actions ─────────────────────
	actions, errDiag := GetFilteredEntityActions(d.client, entity.UUID, []string{"PROVISION", "SUSPEND", "RECONFIGURE"}, []string{"READY"})
	if errDiag != nil {
		errDetail = errDiag.Detail()
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an action", errDetail)
		setDefaultsKubernetesNodeToNewState(&state)
		if err := TagEntity(d.client, entity.UUID); err != nil {
			resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 3. Set action metadata ────────────────────────────────────────────────
	// Node actions are binary flags — no numeric current/new values.
	if len(actions) > 0 {
		tflog.Debug(ctx, fmt.Sprintf("kubernetes node action found: type=%s state=%s", actions[0].ActionType, actions[0].ActionState))
		state.ActionType = types.StringValue(actions[0].ActionType)
		state.ActionState = types.StringValue(actions[0].ActionState)
		state.ActionMode = types.StringValue(actions[0].ActionMode)
	} else {
		state.ActionType = types.StringValue("")
		state.ActionState = types.StringValue("")
		state.ActionMode = types.StringValue("")
	}

	if err := TagEntity(d.client, entity.UUID); err != nil {
		resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func setDefaultsKubernetesNodeToNewState(state *KubernetesNodeEntityModel) {
	if state.ActionType.IsNull() {
		state.ActionType = types.StringValue("")
	}
	if state.ActionState.IsNull() {
		state.ActionState = types.StringValue("")
	}
	if state.ActionMode.IsNull() {
		state.ActionMode = types.StringValue("")
	}
	if state.EntityUUID.IsNull() {
		state.EntityUUID = types.StringValue("")
	}
}
