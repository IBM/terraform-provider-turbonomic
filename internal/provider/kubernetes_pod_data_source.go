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
	_ datasource.DataSource              = &kubernetesPodDataSource{}
	_ datasource.DataSourceWithConfigure = &kubernetesPodDataSource{}
)

// KubernetesPodEntityModel is the full data source state for turbonomic_kubernetes_pod.
type KubernetesPodEntityModel struct {
	// Input fields
	Cluster   types.String `tfsdk:"cluster"`
	Namespace types.String `tfsdk:"namespace"`
	Name      types.String `tfsdk:"name"`

	// Output fields
	EntityUUID      types.String `tfsdk:"entity_uuid"`
	ActionType      types.String `tfsdk:"action_type"`
	ActionState     types.String `tfsdk:"action_state"`
	ActionMode      types.String `tfsdk:"action_mode"`
	CurrentNode     types.String `tfsdk:"current_node"`
	NewNode         types.String `tfsdk:"new_node"`
	CurrentNodeUUID types.String `tfsdk:"current_node_uuid"`
	NewNodeUUID     types.String `tfsdk:"new_node_uuid"`
}

type kubernetesPodDataSource struct {
	client *turboclient.Client
}

func NewKubernetesPodDataSource() datasource.DataSource {
	return &kubernetesPodDataSource{}
}

func (d *kubernetesPodDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_pod"
}

func (d *kubernetesPodDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves Turbonomic optimization recommendations for a Kubernetes ContainerPod. " +
			"Returns the recommended destination node for a MOVE action. " +
			"Pod MOVE recommendations indicate that a pod should be rescheduled to another node for better resource utilization.",
		Attributes: map[string]schema.Attribute{
			// ── Input ──────────────────────────────────────────────────────────
			"cluster": schema.StringAttribute{
				MarkdownDescription: "Name of the Kubernetes cluster as it appears in Turbonomic (e.g. `\"Kubernetes-mycluster\"`). " +
					"Used to disambiguate pods with the same name across multiple clusters.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Kubernetes namespace the pod belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Kubernetes pod. " +
					"Turbonomic stores pod display names as `\"<namespace>/<pod-name>\"` — provide only the pod name here.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			// ── Output ─────────────────────────────────────────────────────────
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the ContainerPod entity.",
				Computed:            true,
			},
			"action_type": schema.StringAttribute{
				MarkdownDescription: "Type of the pending action (`MOVE` or empty when no action exists).",
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
			"current_node": schema.StringAttribute{
				MarkdownDescription: "Display name of the node currently hosting the pod.",
				Computed:            true,
			},
			"new_node": schema.StringAttribute{
				MarkdownDescription: "Display name of the recommended destination node.",
				Computed:            true,
			},
			"current_node_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the current node.",
				Computed:            true,
			},
			"new_node_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the recommended destination node.",
				Computed:            true,
			},
		},
	}
}

func (d *kubernetesPodDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *kubernetesPodDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state KubernetesPodEntityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		setDefaultsKubernetesPodToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 1. Search for the ContainerPod ───────────────────────────────────────
	// Pod display names in Turbonomic are "<namespace>/<pod-name>".
	podDisplayName := state.Namespace.ValueString() + "/" + state.Name.ValueString()
	entityArgs := []EntityOption{
		WithEntityName(podDisplayName),
		WithEntityType("ContainerPod"),
	}
	entities, errDiag := GetEntitiesByName(d.client, entityArgs...)

	var errDetail string
	if errDiag != nil {
		errDetail = errDiag.Detail()
	} else if len(entities) == 0 {
		errDetail = fmt.Sprintf("pod %q not found in namespace %q", state.Name.ValueString(), state.Namespace.ValueString())
	}

	if errDetail == "" {
		// Filter client-side by cluster (discoveredBy.DisplayName)
		cluster := state.Cluster.ValueString()
		filtered := entities[:0]
		for _, e := range entities {
			if e.DiscoveredBy.DisplayName == cluster {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			errDetail = fmt.Sprintf("pod %q not found in cluster %q", podDisplayName, cluster)
		} else {
			entities = filtered
		}
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an entity", errDetail)
		setDefaultsKubernetesPodToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	entity := entities[0]
	state.EntityUUID = types.StringValue(entity.UUID)
	tflog.Debug(ctx, fmt.Sprintf("kubernetes pod entity found: %s", entity.UUID))

	// ── 2. Fetch MOVE actions ─────────────────────────────────────────────────
	actions, errDiag := GetFilteredEntityActions(d.client, entity.UUID, []string{"MOVE"}, []string{"READY"})
	if errDiag != nil {
		errDetail = errDiag.Detail()
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an action", errDetail)
		setDefaultsKubernetesPodToNewState(&state)
		if err := TagEntity(d.client, entity.UUID); err != nil {
			resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 3. Parse MOVE action ──────────────────────────────────────────────────
	if len(actions) > 0 {
		a := actions[0]
		tflog.Debug(ctx, fmt.Sprintf("kubernetes pod action found: type=%s state=%s", a.ActionType, a.ActionState))
		state.ActionType = types.StringValue(a.ActionType)
		state.ActionState = types.StringValue(a.ActionState)
		state.ActionMode = types.StringValue(a.ActionMode)

		// currentEntity is the source node (VirtualMachine), newEntity is the destination node.
		state.CurrentNode = types.StringValue(a.CurrentEntity.DisplayName)
		state.CurrentNodeUUID = types.StringValue(a.CurrentEntity.UUID)
		state.NewNode = types.StringValue(a.NewEntity.DisplayName)
		state.NewNodeUUID = types.StringValue(a.NewEntity.UUID)
	} else {
		setDefaultsKubernetesPodToNewState(&state)
	}

	if err := TagEntity(d.client, entity.UUID); err != nil {
		resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func setDefaultsKubernetesPodToNewState(state *KubernetesPodEntityModel) {
	if state.ActionType.IsNull() {
		state.ActionType = types.StringValue("")
	}
	if state.ActionState.IsNull() {
		state.ActionState = types.StringValue("")
	}
	if state.ActionMode.IsNull() {
		state.ActionMode = types.StringValue("")
	}
	if state.CurrentNode.IsNull() {
		state.CurrentNode = types.StringValue("")
	}
	if state.NewNode.IsNull() {
		state.NewNode = types.StringValue("")
	}
	if state.CurrentNodeUUID.IsNull() {
		state.CurrentNodeUUID = types.StringValue("")
	}
	if state.NewNodeUUID.IsNull() {
		state.NewNodeUUID = types.StringValue("")
	}
	if state.EntityUUID.IsNull() {
		state.EntityUUID = types.StringValue("")
	}
}
