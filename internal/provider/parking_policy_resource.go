// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var (
	_ resource.Resource                = &parkingPolicyResource{}
	_ resource.ResourceWithConfigure   = &parkingPolicyResource{}
	_ resource.ResourceWithImportState = &parkingPolicyResource{}
)

// NewParkingPolicyResource returns a new turbonomic_parking_policy resource instance.
func NewParkingPolicyResource() resource.Resource {
	return &parkingPolicyResource{}
}

type parkingPolicyResource struct {
	v2Client *v2.Client
}

// parkingPolicyResourceModel is the Terraform state model for turbonomic_parking_policy.
type parkingPolicyResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	UUID               types.String `tfsdk:"uuid"`
	DisplayName        types.String `tfsdk:"display_name"`
	Level              types.String `tfsdk:"level"`
	Priority           types.Int64  `tfsdk:"priority"`
	RestrictUnparkable types.Bool   `tfsdk:"restrict_unparkable"`
	AttachSchedule     types.Object `tfsdk:"attach_schedule"`
	CriteriaList       types.List   `tfsdk:"criteria_list"`
}

// parkingPolicyAttachScheduleModel maps the attach_schedule nested attribute.
type parkingPolicyAttachScheduleModel struct {
	TimespanScheduleUUID types.String `tfsdk:"timespan_schedule_uuid"`
	DisplayName          types.String `tfsdk:"display_name"`
}

// parkingPolicyAttachScheduleAttrTypes returns attr.Type map for attach_schedule.
func parkingPolicyAttachScheduleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"timespan_schedule_uuid": types.StringType,
		"display_name":           types.StringType,
	}
}

// parkingCriteriaModel maps a single criteria_list block.
type parkingCriteriaModel struct {
	FilterType    types.String `tfsdk:"filter_type"`
	ExpType       types.String `tfsdk:"exp_type"`
	ExpVal        types.String `tfsdk:"exp_val"`
	CaseSensitive types.Bool   `tfsdk:"case_sensitive"`
}

// parkingCriteriaAttrTypes returns attr.Type map for a single criteria item.
func parkingCriteriaAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"filter_type":    types.StringType,
		"exp_type":       types.StringType,
		"exp_val":        types.StringType,
		"case_sensitive": types.BoolType,
	}
}

// parkingPolicyWriteDTO is the wire DTO for create/update (ParkingPolicyApiDTO).
type parkingPolicyWriteDTO struct {
	DisplayName  string                    `json:"displayName"`
	Level        string                    `json:"level"`
	Priority     *int64                    `json:"priority,omitempty"`
	Impacts      parkingPolicyImpactsDTO   `json:"impacts"`
	CriteriaList []parkingCriteriaWriteDTO `json:"criteriaList,omitempty"`
}

// parkingPolicyImpactsDTO maps ParkingPolicyImpactsApiDTO.
type parkingPolicyImpactsDTO struct {
	RestrictUnparkable bool                      `json:"restrictUnparkable"`
	AttachSchedule     *parkingAttachScheduleDTO `json:"attachSchedule,omitempty"`
}

// parkingAttachScheduleDTO maps ParkingPolicyTimeSpansApiDTO.
type parkingAttachScheduleDTO struct {
	TimespanScheduleUUID string  `json:"timespanScheduleUuid"`
	DisplayName          *string `json:"displayName,omitempty"`
}

// parkingCriteriaWriteDTO maps FilterApiDTO for parking criteria.
type parkingCriteriaWriteDTO struct {
	FilterType    string `json:"filterType"`
	ExpType       string `json:"expType"`
	ExpVal        string `json:"expVal"`
	CaseSensitive bool   `json:"caseSensitive"`
}

// parkingPolicyReadDTO is the response DTO from GET/POST/PUT /parking/policies/{uuid}.
type parkingPolicyReadDTO struct {
	UUID         *string                  `json:"uuid,omitempty"`
	DisplayName  *string                  `json:"displayName,omitempty"`
	Level        *string                  `json:"level,omitempty"`
	Priority     *int64                   `json:"priority,omitempty"`
	Impacts      *parkingPolicyImpactsDTO `json:"impacts,omitempty"`
	CriteriaList []parkingCriteriaReadDTO `json:"criteriaList,omitempty"`
}

// parkingCriteriaReadDTO maps FilterApiDTO from GET response.
type parkingCriteriaReadDTO struct {
	FilterType    *string `json:"filterType,omitempty"`
	ExpType       *string `json:"expType,omitempty"`
	ExpVal        *string `json:"expVal,omitempty"`
	CaseSensitive *bool   `json:"caseSensitive,omitempty"`
}

func (r *parkingPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parking_policy"
}

func (r *parkingPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	criteriaSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"filter_type": schema.StringAttribute{
				Required:    true,
				Description: "Internal filter type name (e.g. vmsByName, vmsByTag). See Turbonomic documentation for valid values.",
			},
			"exp_type": schema.StringAttribute{
				Required:    true,
				Description: "Comparison operator: EQ, NEQ, RXEQ (regex match), RXNEQ (regex non-match), GT, LT, GTE, LTE, EX, NEX.",
				Validators: []validator.String{
					stringvalidator.OneOf("EQ", "NEQ", "GT", "LT", "GTE", "LTE", "RXEQ", "RXNEQ", "EX", "NEX"),
				},
			},
			"exp_val": schema.StringAttribute{
				Required:    true,
				Description: "The value or regex expression to match against.",
			},
			"case_sensitive": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the regex match is case-sensitive.",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages a Turbonomic parking policy. " +
			"Parking policies define rules for automatically stopping (parking) and starting cloud workloads " +
			"on a schedule to reduce costs. " +
			"Reference a timespan schedule UUID from the turbonomic_timespan data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal Terraform identifier, equal to the parking policy UUID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Description: "UUID assigned by Turbonomic after creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Human-readable name of the parking policy.",
				Required:    true,
			},
			"level": schema.StringAttribute{
				Description: "Scope level at which the policy applies: GLOBAL, CLOUD_PROVIDER, ACCOUNT, RESOURCE_GROUP, DATACENTER, or TARGET.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("GLOBAL", "CLOUD_PROVIDER", "ACCOUNT", "RESOURCE_GROUP", "DATACENTER", "TARGET"),
				},
			},
			"priority": schema.Int64Attribute{
				Description: "Priority for conflict resolution when multiple policies apply to the same entity. Lower number = higher priority.",
				Optional:    true,
				Computed:    true,
			},
			"restrict_unparkable": schema.BoolAttribute{
				Description: "When true, entities matched by this policy cannot be manually unparked outside the schedule window.",
				Optional:    true,
				Computed:    true,
			},
			"attach_schedule": schema.SingleNestedAttribute{
				Description: "Timespan schedule to attach to this parking policy. " +
					"Use the turbonomic_timespan data source to look up the UUID.",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"timespan_schedule_uuid": schema.StringAttribute{
						Required:    true,
						Description: "UUID of the timespan schedule (from turbonomic_timespan data source).",
					},
					"display_name": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Display name of the timespan schedule (populated from API on read).",
					},
				},
			},
			"criteria_list": schema.ListNestedAttribute{
				Description:  "List of filters to select the entities this parking policy applies to.",
				Optional:     true,
				Computed:     true,
				NestedObject: criteriaSchema,
			},
		},
	}
}

func (r *parkingPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		resp.Diagnostics.AddError("v2 client not available", "The v2 client is required for parking policy operations but is not available.")
	}
}

func (r *parkingPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan parkingPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto, dtoErr := r.buildWriteDTO(ctx, plan)
	if dtoErr != nil {
		resp.Diagnostics.AddError("Error Building Parking Policy", dtoErr.Error())
		return
	}

	tflog.Debug(ctx, "creating parking policy", map[string]interface{}{"display_name": plan.DisplayName.ValueString()})

	body, err := policyHTTPPost(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/parking/policies", dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Parking Policy", err.Error())
		return
	}

	var created parkingPolicyReadDTO
	if err := json.Unmarshal(body, &created); err != nil {
		resp.Diagnostics.AddError("Error Parsing Create Response", err.Error())
		return
	}
	if created.UUID == nil {
		resp.Diagnostics.AddError("Error Creating Parking Policy", "API response did not include a UUID.")
		return
	}

	r.mapDTOToModel(ctx, &created, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *parkingPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state parkingPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		uuid = state.ID.ValueString()
	}

	body, statusCode, err := policyHTTPGet(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/parking/policies/"+uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Parking Policy", err.Error())
		return
	}
	if statusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	var dto parkingPolicyReadDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		resp.Diagnostics.AddError("Error Parsing Parking Policy Response", err.Error())
		return
	}

	r.mapDTOToModel(ctx, &dto, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *parkingPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan parkingPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state parkingPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.UUID = state.UUID
	plan.ID = state.ID
	uuid := state.UUID.ValueString()

	dto, dtoErr := r.buildWriteDTO(ctx, plan)
	if dtoErr != nil {
		resp.Diagnostics.AddError("Error Building Parking Policy", dtoErr.Error())
		return
	}

	tflog.Debug(ctx, "updating parking policy", map[string]interface{}{"uuid": uuid})

	body, err := policyHTTPPut(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/parking/policies/"+uuid, dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Parking Policy", err.Error())
		return
	}

	var updated parkingPolicyReadDTO
	if err := json.Unmarshal(body, &updated); err != nil {
		resp.Diagnostics.AddError("Error Parsing Update Response", err.Error())
		return
	}

	r.mapDTOToModel(ctx, &updated, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *parkingPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state parkingPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()
	tflog.Debug(ctx, "deleting parking policy", map[string]interface{}{"uuid": uuid})
	if err := policyHTTPDelete(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/parking/policies/"+uuid); err != nil {
		resp.Diagnostics.AddError("Error Deleting Parking Policy", err.Error())
	}
}

func (r *parkingPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *parkingPolicyResource) buildWriteDTO(ctx context.Context, plan parkingPolicyResourceModel) (parkingPolicyWriteDTO, error) {
	dto := parkingPolicyWriteDTO{
		DisplayName: plan.DisplayName.ValueString(),
		Level:       plan.Level.ValueString(),
	}

	setIfKnown(&dto.Priority, plan.Priority, types.Int64.ValueInt64)

	dto.Impacts.RestrictUnparkable = plan.RestrictUnparkable.ValueBool()

	if !plan.AttachSchedule.IsNull() && !plan.AttachSchedule.IsUnknown() {
		var schedModel parkingPolicyAttachScheduleModel
		diags := plan.AttachSchedule.As(ctx, &schedModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return dto, fmt.Errorf("error reading attach_schedule: %s", diagsToString(diags))
		}
		as := &parkingAttachScheduleDTO{
			TimespanScheduleUUID: schedModel.TimespanScheduleUUID.ValueString(),
		}
		setIfKnown(&as.DisplayName, schedModel.DisplayName, types.String.ValueString)
		dto.Impacts.AttachSchedule = as
	}

	if !plan.CriteriaList.IsNull() && !plan.CriteriaList.IsUnknown() {
		var criteria []parkingCriteriaModel
		d := plan.CriteriaList.ElementsAs(ctx, &criteria, false)
		if d.HasError() {
			return dto, fmt.Errorf("error reading criteria_list: %s", diagsToString(d))
		}
		for _, c := range criteria {
			dto.CriteriaList = append(dto.CriteriaList, parkingCriteriaWriteDTO{
				FilterType:    c.FilterType.ValueString(),
				ExpType:       c.ExpType.ValueString(),
				ExpVal:        c.ExpVal.ValueString(),
				CaseSensitive: c.CaseSensitive.ValueBool(),
			})
		}
	}

	return dto, nil
}

func (r *parkingPolicyResource) mapDTOToModel(ctx context.Context, dto *parkingPolicyReadDTO, model *parkingPolicyResourceModel, diags *diag.Diagnostics) {
	if dto.UUID != nil {
		model.UUID = types.StringPointerValue(dto.UUID)
		model.ID = types.StringPointerValue(dto.UUID)
	}
	if dto.DisplayName != nil {
		model.DisplayName = types.StringPointerValue(dto.DisplayName)
	}
	if dto.Level != nil {
		model.Level = types.StringPointerValue(dto.Level)
	}
	if dto.Priority != nil {
		model.Priority = types.Int64PointerValue(dto.Priority)
	} else {
		model.Priority = types.Int64Null()
	}

	if dto.Impacts != nil {
		model.RestrictUnparkable = types.BoolValue(dto.Impacts.RestrictUnparkable)

		if dto.Impacts.AttachSchedule != nil {
			as := dto.Impacts.AttachSchedule
			displayName := types.StringNull()
			if as.DisplayName != nil {
				displayName = types.StringPointerValue(as.DisplayName)
			}
			schedObj, d := types.ObjectValue(parkingPolicyAttachScheduleAttrTypes(), map[string]attr.Value{
				"timespan_schedule_uuid": types.StringValue(as.TimespanScheduleUUID),
				"display_name":           displayName,
			})
			diags.Append(d...)
			if !d.HasError() {
				model.AttachSchedule = schedObj
			}
		} else {
			model.AttachSchedule = types.ObjectNull(parkingPolicyAttachScheduleAttrTypes())
		}
	} else {
		model.RestrictUnparkable = types.BoolValue(false)
		model.AttachSchedule = types.ObjectNull(parkingPolicyAttachScheduleAttrTypes())
	}

	criteriaObjType := types.ObjectType{AttrTypes: parkingCriteriaAttrTypes()}
	elems := make([]attr.Value, 0, len(dto.CriteriaList))
	for _, c := range dto.CriteriaList {
		filterType := types.StringNull()
		if c.FilterType != nil {
			filterType = types.StringPointerValue(c.FilterType)
		}
		expType := types.StringNull()
		if c.ExpType != nil {
			expType = types.StringPointerValue(c.ExpType)
		}
		expVal := types.StringNull()
		if c.ExpVal != nil {
			expVal = types.StringPointerValue(c.ExpVal)
		}
		caseSens := types.BoolValue(false)
		if c.CaseSensitive != nil {
			caseSens = types.BoolPointerValue(c.CaseSensitive)
		}
		obj, d := types.ObjectValue(parkingCriteriaAttrTypes(), map[string]attr.Value{
			"filter_type":    filterType,
			"exp_type":       expType,
			"exp_val":        expVal,
			"case_sensitive": caseSens,
		})
		diags.Append(d...)
		if d.HasError() {
			return
		}
		elems = append(elems, obj)
	}
	criteriaList, d := types.ListValue(criteriaObjType, elems)
	diags.Append(d...)
	if !d.HasError() {
		model.CriteriaList = criteriaList
	}
}

func diagsToString(diags diag.Diagnostics) string {
	if len(diags) == 0 {
		return ""
	}
	msg := ""
	for _, d := range diags {
		msg += d.Summary() + ": " + d.Detail() + "; "
	}
	return msg
}
