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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	v2 "github.com/IBM/turbonomic-go-client/v2"
)

// boolDefaultFalse is a plan modifier that defaults a bool attribute to false
// when it is null (not set in config), so plan = state after creation.
type boolDefaultFalse struct{}

func (boolDefaultFalse) Description(_ context.Context) string {
	return "Defaults to false when not set."
}
func (boolDefaultFalse) MarkdownDescription(_ context.Context) string {
	return "Defaults to `false` when not set."
}
func (boolDefaultFalse) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.PlanValue.IsNull() {
		resp.PlanValue = types.BoolValue(false)
	}
}

var (
	_ resource.Resource                = &settingsPolicyResource{}
	_ resource.ResourceWithConfigure   = &settingsPolicyResource{}
	_ resource.ResourceWithImportState = &settingsPolicyResource{}
)

// NewSettingsPolicyResource constructs a new turbonomic_settings_policy resource.
func NewSettingsPolicyResource() resource.Resource {
	return &settingsPolicyResource{}
}

type settingsPolicyResource struct {
	v2Client *v2.Client
}

// settingModel holds one setting uuid+value pair.
type settingModel struct {
	UUID  types.String `tfsdk:"uuid"`
	Value types.String `tfsdk:"value"`
}

// settingsManagerModel holds one settings manager block.
type settingsManagerModel struct {
	Category types.String `tfsdk:"category"`
	Settings types.List   `tfsdk:"settings"`
}

// settingsPolicyResourceModel is the full Terraform state model.
type settingsPolicyResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	EntityType       types.String `tfsdk:"entity_type"`
	Disabled         types.Bool   `tfsdk:"disabled"`
	ScopeUUIDs       types.List   `tfsdk:"scope_uuids"`
	Note             types.String `tfsdk:"note"`
	ScheduleUUID     types.String `tfsdk:"schedule_uuid"`
	SettingsManagers types.List   `tfsdk:"settings_managers"`
	UUID             types.String `tfsdk:"uuid"`
	ReadOnly         types.Bool   `tfsdk:"read_only"`
	Default          types.Bool   `tfsdk:"default"`
}

// settingItemDTO uses json.RawMessage for Value because the API uses plain scalars
// ("RECOMMEND", 90.0, true) not wrapped objects, despite the generated Go DTO type.
type settingItemDTO struct {
	UUID  string          `json:"uuid"`
	Value json.RawMessage `json:"value,omitempty"`
}

// settingsManagerDTO - "uuid" is what Terraform calls "category".
type settingsManagerDTO struct {
	UUID     string           `json:"uuid"`
	Settings []settingItemDTO `json:"settings"`
}

type settingsScopeDTO struct {
	UUID string `json:"uuid"`
}

type settingsScheduleDTO struct {
	UUID string `json:"uuid"`
}

type settingsPolicyDTO struct {
	UUID             *string              `json:"uuid,omitempty"`
	DisplayName      *string              `json:"displayName,omitempty"`
	EntityType       *string              `json:"entityType,omitempty"`
	Disabled         *bool                `json:"disabled,omitempty"`
	Note             *string              `json:"note,omitempty"`
	Schedule         *settingsScheduleDTO `json:"schedule,omitempty"`
	Scopes           []settingsScopeDTO   `json:"scopes,omitempty"`
	SettingsManagers []settingsManagerDTO `json:"settingsManagers,omitempty"`
	ReadOnly         *bool                `json:"readOnly,omitempty"`
	Default          *bool                `json:"default,omitempty"`
}

var settingAttrTypes = map[string]attr.Type{
	"uuid":  types.StringType,
	"value": types.StringType,
}

var settingObjType = types.ObjectType{AttrTypes: settingAttrTypes}

var managerAttrTypes = map[string]attr.Type{
	"category": types.StringType,
	"settings": types.ListType{ElemType: settingObjType},
}

var managerObjType = types.ObjectType{AttrTypes: managerAttrTypes}

func (r *settingsPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings_policy"
}

func (r *settingsPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Turbonomic settings (automation) policy. Settings policies control action automation modes, scaling limits, and utilisation thresholds for a scoped set of entity groups.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal Terraform identifier, equal to the policy UUID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Display name of the settings policy.",
				Required:    true,
			},
			"entity_type": schema.StringAttribute{
				Description: "Entity type this policy targets. Examples: VirtualMachine, PhysicalMachine, Storage, Container.",
				Required:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "Set to true to disable the policy. Defaults to false when omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolDefaultFalse{},
				},
			},
			"scope_uuids": schema.ListAttribute{
				Description: "UUIDs of the groups this policy is scoped to. When omitted or empty the policy applies globally.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"note": schema.StringAttribute{
				Description: "Free-text note attached to the policy.",
				Optional:    true,
			},
			"schedule_uuid": schema.StringAttribute{
				Description: "UUID of an existing schedule to attach to this policy.",
				Optional:    true,
			},
			"uuid": schema.StringAttribute{
				Description: "UUID assigned by Turbonomic after creation.",
				Computed:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "True when the policy is system-managed and cannot be deleted or modified.",
				Computed:    true,
			},
			"default": schema.BoolAttribute{
				Description: "True when this is a default policy.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"settings_managers": schema.ListNestedBlock{
				Description: "List of settings manager blocks. Each block groups settings by manager category.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"category": schema.StringAttribute{
							Description: "Settings manager category UUID. Known values: automationmanager, marketsettingsmanager, controlmanager, storagesettingsmanager, entityprioritiesmanager, hcisettingsmanager, busappsettingsmanager, appsrvsettingsmanager.",
							Required:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"settings": schema.ListNestedBlock{
							Description: "Individual settings to override within this manager.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"uuid": schema.StringAttribute{
										Description: "Setting UUID (e.g. resizeTargetUtilizationVmem, moveVirtualMachine).",
										Required:    true,
									},
									"value": schema.StringAttribute{
										Description: "Setting value as a string. Enum settings: RECOMMEND, AUTOMATIC, MANUAL, DISABLED. Numeric: e.g. 90.0. Boolean: true or false.",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *settingsPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected resource configure type",
			fmt.Sprintf("Expected *providerData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.v2Client = data.V2Client
	if r.v2Client == nil {
		resp.Diagnostics.AddError("v2 client not available", "The v2 client is required for settings policy operations but is not available.")
		return
	}
}

func (r *settingsPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan settingsPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := r.buildDTO(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating settings policy", map[string]interface{}{"name": plan.Name.ValueString()})

	body, err := policyHTTPPost(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/settingspolicies", dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Settings Policy", err.Error())
		return
	}

	var created settingsPolicyDTO
	if err := json.Unmarshal(body, &created); err != nil {
		resp.Diagnostics.AddError("Error Parsing Create Response", err.Error())
		return
	}
	if created.UUID == nil {
		resp.Diagnostics.AddError("Error Creating Settings Policy", "API response did not include a UUID.")
		return
	}

	plan.UUID = types.StringPointerValue(created.UUID)
	plan.ID = types.StringPointerValue(created.UUID)

	// Re-fetch for full authoritative state (computed fields, read_only, default).
	r.readIntoModel(ctx, *created.UUID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *settingsPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state settingsPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		uuid = state.ID.ValueString()
	}

	body, statusCode, err := policyHTTPGet(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/settingspolicies/"+uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Settings Policy", err.Error())
		return
	}
	if statusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	var dto settingsPolicyDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		resp.Diagnostics.AddError("Error Parsing Settings Policy Response", err.Error())
		return
	}

	r.mapDTOToModel(ctx, &dto, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *settingsPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan settingsPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state settingsPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	plan.UUID = state.UUID
	plan.ID = state.ID

	dto := r.buildDTO(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	dto.UUID = &uuid

	tflog.Debug(ctx, "updating settings policy", map[string]interface{}{"uuid": uuid})

	// PUT with reset_defaults=false - always pass this to preserve user-managed values.
	if _, err := policyHTTPPut(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/settingspolicies/"+uuid+"?reset_defaults=false", dto); err != nil {
		resp.Diagnostics.AddError("Error Updating Settings Policy", err.Error())
		return
	}

	// Re-fetch - PUT response may be partial.
	r.readIntoModel(ctx, uuid, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *settingsPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state settingsPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()
	tflog.Debug(ctx, "deleting settings policy", map[string]interface{}{"uuid": uuid})
	if err := policyHTTPDelete(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/settingspolicies/"+uuid); err != nil {
		resp.Diagnostics.AddError("Error Deleting Settings Policy", err.Error())
	}
}

func (r *settingsPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *settingsPolicyResource) readIntoModel(ctx context.Context, uuid string, model *settingsPolicyResourceModel, diags *diag.Diagnostics) {
	body, statusCode, err := policyHTTPGet(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/settingspolicies/"+uuid)
	if err != nil {
		diags.AddError("Error Reading Settings Policy", err.Error())
		return
	}
	if statusCode == http.StatusNotFound {
		model.UUID = types.StringNull()
		return
	}
	var dto settingsPolicyDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		diags.AddError("Error Parsing Settings Policy Response", err.Error())
		return
	}
	r.mapDTOToModel(ctx, &dto, model, diags)
}

func (r *settingsPolicyResource) buildDTO(ctx context.Context, plan settingsPolicyResourceModel, diags *diag.Diagnostics) settingsPolicyDTO {
	dto := settingsPolicyDTO{}

	name := plan.Name.ValueString()
	dto.DisplayName = &name
	et := plan.EntityType.ValueString()
	dto.EntityType = &et

	setIfKnown(&dto.Disabled, plan.Disabled, types.Bool.ValueBool)
	setIfKnown(&dto.Note, plan.Note, types.String.ValueString)
	if !plan.ScheduleUUID.IsNull() && !plan.ScheduleUUID.IsUnknown() {
		dto.Schedule = &settingsScheduleDTO{UUID: plan.ScheduleUUID.ValueString()}
	}

	if !plan.ScopeUUIDs.IsNull() && !plan.ScopeUUIDs.IsUnknown() {
		var uuids []string
		diags.Append(plan.ScopeUUIDs.ElementsAs(ctx, &uuids, false)...)
		scopes := make([]settingsScopeDTO, 0, len(uuids))
		for _, u := range uuids {
			scopes = append(scopes, settingsScopeDTO{UUID: u})
		}
		dto.Scopes = scopes
	}

	if !plan.SettingsManagers.IsNull() && !plan.SettingsManagers.IsUnknown() {
		var managerModels []settingsManagerModel
		diags.Append(plan.SettingsManagers.ElementsAs(ctx, &managerModels, false)...)
		if !diags.HasError() {
			managers := make([]settingsManagerDTO, 0, len(managerModels))
			for _, m := range managerModels {
				var settingModels []settingModel
				diags.Append(m.Settings.ElementsAs(ctx, &settingModels, false)...)

				items := make([]settingItemDTO, 0, len(settingModels))
				for _, s := range settingModels {
					if s.Value.IsNull() || s.Value.IsUnknown() {
						continue
					}
					items = append(items, settingItemDTO{
						UUID:  s.UUID.ValueString(),
						Value: stringToRawValue(s.Value.ValueString()),
					})
				}
				// Only include managers with at least one valued setting - mirrors UI behaviour.
				if len(items) > 0 {
					managers = append(managers, settingsManagerDTO{
						UUID:     m.Category.ValueString(),
						Settings: items,
					})
				}
			}
			dto.SettingsManagers = managers
		}
	}

	return dto
}

func (r *settingsPolicyResource) mapDTOToModel(ctx context.Context, dto *settingsPolicyDTO, model *settingsPolicyResourceModel, diags *diag.Diagnostics) {
	if dto.UUID != nil {
		model.UUID = types.StringPointerValue(dto.UUID)
		model.ID = types.StringPointerValue(dto.UUID)
	}
	if dto.DisplayName != nil {
		model.Name = types.StringPointerValue(dto.DisplayName)
	}
	if dto.EntityType != nil {
		model.EntityType = types.StringPointerValue(dto.EntityType)
	}
	if dto.Disabled != nil {
		model.Disabled = types.BoolPointerValue(dto.Disabled)
	} else {
		// The API omits `disabled` when the policy is active. Default to false rather than
		// null so Terraform does not see a plan→state mismatch after create/update.
		model.Disabled = types.BoolValue(false)
	}
	if dto.Note != nil {
		model.Note = types.StringPointerValue(dto.Note)
		// else: The API does not echo `note` back on read. Preserve the plan value to avoid
		// "was cty.StringVal(…), but now null" inconsistent-result errors.
		// model.Note is already set to the plan value at this point - do not overwrite it.
	}
	if dto.Schedule != nil {
		model.ScheduleUUID = types.StringValue(dto.Schedule.UUID)
	} else {
		model.ScheduleUUID = types.StringNull()
	}
	if dto.ReadOnly != nil {
		model.ReadOnly = types.BoolPointerValue(dto.ReadOnly)
	} else {
		model.ReadOnly = types.BoolNull()
	}
	if dto.Default != nil {
		model.Default = types.BoolPointerValue(dto.Default)
	} else {
		model.Default = types.BoolNull()
	}

	// scope_uuids - extract UUID from each scope object.
	// Preserve an empty list when the plan had [] so we don't flip null→[] or []→null.
	if len(dto.Scopes) > 0 {
		elems := make([]attr.Value, 0, len(dto.Scopes))
		for _, s := range dto.Scopes {
			elems = append(elems, types.StringValue(s.UUID))
		}
		var d diag.Diagnostics
		model.ScopeUUIDs, d = types.ListValue(types.StringType, elems)
		diags.Append(d...)
	} else if !model.ScopeUUIDs.IsNull() && !model.ScopeUUIDs.IsUnknown() {
		// Plan had an explicit empty list - keep it as empty list, not null.
		model.ScopeUUIDs, _ = types.ListValue(types.StringType, []attr.Value{})
	} else {
		model.ScopeUUIDs = types.ListNull(types.StringType)
	}

	// settings_managers - only include managers/settings with a non-empty value.
	// Both the managers list and settings within each manager are reordered to match
	// the plan/state order so list indices never change after a round-trip through the API.
	if len(dto.SettingsManagers) > 0 {
		// Build plan order maps from the existing model (populated from plan before this call).
		type settingEntry struct {
			uuid  string
			value string
		}
		planManagerOrder := map[string]int{}            // managerCategory → index
		planSettingOrder := map[string]map[string]int{} // managerCategory → settingUUID → index
		if !model.SettingsManagers.IsNull() && !model.SettingsManagers.IsUnknown() {
			var existingManagers []settingsManagerModel
			if d := model.SettingsManagers.ElementsAs(ctx, &existingManagers, false); !d.HasError() {
				for i, em := range existingManagers {
					cat := em.Category.ValueString()
					planManagerOrder[cat] = i
					planSettingOrder[cat] = map[string]int{}
					var existingSettings []settingModel
					if sd := em.Settings.ElementsAs(ctx, &existingSettings, false); !sd.HasError() {
						for j, es := range existingSettings {
							planSettingOrder[cat][es.UUID.ValueString()] = j
						}
					}
				}
			}
		}

		// Build a map of manager category → collected settings from the API response.
		type managerEntry struct {
			category string
			settings []settingEntry
		}
		apiManagers := map[string][]settingEntry{}
		apiManagerOrder := []string{} // original API order, for extras not in plan
		for _, m := range dto.SettingsManagers {
			apiManagerOrder = append(apiManagerOrder, m.UUID)
			for _, s := range m.Settings {
				valStr := rawValueToString(s.Value)
				if valStr == "" {
					continue
				}
				apiManagers[m.UUID] = append(apiManagers[m.UUID], settingEntry{uuid: s.UUID, value: valStr})
			}
		}

		// Produce managers in plan order, then any extras from the API not in plan.
		orderedManagers := make([]managerEntry, len(planManagerOrder))
		seen := map[string]bool{}
		for cat, idx := range planManagerOrder {
			orderedManagers[idx] = managerEntry{category: cat, settings: apiManagers[cat]}
			seen[cat] = true
		}
		var extraManagers []managerEntry
		for _, cat := range apiManagerOrder {
			if !seen[cat] {
				extraManagers = append(extraManagers, managerEntry{category: cat, settings: apiManagers[cat]})
			}
		}
		allManagers := append(orderedManagers, extraManagers...)

		managerElems := make([]attr.Value, 0, len(allManagers))
		for _, mgr := range allManagers {
			if mgr.category == "" {
				continue
			}
			collected := mgr.settings

			// Reorder settings within this manager to match the plan order.
			if orderMap, ok := planSettingOrder[mgr.category]; ok && len(orderMap) > 0 {
				known := make([]settingEntry, len(orderMap))
				var extras []settingEntry
				for _, e := range collected {
					if idx, found := orderMap[e.uuid]; found {
						known[idx] = e
					} else {
						extras = append(extras, e)
					}
				}
				var ordered []settingEntry
				for _, e := range known {
					if e.uuid != "" {
						ordered = append(ordered, e)
					}
				}
				collected = append(ordered, extras...)
			}

			settingElems := make([]attr.Value, 0, len(collected))
			for _, e := range collected {
				obj, d := types.ObjectValue(settingAttrTypes, map[string]attr.Value{
					"uuid":  types.StringValue(e.uuid),
					"value": types.StringValue(e.value),
				})
				diags.Append(d...)
				settingElems = append(settingElems, obj)
			}
			settingsList, d := types.ListValue(settingObjType, settingElems)
			diags.Append(d...)

			managerObj, d := types.ObjectValue(managerAttrTypes, map[string]attr.Value{
				"category": types.StringValue(mgr.category),
				"settings": settingsList,
			})
			diags.Append(d...)
			managerElems = append(managerElems, managerObj)
		}
		var d diag.Diagnostics
		model.SettingsManagers, d = types.ListValue(managerObjType, managerElems)
		diags.Append(d...)
	} else {
		model.SettingsManagers = types.ListNull(managerObjType)
	}
}
