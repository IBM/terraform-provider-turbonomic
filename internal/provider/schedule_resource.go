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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &scheduleResource{}
	_ resource.ResourceWithConfigure   = &scheduleResource{}
	_ resource.ResourceWithImportState = &scheduleResource{}
)

// NewScheduleResource returns a new turbonomic_schedule resource instance.
func NewScheduleResource() resource.Resource {
	return &scheduleResource{}
}

type scheduleResource struct {
	v2Client *v2.Client
}

// scheduleResourceModel is the Terraform state model for turbonomic_schedule.
type scheduleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	UUID        types.String `tfsdk:"uuid"`
	DisplayName types.String `tfsdk:"display_name"`
	StartDate   types.String `tfsdk:"start_date"`
	StartTime   types.String `tfsdk:"start_time"`
	EndTime     types.String `tfsdk:"end_time"`
	EndDate     types.String `tfsdk:"end_date"`
	TimeZone    types.String `tfsdk:"time_zone"`
	Recurrence  types.Object `tfsdk:"recurrence"`

	// Computed-only
	NextOccurrence          types.String `tfsdk:"next_occurrence"`
	NextOccurrenceTimestamp types.Int64  `tfsdk:"next_occurrence_timestamp"`
	RemainingTimeActiveMs   types.Int64  `tfsdk:"remaining_time_active_ms"`
}

// scheduleRecurrenceModel maps the recurrence nested attribute for state decoding.
type scheduleRecurrenceModel struct {
	Type           types.String `tfsdk:"type"`
	Interval       types.Int64  `tfsdk:"interval"`
	DaysOfWeek     types.List   `tfsdk:"days_of_week"`
	DaysOfMonth    types.List   `tfsdk:"days_of_month"`
	WeekOfTheMonth types.List   `tfsdk:"week_of_the_month"`
}

// scheduleInputDTO is the write-side DTO (omits computed/read-only fields).
type scheduleInputDTO struct {
	DisplayName string         `json:"displayName"`
	StartDate   *string        `json:"startDate,omitempty"`
	StartTime   string         `json:"startTime"`
	EndTime     string         `json:"endTime"`
	EndDate     *string        `json:"endDate,omitempty"`
	TimeZone    *string        `json:"timeZone,omitempty"`
	Recurrence  *recurrenceDTO `json:"recurrence,omitempty"`
}

func (r *scheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

func (r *scheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Turbonomic action execution schedule. " +
			"Schedules restrict when automated actions can run. " +
			"Reference a schedule's UUID in turbonomic_settings_policy via schedule_uuid.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal Terraform identifier, equal to the schedule UUID.",
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
				Description: "Human-readable name of the schedule.",
				Required:    true,
			},
			"start_date": schema.StringAttribute{
				Description: "Start date/time of the schedule window in ISO 8601 local time (YYYY-MM-DDTHH:MM).",
				Required:    true,
			},
			"start_time": schema.StringAttribute{
				Description: "Start time of the daily window in ISO 8601 format (e.g. 2000-01-01T22:00:00).",
				Required:    true,
			},
			"end_time": schema.StringAttribute{
				Description: "End time of the daily window in ISO 8601 format (e.g. 2000-01-02T06:00:00).",
				Required:    true,
			},
			"end_date": schema.StringAttribute{
				Description: "End date after which the schedule no longer recurs (ISO 8601 local time). Omit for open-ended schedules.",
				Optional:    true,
			},
			"time_zone": schema.StringAttribute{
				Description: "IANA timezone name (e.g. America/New_York, UTC). Defaults to UTC when omitted.",
				Optional:    true,
			},
			"recurrence": schema.SingleNestedAttribute{
				Description: "Recurrence rule for repeating schedules. Omit for a one-time schedule.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Recurrence type: DAILY, WEEKLY, or MONTHLY.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("DAILY", "WEEKLY", "MONTHLY"),
						},
					},
					"interval": schema.Int64Attribute{
						Description: "Repeat every N days/weeks/months.",
						Optional:    true,
					},
					"days_of_week": schema.ListAttribute{
						Description: "Days of the week for WEEKLY recurrence. Values: Mon, Tue, Wed, Thu, Fri, Sat, Sun.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"days_of_month": schema.ListAttribute{
						Description: "Days of the month for MONTHLY recurrence (e.g. [1, 15]).",
						Optional:    true,
						ElementType: types.Int64Type,
					},
					"week_of_the_month": schema.ListAttribute{
						Description: "Which week(s) of the month for MONTHLY recurrence. -1=last, 1=first, etc.",
						Optional:    true,
						ElementType: types.Int64Type,
					},
				},
			},
			"next_occurrence": schema.StringAttribute{
				Description: "ISO 8601 datetime of the next trigger. Computed by Turbonomic.",
				Computed:    true,
			},
			"next_occurrence_timestamp": schema.Int64Attribute{
				Description: "Unix millisecond timestamp of the next trigger. Computed by Turbonomic.",
				Computed:    true,
			},
			"remaining_time_active_ms": schema.Int64Attribute{
				Description: "Milliseconds remaining in the current active window. Zero when not active.",
				Computed:    true,
			},
		},
	}
}

func (r *scheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		resp.Diagnostics.AddError("v2 client not available", "The v2 client is required for schedule operations but is not available.")
	}
}

func (r *scheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := r.buildInputDTO(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating schedule", map[string]interface{}{"display_name": plan.DisplayName.ValueString()})

	body, err := policyHTTPPost(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/schedules", dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Schedule", err.Error())
		return
	}

	var created scheduleDTO
	if err := json.Unmarshal(body, &created); err != nil {
		resp.Diagnostics.AddError("Error Parsing Create Response", err.Error())
		return
	}
	if created.Uuid == nil {
		resp.Diagnostics.AddError("Error Creating Schedule", "API response did not include a UUID.")
		return
	}

	plan.UUID = types.StringPointerValue(created.Uuid)
	plan.ID = types.StringPointerValue(created.Uuid)

	r.mapDTOToModel(ctx, &created, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *scheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		uuid = state.ID.ValueString()
	}

	body, statusCode, err := policyHTTPGet(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/schedules/"+uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Schedule", err.Error())
		return
	}
	if statusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	var dto scheduleDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		resp.Diagnostics.AddError("Error Parsing Schedule Response", err.Error())
		return
	}

	r.mapDTOToModel(ctx, &dto, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *scheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state scheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	plan.UUID = state.UUID
	plan.ID = state.ID

	dto := r.buildInputDTO(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating schedule", map[string]interface{}{"uuid": uuid})

	body, err := policyHTTPPut(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/schedules/"+uuid, dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Schedule", err.Error())
		return
	}

	var updated scheduleDTO
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

func (r *scheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()
	tflog.Debug(ctx, "deleting schedule", map[string]interface{}{"uuid": uuid})
	if err := policyHTTPDelete(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/schedules/"+uuid); err != nil {
		resp.Diagnostics.AddError("Error Deleting Schedule", err.Error())
	}
}

func (r *scheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// truncateToMinutes strips trailing seconds (":ss") from a datetime string like
// "2000-01-01T22:00:00" → "2000-01-01T22:00". The Turbonomic API rejects any
// startTime/endTime value that includes seconds.
func truncateToMinutes(s string) string {
	// DateTime format the API accepts: "YYYY-MM-DDTHH:MM"  (16 chars)
	if len(s) > 16 {
		return s[:16]
	}
	return s
}

func (r *scheduleResource) buildInputDTO(ctx context.Context, plan scheduleResourceModel, diags *diag.Diagnostics) scheduleInputDTO {
	isOneTime := plan.Recurrence.IsNull() || plan.Recurrence.IsUnknown()

	// The Turbonomic API uses startTime as a full ISO datetime (YYYY-MM-DDTHH:MM).
	// For one-time schedules, start_date holds the occurrence datetime and is sent
	// as startTime. For recurring schedules, start_time holds the daily window time
	// and start_date is sent separately as startDate (the recurrence anchor).
	var startTime string
	if isOneTime {
		startTime = truncateToMinutes(plan.StartDate.ValueString())
	} else {
		startTime = truncateToMinutes(plan.StartTime.ValueString())
	}

	dto := scheduleInputDTO{
		DisplayName: plan.DisplayName.ValueString(),
		StartTime:   startTime,
		EndTime:     truncateToMinutes(plan.EndTime.ValueString()),
	}

	if !isOneTime {
		if !plan.StartDate.IsNull() && !plan.StartDate.IsUnknown() && plan.StartDate.ValueString() != "" {
			v := plan.StartDate.ValueString()
			dto.StartDate = &v
		}
	}
	setIfKnown(&dto.EndDate, plan.EndDate, types.String.ValueString)
	setIfKnown(&dto.TimeZone, plan.TimeZone, types.String.ValueString)
	if !plan.Recurrence.IsNull() && !plan.Recurrence.IsUnknown() {
		var rec scheduleRecurrenceModel
		diags.Append(plan.Recurrence.As(ctx, &rec, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			rdto := &recurrenceDTO{Type: rec.Type.ValueString()}
			if !rec.Interval.IsNull() && !rec.Interval.IsUnknown() {
				v := int32(rec.Interval.ValueInt64())
				rdto.Interval = &v
			}
			if !rec.DaysOfWeek.IsNull() && !rec.DaysOfWeek.IsUnknown() {
				var days []string
				diags.Append(rec.DaysOfWeek.ElementsAs(ctx, &days, false)...)
				rdto.DaysOfWeek = days
			}
			if !rec.DaysOfMonth.IsNull() && !rec.DaysOfMonth.IsUnknown() {
				var doms []int64
				diags.Append(rec.DaysOfMonth.ElementsAs(ctx, &doms, false)...)
				idoms := make([]int32, len(doms))
				for i, v := range doms {
					idoms[i] = int32(v)
				}
				rdto.DaysOfMonth = idoms
			}
			if !rec.WeekOfTheMonth.IsNull() && !rec.WeekOfTheMonth.IsUnknown() {
				var wotm []int64
				diags.Append(rec.WeekOfTheMonth.ElementsAs(ctx, &wotm, false)...)
				iwotm := make([]int32, len(wotm))
				for i, v := range wotm {
					iwotm[i] = int32(v)
				}
				rdto.WeekOfTheMonth = iwotm
			}
			dto.Recurrence = rdto
		}
	}
	return dto
}

func (r *scheduleResource) mapDTOToModel(ctx context.Context, dto *scheduleDTO, model *scheduleResourceModel, diags *diag.Diagnostics) {
	if dto.Uuid != nil {
		model.UUID = types.StringPointerValue(dto.Uuid)
		model.ID = types.StringPointerValue(dto.Uuid)
	}
	if dto.DisplayName != nil {
		model.DisplayName = types.StringPointerValue(dto.DisplayName)
	}
	// Preserve the plan values for start_date, start_time, end_time.
	// The API strips seconds from datetime strings (returns "YYYY-MM-DDTHH:MM") but
	// the user's config may supply seconds (e.g. "2000-01-01T22:00:00"). If we
	// overwrite with the API echo we get an inconsistent-result error on every apply.
	// Only update these if the model (plan) didn't already have a value set.
	if model.StartDate.IsNull() || model.StartDate.IsUnknown() {
		model.StartDate = types.StringValue(dto.StartDate)
	}
	if model.StartTime.IsNull() || model.StartTime.IsUnknown() {
		model.StartTime = types.StringValue(dto.StartTime)
	}
	if model.EndTime.IsNull() || model.EndTime.IsUnknown() {
		model.EndTime = types.StringValue(dto.EndTime)
	}

	if dto.EndDate != nil {
		model.EndDate = types.StringPointerValue(dto.EndDate)
	} else {
		model.EndDate = types.StringNull()
	}
	if dto.TimeZone != nil {
		model.TimeZone = types.StringPointerValue(dto.TimeZone)
	} else {
		model.TimeZone = types.StringNull()
	}
	if dto.NextOccurrence != nil {
		model.NextOccurrence = types.StringPointerValue(dto.NextOccurrence)
	} else {
		model.NextOccurrence = types.StringNull()
	}
	model.NextOccurrenceTimestamp = types.Int64Value(0)
	if dto.NextOccurrenceTimestamp != nil {
		model.NextOccurrenceTimestamp = types.Int64Value(*dto.NextOccurrenceTimestamp)
	}
	model.RemainingTimeActiveMs = types.Int64Value(0)
	if dto.RemaingTimeActiveInMs != nil {
		model.RemainingTimeActiveMs = types.Int64Value(*dto.RemaingTimeActiveInMs)
	}

	recObj, d := buildRecurrenceObject(dto.Recurrence)
	diags.Append(d...)
	if d.HasError() {
		return
	}
	if obj, ok := recObj.(types.Object); ok {
		model.Recurrence = obj
	} else {
		model.Recurrence = types.ObjectNull(recurrenceAttrTypes())
	}
}
