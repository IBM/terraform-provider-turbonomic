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
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	turboclient "github.com/IBM/turbonomic-go-client"
)

var (
	_ datasource.DataSource              = &kubernetesWorkloadDataSource{}
	_ datasource.DataSourceWithConfigure = &kubernetesWorkloadDataSource{}

	kubernetesWorkloadContainerAttrTypes = map[string]attr.Type{
		"name":                   types.StringType,
		"current_cpu_request":    types.StringType,
		"new_cpu_request":        types.StringType,
		"current_cpu_limit":      types.StringType,
		"new_cpu_limit":          types.StringType,
		"current_memory_request": types.StringType,
		"new_memory_request":     types.StringType,
		"current_memory_limit":   types.StringType,
		"new_memory_limit":       types.StringType,
	}
)

// KubernetesWorkloadContainerModel is the per-container block in state.
type KubernetesWorkloadContainerModel struct {
	Name                 types.String `tfsdk:"name"`
	CurrentCPURequest    types.String `tfsdk:"current_cpu_request"`
	NewCPURequest        types.String `tfsdk:"new_cpu_request"`
	CurrentCPULimit      types.String `tfsdk:"current_cpu_limit"`
	NewCPULimit          types.String `tfsdk:"new_cpu_limit"`
	CurrentMemoryRequest types.String `tfsdk:"current_memory_request"`
	NewMemoryRequest     types.String `tfsdk:"new_memory_request"`
	CurrentMemoryLimit   types.String `tfsdk:"current_memory_limit"`
	NewMemoryLimit       types.String `tfsdk:"new_memory_limit"`
}

// KubernetesWorkloadDefaultContainerModel is the user-supplied default per-container block.
type KubernetesWorkloadDefaultContainerModel struct {
	Name                    types.String `tfsdk:"name"`
	DefaultCPURequest       types.String `tfsdk:"default_cpu_request"`
	DefaultCPULimit         types.String `tfsdk:"default_cpu_limit"`
	DefaultMemoryRequest    types.String `tfsdk:"default_memory_request"`
	DefaultMemoryLimit      types.String `tfsdk:"default_memory_limit"`
}

// KubernetesWorkloadEntityModel is the full data source state.
type KubernetesWorkloadEntityModel struct {
	// Input fields
	Cluster          types.String `tfsdk:"cluster"`
	Namespace        types.String `tfsdk:"namespace"`
	Name             types.String `tfsdk:"name"`
	Kind             types.String `tfsdk:"kind"`
	DefaultReplicas  types.Int64  `tfsdk:"default_replicas"`
	DefaultContainers []KubernetesWorkloadDefaultContainerModel `tfsdk:"default_containers"`

	// Output fields
	EntityUUID      types.String `tfsdk:"entity_uuid"`
	CurrentReplicas types.Int64  `tfsdk:"current_replicas"`
	NewReplicas     types.Int64  `tfsdk:"new_replicas"`
	Containers      types.List   `tfsdk:"containers"`
	ActionType      types.String `tfsdk:"action_type"`
	ActionState     types.String `tfsdk:"action_state"`
	ActionMode      types.String `tfsdk:"action_mode"`
}

type kubernetesWorkloadDataSource struct {
	client *turboclient.Client
}

func NewKubernetesWorkloadDataSource() datasource.DataSource {
	return &kubernetesWorkloadDataSource{}
}

func (d *kubernetesWorkloadDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_workload"
}

func (d *kubernetesWorkloadDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves Turbonomic optimization recommendations for a Kubernetes WorkloadController (Deployment, StatefulSet, etc.). " +
			"Returns recommended replica counts and per-container CPU/memory resource values.",
		Attributes: map[string]schema.Attribute{
			// ── Input ──────────────────────────────────────────────────────────
			"cluster": schema.StringAttribute{
				MarkdownDescription: "Name of the Kubernetes cluster target as it appears in Turbonomic (e.g. `\"Kubernetes-mycluster\"`). " +
					"Used to disambiguate workloads with the same name in the same namespace across multiple clusters.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Kubernetes namespace the workload belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the WorkloadController resource (e.g. Deployment name).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "Workload kind hint (e.g. `\"DEPLOYMENT_INFO\"`, `\"STATEFUL_SET_INFO\"`). " +
					"Accepted but not used for filtering: the Turbonomic API does not return `controllerType` in " +
					"WorkloadController search results, so kind-based disambiguation is not possible. " +
					"Provided for documentation purposes and future API compatibility.",
				Optional: true,
			},
			"default_replicas": schema.Int64Attribute{
				MarkdownDescription: "Fallback replica count to use when Turbonomic has no replica recommendation.",
				Optional:            true,
			},
			"default_containers": schema.ListNestedAttribute{
				MarkdownDescription: "Fallback resource values for containers when Turbonomic has no RESIZE recommendation.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Container name.",
							Required:            true,
						},
						"default_cpu_request": schema.StringAttribute{
							MarkdownDescription: "Default CPU request (e.g. `\"200m\"`).",
							Optional:            true,
						},
						"default_cpu_limit": schema.StringAttribute{
							MarkdownDescription: "Default CPU limit.",
							Optional:            true,
						},
						"default_memory_request": schema.StringAttribute{
							MarkdownDescription: "Default memory request (e.g. `\"256Mi\"`).",
							Optional:            true,
						},
						"default_memory_limit": schema.StringAttribute{
							MarkdownDescription: "Default memory limit.",
							Optional:            true,
						},
					},
				},
			},

			// ── Output ─────────────────────────────────────────────────────────
			"entity_uuid": schema.StringAttribute{
				MarkdownDescription: "Turbonomic UUID of the WorkloadController entity.",
				Computed:            true,
			},
			"current_replicas": schema.Int64Attribute{
				MarkdownDescription: "Current replica count reported by Turbonomic.",
				Computed:            true,
			},
			"new_replicas": schema.Int64Attribute{
				MarkdownDescription: "Recommended replica count. Falls back to `current_replicas`, then `default_replicas`.",
				Computed:            true,
			},
			"containers": schema.ListNestedAttribute{
				MarkdownDescription: "Per-container current and recommended resource values.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Container name.",
							Computed:            true,
						},
						"current_cpu_request": schema.StringAttribute{
							MarkdownDescription: "Current CPU request (e.g. `\"200m\"`).",
							Computed:            true,
						},
						"new_cpu_request": schema.StringAttribute{
							MarkdownDescription: "Recommended CPU request.",
							Computed:            true,
						},
						"current_cpu_limit": schema.StringAttribute{
							MarkdownDescription: "Current CPU limit.",
							Computed:            true,
						},
						"new_cpu_limit": schema.StringAttribute{
							MarkdownDescription: "Recommended CPU limit.",
							Computed:            true,
						},
						"current_memory_request": schema.StringAttribute{
							MarkdownDescription: "Current memory request (e.g. `\"256Mi\"`).",
							Computed:            true,
						},
						"new_memory_request": schema.StringAttribute{
							MarkdownDescription: "Recommended memory request.",
							Computed:            true,
						},
						"current_memory_limit": schema.StringAttribute{
							MarkdownDescription: "Current memory limit.",
							Computed:            true,
						},
						"new_memory_limit": schema.StringAttribute{
							MarkdownDescription: "Recommended memory limit.",
							Computed:            true,
						},
					},
				},
			},
			"action_type": schema.StringAttribute{
				MarkdownDescription: "Type of the pending action (`RESIZE`, `SCALE`, or empty when no action exists).",
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

func (d *kubernetesWorkloadDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *kubernetesWorkloadDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state KubernetesWorkloadEntityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		// No client — apply defaults and return
		setDefaultsKubernetesWorkloadToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 1. Search for the WorkloadController ─────────────────────────────────
	entityArgs := []EntityOption{
		WithEntityName(state.Name.ValueString()),
		WithEntityType("WorkloadController"),
		WithSearchParam("workloadControllersByNamespace", state.Namespace.ValueString()),
	}
	entities, errDiag := GetEntitiesByName(d.client, entityArgs...)

	var errDetail string
	if errDiag != nil {
		errDetail = errDiag.Detail()
	} else if len(entities) == 0 {
		errDetail = fmt.Sprintf("workload controller %q not found in namespace %q", state.Name.ValueString(), state.Namespace.ValueString())
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
			errDetail = fmt.Sprintf("workload controller %q not found in cluster %q", state.Name.ValueString(), cluster)
		} else {
			entities = filtered
		}
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an entity", errDetail)
		setDefaultsKubernetesWorkloadToNewState(&state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	entity := entities[0]
	state.EntityUUID = types.StringValue(entity.UUID)
	tflog.Debug(ctx, fmt.Sprintf("kubernetes workload entity found: %s", entity.UUID))

	// ── 2. Fetch RESIZE + SCALE actions ──────────────────────────────────────
	actions, errDiag := GetFilteredEntityActions(d.client, entity.UUID, []string{"RESIZE", "SCALE"}, []string{"READY"})

	if errDiag != nil {
		errDetail = errDiag.Detail()
	}

	if errDetail != "" {
		tflog.Warn(ctx, errDetail)
		resp.Diagnostics.AddWarning("error while getting an action", errDetail)
		setCurrentKubernetesWorkloadToNewState(&state)
		setDefaultsKubernetesWorkloadToNewState(&state)
		if err := TagEntity(d.client, entity.UUID); err != nil {
			resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	// ── 3. Set action metadata ────────────────────────────────────────────────
	if len(actions) > 0 {
		tflog.Debug(ctx, fmt.Sprintf("kubernetes workload action found: type=%s state=%s", actions[0].ActionType, actions[0].ActionState))
		state.ActionType = types.StringValue(actions[0].ActionType)
		state.ActionState = types.StringValue(actions[0].ActionState)
		state.ActionMode = types.StringValue(actions[0].ActionMode)

		// Extract current_replicas from action Target.Aspects
		if rc, ok := extractReplicaCountFromAspects(actions[0].Target.Aspects); ok {
			state.CurrentReplicas = types.Int64Value(rc)
		}
	} else {
		state.ActionType = types.StringValue("")
		state.ActionState = types.StringValue("")
		state.ActionMode = types.StringValue("")
	}

	// ── 4. Handle SCALE ───────────────────────────────────────────────────────
	if curR, newR, ok := HandleKubernetesWorkloadScaleAction(actions); ok {
		if state.CurrentReplicas.IsNull() {
			state.CurrentReplicas = types.Int64Value(curR)
		}
		state.NewReplicas = types.Int64Value(newR)
	}

	// ── 5. Handle RESIZE ──────────────────────────────────────────────────────
	containerMap, err := HandleKubernetesWorkloadResizeAction(actions)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("error processing RESIZE action: %s", err.Error()))
		resp.Diagnostics.AddWarning("error processing RESIZE action", err.Error())
	}

	// ── 6. Build containers list ──────────────────────────────────────────────
	state.Containers = buildContainersList(containerMap, state.DefaultContainers)

	// ── 7. Apply fallback chain ───────────────────────────────────────────────
	setDefaultsKubernetesWorkloadToCurrentState(&state)
	setDefaultsKubernetesWorkloadToNewState(&state)

	if err := TagEntity(d.client, entity.UUID); err != nil {
		resp.Diagnostics.AddWarning("error while tagging an entity", err.Error())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// buildContainersList converts the containerResources map (from RESIZE actions) into a
// types.List using default_containers as fallback for any absent values.
func buildContainersList(containerMap map[string]*containerResources, defaults []KubernetesWorkloadDefaultContainerModel) types.List {
	// Build a name-indexed map of defaults
	defMap := make(map[string]KubernetesWorkloadDefaultContainerModel, len(defaults))
	for _, d := range defaults {
		defMap[d.Name.ValueString()] = d
	}

	// Collect sorted container names (recommendation takes priority; union with defaults)
	nameSet := make(map[string]struct{})
	for n := range containerMap {
		nameSet[n] = struct{}{}
	}
	for n := range defMap {
		nameSet[n] = struct{}{}
	}

	names := make([]string, 0, len(nameSet))
	for n := range nameSet {
		names = append(names, n)
	}
	sort.Strings(names)

	elems := make([]attr.Value, 0, len(names))
	for _, n := range names {
		cr := &containerResources{}
		if v, ok := containerMap[n]; ok {
			cr = v
		}
		def, hasDef := defMap[n]

		m := containerToTF(n, cr)

		// Apply defaults where recommendation is empty
		if hasDef {
			if m.NewCPURequest.ValueString() == "" {
				m.NewCPURequest = applyDefaultIfEmptyGeneric(m.NewCPURequest, def.DefaultCPURequest)
			}
			if m.NewCPULimit.ValueString() == "" {
				m.NewCPULimit = applyDefaultIfEmptyGeneric(m.NewCPULimit, def.DefaultCPULimit)
			}
			if m.NewMemoryRequest.ValueString() == "" {
				m.NewMemoryRequest = applyDefaultIfEmptyGeneric(m.NewMemoryRequest, def.DefaultMemoryRequest)
			}
			if m.NewMemoryLimit.ValueString() == "" {
				m.NewMemoryLimit = applyDefaultIfEmptyGeneric(m.NewMemoryLimit, def.DefaultMemoryLimit)
			}
		}

		objVal, _ := types.ObjectValueFrom(context.Background(), kubernetesWorkloadContainerAttrTypes, m)
		elems = append(elems, objVal)
	}

	listVal, _ := types.ListValue(
		types.ObjectType{AttrTypes: kubernetesWorkloadContainerAttrTypes},
		elems,
	)
	return listVal
}

func setDefaultsKubernetesWorkloadToCurrentState(state *KubernetesWorkloadEntityModel) {
	state.CurrentReplicas = applyDefaultIfEmptyGeneric(state.CurrentReplicas, state.DefaultReplicas)
}

func setDefaultsKubernetesWorkloadToNewState(state *KubernetesWorkloadEntityModel) {
	state.NewReplicas = applyDefaultIfEmptyGeneric(state.NewReplicas, state.DefaultReplicas)
	if state.Containers.IsNull() || state.Containers.IsUnknown() {
		state.Containers = buildContainersList(nil, state.DefaultContainers)
	}
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
	if state.CurrentReplicas.IsNull() {
		state.CurrentReplicas = types.Int64Value(0)
	}
}

func setCurrentKubernetesWorkloadToNewState(state *KubernetesWorkloadEntityModel) {
	state.NewReplicas = applyDefaultIfEmptyGeneric(state.NewReplicas, state.CurrentReplicas)
}
