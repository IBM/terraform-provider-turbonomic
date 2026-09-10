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

// aksNodePoolTagKey is the Turbonomic tag key used to identify an AKS node pool
// membership on a node VM. Confirmed from live almostlive API research.
const aksNodePoolTagKey = "aks-managed-poolName"

var (
	_ datasource.DataSource              = &azurermAKSNodePoolDataSource{}
	_ datasource.DataSourceWithConfigure = &azurermAKSNodePoolDataSource{}
)

// AzurermAKSNodePoolEntityModel is the full data source state.
type AzurermAKSNodePoolEntityModel struct {
	// Input fields
	ClusterName      types.String `tfsdk:"cluster_name"`
	Name             types.String `tfsdk:"name"`
	DefaultNodeCount types.Int64  `tfsdk:"default_node_count"`

	// Output fields
	EntityUUID       types.String `tfsdk:"entity_uuid"`
	CurrentNodeCount types.Int64  `tfsdk:"current_node_count"`
	NewNodeCount     types.Int64  `tfsdk:"new_node_count"`
	ActionType       types.String `tfsdk:"action_type"`
	ActionState      types.String `tfsdk:"action_state"`
	ActionMode       types.String `tfsdk:"action_mode"`
}

type azurermAKSNodePoolDataSource struct {
	client *turboclient.Client
}

func NewAzurermAKSNodePoolDataSource() datasource.DataSource {
	return &azurermAKSNodePoolDataSource{}
}

func (d *azurermAKSNodePoolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azurerm_aks_node_pool"
}

func (d *azurermAKSNodePoolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves Turbonomic optimization recommendations for an Azure AKS node pool. " +
			"Returns the recommended node count derived from PROVISION/SUSPEND actions on the node VMs that belong to the pool.",
		Attributes: map[string]schema.Attribute{
			// ── Input ──────────────────────────────────────────────────────────
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "Name of the AKS cluster as it appears in Turbonomic.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the AKS node pool (matched against the `aks-managed-poolName` tag on node VMs in Turbonomic).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"default_node_count": schema.Int64Attribute{
				MarkdownDescription: "Fallback node count to use when Turbonomic has no recommendation.",
				Optional:            true,
			},

			// ── Output ─────────────────────────────────────────────────────────
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the first matched node VM (representative entity for the node pool).",
				Computed:            true,
			},
			"current_node_count": schema.Int64Attribute{
				MarkdownDescription: "Current number of node VMs discovered in this node pool.",
				Computed:            true,
			},
			"new_node_count": schema.Int64Attribute{
				MarkdownDescription: "Recommended node count. Falls back to `current_node_count`, then `default_node_count`.",
				Computed:            true,
			},
			"action_type": schema.StringAttribute{
				MarkdownDescription: "Type of the pending action (`PROVISION`, `SUSPEND`, or empty when no recommendation exists).",
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

func (d *azurermAKSNodePoolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *azurermAKSNodePoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AzurermAKSNodePoolEntityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		setDefaultsAKSNodePoolToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 1. Search VirtualMachine entities scoped to Azure cloud ──────────────
	entities, errDiag := GetAllEntitiesByCloudType(d.client, VirtualMachineEntityType, "AZURE", "CLOUD")

	var errDetail string
	if errDiag != nil {
		errDetail = errDiag.Detail()
	}

	if errDetail == "" {
		// Filter client-side: tag key must match node pool name
		entities = filterVMsByTag(entities, aksNodePoolTagKey, state.Name.ValueString())
		// Further filter by cluster (discoveredBy.displayName)
		cluster := state.ClusterName.ValueString()
		filtered := entities[:0]
		for _, e := range entities {
			if e.DiscoveredBy.DisplayName == cluster {
				filtered = append(filtered, e)
			}
		}
		entities = filtered
	}

	if errDetail == "" && len(entities) == 0 {
		errDetail = fmt.Sprintf("no AKS node VMs found for node pool %q in cluster %q", state.Name.ValueString(), state.ClusterName.ValueString())
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an entity", errDetail)
		setDefaultsAKSNodePoolToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	state.EntityUUID = types.StringValue(entities[0].UUID)
	state.CurrentNodeCount = types.Int64Value(int64(len(entities)))
	tflog.Debug(ctx, fmt.Sprintf("azurerm aks node pool found: %d node VMs in cluster %q, representative entity: %s", len(entities), state.ClusterName.ValueString(), entities[0].UUID))

	// ── 2. Aggregate PROVISION/SUSPEND actions across all node VMs ────────────
	totalDelta := 0
	var firstType, firstState, firstMode string

	for _, e := range entities {
		actions, errDiag := GetFilteredEntityActions(d.client, e.UUID, []string{"PROVISION", "SUSPEND"}, []string{"READY"})
		if errDiag != nil {
			tflog.Warn(ctx, fmt.Sprintf("error getting actions for node VM %s: %s", e.UUID, errDiag.Detail()))
			continue
		}
		if len(actions) > 0 {
			tflog.Debug(ctx, fmt.Sprintf("azurerm aks node pool action found for node VM %s: type=%s state=%s", e.UUID, actions[0].ActionType, actions[0].ActionState))
		}
		totalDelta += nodePoolActionDelta(actions)
		if firstType == "" {
			firstType, firstState, firstMode = firstActionMeta(actions)
		}
	}

	state.ActionType = types.StringValue(firstType)
	state.ActionState = types.StringValue(firstState)
	state.ActionMode = types.StringValue(firstMode)

	if totalDelta != 0 {
		state.NewNodeCount = types.Int64Value(state.CurrentNodeCount.ValueInt64() + int64(totalDelta))
	}

	// ── 3. Fallback chain ─────────────────────────────────────────────────────
	setDefaultsAKSNodePoolToNewState(&state)

	if err := TagEntity(d.client, entities[0].UUID); err != nil {
		resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func setDefaultsAKSNodePoolToNewState(state *AzurermAKSNodePoolEntityModel) {
	state.NewNodeCount = applyDefaultIfEmptyGeneric(state.NewNodeCount, state.CurrentNodeCount)
	state.NewNodeCount = applyDefaultIfEmptyGeneric(state.NewNodeCount, state.DefaultNodeCount)
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
	if state.CurrentNodeCount.IsNull() {
		state.CurrentNodeCount = types.Int64Value(0)
	}
}
