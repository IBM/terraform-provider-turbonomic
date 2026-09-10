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
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	turboclient "github.com/IBM/turbonomic-go-client"
)

var (
	_ datasource.DataSource              = &kubernetesVolumeDataSource{}
	_ datasource.DataSourceWithConfigure = &kubernetesVolumeDataSource{}
)

// KubernetesVolumeEntityModel is the full data source state for turbonomic_kubernetes_volume.
type KubernetesVolumeEntityModel struct {
	// Input fields
	Cluster        types.String `tfsdk:"cluster"`
	Name           types.String `tfsdk:"name"`
	DefaultSizeGiB types.Int64  `tfsdk:"default_size_gib"`

	// Output fields
	EntityUUID    types.String `tfsdk:"entity_uuid"`
	CurrentSizeGiB types.Int64  `tfsdk:"current_size_gib"`
	NewSizeGiB    types.Int64  `tfsdk:"new_size_gib"`
	ActionType    types.String `tfsdk:"action_type"`
	ActionState   types.String `tfsdk:"action_state"`
	ActionMode    types.String `tfsdk:"action_mode"`
}

type kubernetesVolumeDataSource struct {
	client *turboclient.Client
}

func NewKubernetesVolumeDataSource() datasource.DataSource {
	return &kubernetesVolumeDataSource{}
}

func (d *kubernetesVolumeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_volume"
}

func (d *kubernetesVolumeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves Turbonomic optimization recommendations for a Kubernetes PersistentVolumeClaim (PVC). " +
			"Kubernetes PVCs are modeled as VirtualVolume entities with a Kubernetes discoveredBy probe. " +
			"Returns a recommended storage size in GiB when Turbonomic has a SCALE action for the volume.",
		Attributes: map[string]schema.Attribute{
			// ── Input ──────────────────────────────────────────────────────────
			"cluster": schema.StringAttribute{
				MarkdownDescription: "Name of the Kubernetes cluster target as it appears in Turbonomic " +
					"(e.g. `\"Kubernetes-mycluster\"`). Used to disambiguate PVCs with the same name across clusters.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the PersistentVolumeClaim as discovered by Turbonomic " +
					"(e.g. `\"pvc-0bab48cf-92c5-4e79-9dd8-62bd4d2fa634\"`).",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"default_size_gib": schema.Int64Attribute{
				MarkdownDescription: "Fallback storage size in GiB to use when Turbonomic has no SCALE recommendation.",
				Optional:            true,
			},

			// ── Output ─────────────────────────────────────────────────────────
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the VirtualVolume entity.",
				Computed:            true,
			},
			"current_size_gib": schema.Int64Attribute{
				MarkdownDescription: "Current storage size in GiB as reported by Turbonomic.",
				Computed:            true,
			},
			"new_size_gib": schema.Int64Attribute{
				MarkdownDescription: "Recommended storage size in GiB. Falls back to `current_size_gib`, then `default_size_gib`.",
				Computed:            true,
			},
			"action_type": schema.StringAttribute{
				MarkdownDescription: "Type of the pending action (`SCALE` or empty when no action exists).",
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

func (d *kubernetesVolumeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *kubernetesVolumeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state KubernetesVolumeEntityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		setDefaultsKubernetesVolumeToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 1. Search for the VirtualVolume (k8s PVC) ────────────────────────────
	entityArgs := []EntityOption{
		WithEntityName(state.Name.ValueString()),
		WithEntityType("VirtualVolume"),
	}
	entities, errDiag := GetEntitiesByName(d.client, entityArgs...)

	var errDetail string
	if errDiag != nil {
		errDetail = errDiag.Detail()
	} else if len(entities) == 0 {
		errDetail = fmt.Sprintf("kubernetes volume %q not found", state.Name.ValueString())
	}

	if errDetail == "" {
		// Filter by cluster (discoveredBy.DisplayName) and Kubernetes probe type.
		cluster := state.Cluster.ValueString()
		filtered := entities[:0]
		for _, e := range entities {
			if e.DiscoveredBy.DisplayName == cluster && e.DiscoveredBy.Type == "Kubernetes" {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			errDetail = fmt.Sprintf("kubernetes volume %q not found in cluster %q", state.Name.ValueString(), cluster)
		} else {
			entities = filtered
		}
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an entity", errDetail)
		setDefaultsKubernetesVolumeToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	entity := entities[0]
	state.EntityUUID = types.StringValue(entity.UUID)
	tflog.Debug(ctx, fmt.Sprintf("kubernetes volume entity found: %s", entity.UUID))

	// ── 2. Fetch SCALE actions ────────────────────────────────────────────────
	actions, errDiag := GetFilteredEntityActions(d.client, entity.UUID, []string{"SCALE"}, []string{"READY"})
	if errDiag != nil {
		errDetail = errDiag.Detail()
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an action", errDetail)
		setDefaultsKubernetesVolumeToNewState(&state)
		if err := TagEntity(d.client, entity.UUID); err != nil {
			resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 3. Parse SCALE action ─────────────────────────────────────────────────
	if len(actions) > 0 {
		a := actions[0]
		tflog.Debug(ctx, fmt.Sprintf("kubernetes volume action found: type=%s state=%s", a.ActionType, a.ActionState))
		state.ActionType = types.StringValue(a.ActionType)
		state.ActionState = types.StringValue(a.ActionState)
		state.ActionMode = types.StringValue(a.ActionMode)

		// currentValue / newValue are storage in MiB — convert to GiB.
		if curMiB, err := strconv.ParseFloat(a.CurrentValue, 64); err == nil {
			state.CurrentSizeGiB = types.Int64Value(convertMiBtoGiB(curMiB))
		}
		if newMiB, err := strconv.ParseFloat(a.NewValue, 64); err == nil {
			state.NewSizeGiB = types.Int64Value(convertMiBtoGiB(newMiB))
		}
	} else {
		state.ActionType = types.StringValue("")
		state.ActionState = types.StringValue("")
		state.ActionMode = types.StringValue("")
	}

	// ── 4. Apply fallback chain ───────────────────────────────────────────────
	setDefaultsKubernetesVolumeToCurrentState(&state)
	setDefaultsKubernetesVolumeToNewState(&state)

	if err := TagEntity(d.client, entity.UUID); err != nil {
		resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func setDefaultsKubernetesVolumeToCurrentState(state *KubernetesVolumeEntityModel) {
	state.CurrentSizeGiB = applyDefaultIfEmptyGeneric(state.CurrentSizeGiB, state.DefaultSizeGiB)
}

func setDefaultsKubernetesVolumeToNewState(state *KubernetesVolumeEntityModel) {
	state.NewSizeGiB = applyDefaultIfEmptyGeneric(state.NewSizeGiB, state.CurrentSizeGiB)
	state.NewSizeGiB = applyDefaultIfEmptyGeneric(state.NewSizeGiB, state.DefaultSizeGiB)
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
	if state.CurrentSizeGiB.IsNull() {
		state.CurrentSizeGiB = types.Int64Value(0)
	}
	if state.NewSizeGiB.IsNull() {
		state.NewSizeGiB = types.Int64Value(0)
	}
}
