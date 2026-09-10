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
	_ datasource.DataSource              = &kubernetesNamespaceDataSource{}
	_ datasource.DataSourceWithConfigure = &kubernetesNamespaceDataSource{}
)

// KubernetesNamespaceEntityModel is the full data source state for turbonomic_kubernetes_namespace.
type KubernetesNamespaceEntityModel struct {
	// Input fields
	Cluster                    types.String `tfsdk:"cluster"`
	Name                       types.String `tfsdk:"name"`
	DefaultCPULimitQuota       types.String `tfsdk:"default_cpu_limit_quota"`
	DefaultCPURequestQuota     types.String `tfsdk:"default_cpu_request_quota"`
	DefaultMemLimitQuota       types.String `tfsdk:"default_mem_limit_quota"`
	DefaultMemRequestQuota     types.String `tfsdk:"default_mem_request_quota"`

	// Output fields
	EntityUUID             types.String `tfsdk:"entity_uuid"`
	ActionType             types.String `tfsdk:"action_type"`
	ActionState            types.String `tfsdk:"action_state"`
	ActionMode             types.String `tfsdk:"action_mode"`
	CurrentCPULimitQuota   types.String `tfsdk:"current_cpu_limit_quota"`
	NewCPULimitQuota       types.String `tfsdk:"new_cpu_limit_quota"`
	CurrentCPURequestQuota types.String `tfsdk:"current_cpu_request_quota"`
	NewCPURequestQuota     types.String `tfsdk:"new_cpu_request_quota"`
	CurrentMemLimitQuota   types.String `tfsdk:"current_mem_limit_quota"`
	NewMemLimitQuota       types.String `tfsdk:"new_mem_limit_quota"`
	CurrentMemRequestQuota types.String `tfsdk:"current_mem_request_quota"`
	NewMemRequestQuota     types.String `tfsdk:"new_mem_request_quota"`
}

type kubernetesNamespaceDataSource struct {
	client *turboclient.Client
}

func NewKubernetesNamespaceDataSource() datasource.DataSource {
	return &kubernetesNamespaceDataSource{}
}

func (d *kubernetesNamespaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_namespace"
}

func (d *kubernetesNamespaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves Turbonomic optimization recommendations for a Kubernetes Namespace. " +
			"Returns recommended quota values for CPU and memory requests and limits. " +
			"Turbonomic only resizes namespace quotas upward — recommendations will never reduce a quota below its current value.",
		Attributes: map[string]schema.Attribute{
			// ── Input ──────────────────────────────────────────────────────────
			"cluster": schema.StringAttribute{
				MarkdownDescription: "Name of the Kubernetes cluster as it appears in Turbonomic (e.g. `\"Kubernetes-mycluster\"`). " +
					"Used to disambiguate namespaces with the same name across multiple clusters.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Kubernetes namespace (e.g. `\"production\"`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"default_cpu_limit_quota": schema.StringAttribute{
				MarkdownDescription: "Fallback CPU limit quota (in mCores) when Turbonomic has no recommendation.",
				Optional:            true,
			},
			"default_cpu_request_quota": schema.StringAttribute{
				MarkdownDescription: "Fallback CPU request quota (in mCores) when Turbonomic has no recommendation.",
				Optional:            true,
			},
			"default_mem_limit_quota": schema.StringAttribute{
				MarkdownDescription: "Fallback memory limit quota (in MB) when Turbonomic has no recommendation.",
				Optional:            true,
			},
			"default_mem_request_quota": schema.StringAttribute{
				MarkdownDescription: "Fallback memory request quota (in MB) when Turbonomic has no recommendation.",
				Optional:            true,
			},

			// ── Output ─────────────────────────────────────────────────────────
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the Namespace entity.",
				Computed:            true,
			},
			"action_type": schema.StringAttribute{
				MarkdownDescription: "Type of the pending action (`RESIZE` or empty when no action exists).",
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
			"current_cpu_limit_quota": schema.StringAttribute{
				MarkdownDescription: "Current CPU limit quota in mCores as reported by Turbonomic.",
				Computed:            true,
			},
			"new_cpu_limit_quota": schema.StringAttribute{
				MarkdownDescription: "Recommended CPU limit quota in mCores. Falls back to `default_cpu_limit_quota`.",
				Computed:            true,
			},
			"current_cpu_request_quota": schema.StringAttribute{
				MarkdownDescription: "Current CPU request quota in mCores as reported by Turbonomic.",
				Computed:            true,
			},
			"new_cpu_request_quota": schema.StringAttribute{
				MarkdownDescription: "Recommended CPU request quota in mCores. Falls back to `default_cpu_request_quota`.",
				Computed:            true,
			},
			"current_mem_limit_quota": schema.StringAttribute{
				MarkdownDescription: "Current memory limit quota in MB as reported by Turbonomic.",
				Computed:            true,
			},
			"new_mem_limit_quota": schema.StringAttribute{
				MarkdownDescription: "Recommended memory limit quota in MB. Falls back to `default_mem_limit_quota`.",
				Computed:            true,
			},
			"current_mem_request_quota": schema.StringAttribute{
				MarkdownDescription: "Current memory request quota in MB as reported by Turbonomic.",
				Computed:            true,
			},
			"new_mem_request_quota": schema.StringAttribute{
				MarkdownDescription: "Recommended memory request quota in MB. Falls back to `default_mem_request_quota`.",
				Computed:            true,
			},
		},
	}
}

func (d *kubernetesNamespaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *kubernetesNamespaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state KubernetesNamespaceEntityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		setDefaultsKubernetesNamespaceToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 1. Search for the Namespace entity ───────────────────────────────────
	entityArgs := []EntityOption{
		WithEntityName(state.Name.ValueString()),
		WithEntityType("Namespace"),
	}
	entities, errDiag := GetEntitiesByName(d.client, entityArgs...)

	var errDetail string
	if errDiag != nil {
		errDetail = errDiag.Detail()
	} else if len(entities) == 0 {
		errDetail = fmt.Sprintf("namespace %q not found", state.Name.ValueString())
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
			errDetail = fmt.Sprintf("namespace %q not found in cluster %q", state.Name.ValueString(), cluster)
		} else {
			entities = filtered
		}
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an entity", errDetail)
		setDefaultsKubernetesNamespaceToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	entity := entities[0]
	state.EntityUUID = types.StringValue(entity.UUID)
	tflog.Debug(ctx, fmt.Sprintf("kubernetes namespace entity found: %s", entity.UUID))

	// ── 2. Fetch RESIZE actions ───────────────────────────────────────────────
	actions, errDiag := GetFilteredEntityActions(d.client, entity.UUID, []string{"RESIZE"}, []string{"READY"})
	if errDiag != nil {
		errDetail = errDiag.Detail()
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an action", errDetail)
		setDefaultsKubernetesNamespaceToNewState(&state)
		if err := TagEntity(d.client, entity.UUID); err != nil {
			resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 3. Set action metadata from first action ──────────────────────────────
	if len(actions) > 0 {
		tflog.Debug(ctx, fmt.Sprintf("kubernetes namespace action found: type=%s state=%s", actions[0].ActionType, actions[0].ActionState))
		state.ActionType = types.StringValue(actions[0].ActionType)
		state.ActionState = types.StringValue(actions[0].ActionState)
		state.ActionMode = types.StringValue(actions[0].ActionMode)
	} else {
		state.ActionType = types.StringValue("")
		state.ActionState = types.StringValue("")
		state.ActionMode = types.StringValue("")
	}

	// ── 4. Parse quota recommendations from all RESIZE actions ───────────────
	//
	// Live API shape: a single Namespace RESIZE action
	// may cover multiple commodity types. The per-commodity current/new values and
	// commodity identification are in compoundActions[], NOT in the top-level
	// currentValue/newValue fields (those are null for Namespace quota actions).
	// Each compoundAction has: currentValue, newValue, valueUnits, and
	// risk.reasonCommodities[0] identifying the commodity type.
	//
	// For backward compatibility with older API responses that do carry top-level
	// currentValue/newValue (one commodity per action), we fall back to reading the
	// top-level fields when compoundActions is empty.
	for _, a := range actions {
		type quotaEntry struct {
			cur      string
			nw       string
			commodity string
		}
		var entries []quotaEntry

		if len(a.CompoundActions) > 0 {
			for _, ca := range a.CompoundActions {
				commodity := ""
				if len(ca.Risk.ReasonCommodities) > 0 {
					commodity = ca.Risk.ReasonCommodities[0]
				}
				entries = append(entries, quotaEntry{cur: ca.CurrentValue, nw: ca.NewValue, commodity: commodity})
			}
		} else {
			// Fallback: older single-commodity-per-action format
			commodity := ""
			if len(a.Risk.ReasonCommodities) > 0 {
				commodity = a.Risk.ReasonCommodities[0]
			}
			entries = append(entries, quotaEntry{cur: a.CurrentValue, nw: a.NewValue, commodity: commodity})
		}

		for _, e := range entries {
			switch e.commodity {
			case "VCPULimitQuota", "VCPU_LIMIT_QUOTA":
				state.CurrentCPULimitQuota = types.StringValue(e.cur)
				state.NewCPULimitQuota = types.StringValue(e.nw)
			case "VCPURequestQuota", "VCPU_REQUEST_QUOTA":
				state.CurrentCPURequestQuota = types.StringValue(e.cur)
				state.NewCPURequestQuota = types.StringValue(e.nw)
			case "VMemLimitQuota", "VMEM_LIMIT_QUOTA":
				state.CurrentMemLimitQuota = types.StringValue(e.cur)
				state.NewMemLimitQuota = types.StringValue(e.nw)
			case "VMemRequestQuota", "VMEM_REQUEST_QUOTA":
				state.CurrentMemRequestQuota = types.StringValue(e.cur)
				state.NewMemRequestQuota = types.StringValue(e.nw)
			}
		}
	}

	// ── 5. Apply default fallbacks ────────────────────────────────────────────
	setDefaultsKubernetesNamespaceToCurrentState(&state)
	setCurrentKubernetesNamespaceToNewState(&state)
	setDefaultsKubernetesNamespaceToNewState(&state)

	if err := TagEntity(d.client, entity.UUID); err != nil {
		resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// setDefaultsKubernetesNamespaceToCurrentState sets current quota fields to the
// user-supplied defaults when the API returned no value for them.
func setDefaultsKubernetesNamespaceToCurrentState(state *KubernetesNamespaceEntityModel) {
	state.CurrentCPULimitQuota = applyDefaultIfEmptyGeneric(state.CurrentCPULimitQuota, state.DefaultCPULimitQuota)
	state.CurrentCPURequestQuota = applyDefaultIfEmptyGeneric(state.CurrentCPURequestQuota, state.DefaultCPURequestQuota)
	state.CurrentMemLimitQuota = applyDefaultIfEmptyGeneric(state.CurrentMemLimitQuota, state.DefaultMemLimitQuota)
	state.CurrentMemRequestQuota = applyDefaultIfEmptyGeneric(state.CurrentMemRequestQuota, state.DefaultMemRequestQuota)
}

// setCurrentKubernetesNamespaceToNewState propagates current quota values into
// the new fields so that a partial or absent recommendation does not leave new
// values empty.
func setCurrentKubernetesNamespaceToNewState(state *KubernetesNamespaceEntityModel) {
	state.NewCPULimitQuota = applyDefaultIfEmptyGeneric(state.NewCPULimitQuota, state.CurrentCPULimitQuota)
	state.NewCPURequestQuota = applyDefaultIfEmptyGeneric(state.NewCPURequestQuota, state.CurrentCPURequestQuota)
	state.NewMemLimitQuota = applyDefaultIfEmptyGeneric(state.NewMemLimitQuota, state.CurrentMemLimitQuota)
	state.NewMemRequestQuota = applyDefaultIfEmptyGeneric(state.NewMemRequestQuota, state.CurrentMemRequestQuota)
}

func setDefaultsKubernetesNamespaceToNewState(state *KubernetesNamespaceEntityModel) {
	state.NewCPULimitQuota = applyDefaultIfEmptyGeneric(state.NewCPULimitQuota, state.DefaultCPULimitQuota)
	state.NewCPURequestQuota = applyDefaultIfEmptyGeneric(state.NewCPURequestQuota, state.DefaultCPURequestQuota)
	state.NewMemLimitQuota = applyDefaultIfEmptyGeneric(state.NewMemLimitQuota, state.DefaultMemLimitQuota)
	state.NewMemRequestQuota = applyDefaultIfEmptyGeneric(state.NewMemRequestQuota, state.DefaultMemRequestQuota)

	if state.ActionType.IsNull() {
		state.ActionType = types.StringValue("")
	}
	if state.ActionState.IsNull() {
		state.ActionState = types.StringValue("")
	}
	if state.ActionMode.IsNull() {
		state.ActionMode = types.StringValue("")
	}
	if state.CurrentCPULimitQuota.IsNull() {
		state.CurrentCPULimitQuota = types.StringValue("")
	}
	if state.CurrentCPURequestQuota.IsNull() {
		state.CurrentCPURequestQuota = types.StringValue("")
	}
	if state.CurrentMemLimitQuota.IsNull() {
		state.CurrentMemLimitQuota = types.StringValue("")
	}
	if state.CurrentMemRequestQuota.IsNull() {
		state.CurrentMemRequestQuota = types.StringValue("")
	}
	if state.NewCPULimitQuota.IsNull() {
		state.NewCPULimitQuota = types.StringValue("")
	}
	if state.NewCPURequestQuota.IsNull() {
		state.NewCPURequestQuota = types.StringValue("")
	}
	if state.NewMemLimitQuota.IsNull() {
		state.NewMemLimitQuota = types.StringValue("")
	}
	if state.NewMemRequestQuota.IsNull() {
		state.NewMemRequestQuota = types.StringValue("")
	}
	if state.EntityUUID.IsNull() {
		state.EntityUUID = types.StringValue("")
	}
}
