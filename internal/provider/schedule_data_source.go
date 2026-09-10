// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var _ datasource.DataSource = &scheduleDataSource{}
var _ datasource.DataSourceWithConfigure = &scheduleDataSource{}

// NewScheduleDataSource returns a new instance of the turbonomic_schedule data source.
func NewScheduleDataSource() datasource.DataSource {
	return &scheduleDataSource{}
}

type scheduleDataSource struct {
	v2Client *v2.Client
}

// scheduleDataSourceModel is the root Terraform state model.
type scheduleDataSourceModel struct {
	DisplayNameFilter types.String `tfsdk:"display_name"`
	Schedules         types.List   `tfsdk:"schedules"`
}

// scheduleDTO is a local wire DTO for JSON decoding.
type scheduleDTO struct {
	Uuid                    *string        `json:"uuid,omitempty"`
	DisplayName             *string        `json:"displayName,omitempty"`
	ClassName               *string        `json:"className,omitempty"`
	StartDate               string         `json:"startDate"`
	StartTime               string         `json:"startTime"`
	EndTime                 string         `json:"endTime"`
	EndDate                 *string        `json:"endDate,omitempty"`
	TimeZone                *string        `json:"timeZone,omitempty"`
	NextOccurrence          *string        `json:"nextOccurrence,omitempty"`
	NextOccurrenceTimestamp *int64         `json:"nextOccurrenceTimestamp,omitempty"`
	RemaingTimeActiveInMs   *int64         `json:"remaingTimeActiveInMs,omitempty"`
	Recurrence              *recurrenceDTO `json:"recurrence,omitempty"`
}

// recurrenceDTO is a local wire DTO for recurrence.
type recurrenceDTO struct {
	Type           string   `json:"type"`
	Interval       *int32   `json:"interval,omitempty"`
	DaysOfWeek     []string `json:"daysOfWeek,omitempty"`
	DaysOfMonth    []int32  `json:"daysOfMonth,omitempty"`
	WeekOfTheMonth []int32  `json:"weekOfTheMonth,omitempty"`
}

// recurrenceAttrTypes returns the canonical attr.Type map for the recurrence object.
func recurrenceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":             types.StringType,
		"interval":         types.Int64Type,
		"days_of_week":     types.ListType{ElemType: types.StringType},
		"days_of_month":    types.ListType{ElemType: types.Int64Type},
		"week_of_the_month": types.ListType{ElemType: types.Int64Type},
	}
}

// scheduleItemAttrTypes returns the canonical attr.Type map for a single schedule item.
func scheduleItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uuid":                       types.StringType,
		"display_name":               types.StringType,
		"start_date":                 types.StringType,
		"start_time":                 types.StringType,
		"end_time":                   types.StringType,
		"end_date":                   types.StringType,
		"time_zone":                  types.StringType,
		"next_occurrence":            types.StringType,
		"next_occurrence_timestamp":  types.Int64Type,
		"remaining_time_active_ms":   types.Int64Type,
		"recurrence":                 types.ObjectType{AttrTypes: recurrenceAttrTypes()},
	}
}

func (d *scheduleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

func (d *scheduleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	recurrenceSchema := schema.SingleNestedAttribute{
		Computed:    true,
		Description: "Recurrence rule for repeating schedules. Null for one-time schedules.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Recurrence type: DAILY, WEEKLY, or MONTHLY.",
			},
			"interval": schema.Int64Attribute{
				Computed:    true,
				Description: "Repeat every N days/weeks/months.",
			},
			"days_of_week": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Days of the week for WEEKLY recurrence. Values: Mon, Tue, Wed, Thu, Fri, Sat, Sun.",
			},
			"days_of_month": schema.ListAttribute{
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "Days of the month for MONTHLY recurrence (e.g. [1, 15]).",
			},
			"week_of_the_month": schema.ListAttribute{
				Computed:    true,
				ElementType: types.Int64Type,
				Description: "Which week(s) of the month for MONTHLY recurrence. -1=last, 1=first, etc.",
			},
		},
	}

	scheduleItemSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The UUID of the schedule.",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "The human-readable name of the schedule.",
			},
			"start_date": schema.StringAttribute{
				Computed:    true,
				Description: "Start date/time of the schedule window in ISO8601 local time (YYYY-MM-DDTHH:MM).",
			},
			"start_time": schema.StringAttribute{
				Computed:    true,
				Description: "Start time of the daily window in ISO8601 format.",
			},
			"end_time": schema.StringAttribute{
				Computed:    true,
				Description: "End time of the daily window in ISO8601 format.",
			},
			"end_date": schema.StringAttribute{
				Computed:    true,
				Description: "End date after which the schedule no longer recurs. Null for open-ended schedules.",
			},
			"time_zone": schema.StringAttribute{
				Computed:    true,
				Description: "IANA timezone name (e.g. America/New_York).",
			},
			"next_occurrence": schema.StringAttribute{
				Computed:    true,
				Description: "ISO8601 local datetime of the next trigger.",
			},
			"next_occurrence_timestamp": schema.Int64Attribute{
				Computed:    true,
				Description: "Unix millisecond timestamp of the next trigger.",
			},
			"remaining_time_active_ms": schema.Int64Attribute{
				Computed:    true,
				Description: "Milliseconds remaining in the current active window. Zero when not active.",
			},
			"recurrence": recurrenceSchema,
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns all Turbonomic schedules, optionally filtered by display name (exact match). " +
			"Schedules are referenced by UUID in turbonomic_settings_policy resources.",
		Attributes: map[string]schema.Attribute{
			"display_name": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to schedules whose display name matches this value exactly. Omit to return all schedules.",
			},
			"schedules": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "List of schedules matching the supplied filter. Empty when no schedules match.",
				NestedObject: scheduleItemSchema,
			},
		},
	}
}

func (d *scheduleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected data source configure type",
			fmt.Sprintf("Expected *providerData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.v2Client = data.V2Client

	if d.v2Client == nil {
		resp.Diagnostics.AddError(
			"v2 client not available",
			"The v2 client is required for schedule data source operations but is not available.",
		)
	}
}

func (d *scheduleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scheduleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	displayNameFilter := config.DisplayNameFilter.ValueString()

	tflog.Debug(ctx, "reading schedule data source", map[string]interface{}{
		"display_name_filter": displayNameFilter,
	})

	// Fetch all schedules via raw HTTP (same pattern as policy resources).
	httpClient := d.v2Client.GetHTTPClient()
	baseURL := d.v2Client.GetBaseURL()
	url := baseURL + "/schedules"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("error building schedules request", err.Error())
		return
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("error fetching schedules", err.Error())
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"unexpected status fetching schedules",
			fmt.Sprintf("GET %s returned HTTP %d", url, httpResp.StatusCode),
		)
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("error reading schedules response body", err.Error())
		return
	}

	var allSchedules []scheduleDTO
	if err := json.Unmarshal(body, &allSchedules); err != nil {
		resp.Diagnostics.AddError("error parsing schedules response", err.Error())
		return
	}

	// Apply filter and build result objects.
	itemType := types.ObjectType{AttrTypes: scheduleItemAttrTypes()}
	var elements []attr.Value

	for i := range allSchedules {
		s := &allSchedules[i]
		if displayNameFilter != "" {
			if s.DisplayName == nil || *s.DisplayName != displayNameFilter {
				continue
			}
		}
		obj, diags := buildScheduleItemObject(s)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elements = append(elements, obj)
	}

	if elements == nil {
		elements = []attr.Value{}
	}

	schedulesList, diags := types.ListValue(itemType, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "schedule data source read complete", map[string]interface{}{
		"total_fetched": len(allSchedules),
		"matched":       len(elements),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &scheduleDataSourceModel{
		DisplayNameFilter: config.DisplayNameFilter,
		Schedules:         schedulesList,
	})...)
}

// buildScheduleItemObject constructs a types.Object for a single scheduleDTO.
func buildScheduleItemObject(s *scheduleDTO) (attr.Value, diag.Diagnostics) {
	uuid := types.StringNull()
	if s.Uuid != nil {
		uuid = types.StringPointerValue(s.Uuid)
	}
	displayName := types.StringNull()
	if s.DisplayName != nil {
		displayName = types.StringPointerValue(s.DisplayName)
	}
	endDate := types.StringNull()
	if s.EndDate != nil {
		endDate = types.StringPointerValue(s.EndDate)
	}
	timeZone := types.StringNull()
	if s.TimeZone != nil {
		timeZone = types.StringPointerValue(s.TimeZone)
	}
	nextOccurrence := types.StringNull()
	if s.NextOccurrence != nil {
		nextOccurrence = types.StringPointerValue(s.NextOccurrence)
	}
	nextOccurrenceTS := types.Int64Null()
	if s.NextOccurrenceTimestamp != nil {
		nextOccurrenceTS = types.Int64Value(*s.NextOccurrenceTimestamp)
	}
	remainingMS := types.Int64Value(0)
	if s.RemaingTimeActiveInMs != nil {
		remainingMS = types.Int64Value(*s.RemaingTimeActiveInMs)
	}

	// Build recurrence object.
	recurrenceObj, recDiags := buildRecurrenceObject(s.Recurrence)
	if recDiags.HasError() {
		return nil, recDiags
	}

	obj, objDiags := types.ObjectValue(scheduleItemAttrTypes(), map[string]attr.Value{
		"uuid":                      uuid,
		"display_name":              displayName,
		"start_date":                types.StringValue(s.StartDate),
		"start_time":                types.StringValue(s.StartTime),
		"end_time":                  types.StringValue(s.EndTime),
		"end_date":                  endDate,
		"time_zone":                 timeZone,
		"next_occurrence":           nextOccurrence,
		"next_occurrence_timestamp": nextOccurrenceTS,
		"remaining_time_active_ms":  remainingMS,
		"recurrence":                recurrenceObj,
	})
	return obj, objDiags
}

// buildRecurrenceObject constructs a types.Object for a recurrenceDTO, or a null object when r is nil.
func buildRecurrenceObject(r *recurrenceDTO) (attr.Value, diag.Diagnostics) {
	if r == nil {
		return types.ObjectNull(recurrenceAttrTypes()), nil
	}

	// days_of_week - null when absent (not an empty list) so the plan stays consistent.
	var dowList types.List
	if len(r.DaysOfWeek) == 0 {
		dowList = types.ListNull(types.StringType)
	} else {
		dowElems := make([]attr.Value, len(r.DaysOfWeek))
		for i, d := range r.DaysOfWeek {
			dowElems[i] = types.StringValue(d)
		}
		var diags diag.Diagnostics
		dowList, diags = types.ListValue(types.StringType, dowElems)
		if diags.HasError() {
			return nil, diags
		}
	}

	// days_of_month - null when absent.
	var domList types.List
	if len(r.DaysOfMonth) == 0 {
		domList = types.ListNull(types.Int64Type)
	} else {
		domElems := make([]attr.Value, len(r.DaysOfMonth))
		for i, d := range r.DaysOfMonth {
			domElems[i] = types.Int64Value(int64(d))
		}
		var diags diag.Diagnostics
		domList, diags = types.ListValue(types.Int64Type, domElems)
		if diags.HasError() {
			return nil, diags
		}
	}

	// week_of_the_month - null when absent.
	var wotmList types.List
	if len(r.WeekOfTheMonth) == 0 {
		wotmList = types.ListNull(types.Int64Type)
	} else {
		wotmElems := make([]attr.Value, len(r.WeekOfTheMonth))
		for i, w := range r.WeekOfTheMonth {
			wotmElems[i] = types.Int64Value(int64(w))
		}
		var diags diag.Diagnostics
		wotmList, diags = types.ListValue(types.Int64Type, wotmElems)
		if diags.HasError() {
			return nil, diags
		}
	}

	interval := types.Int64Null()
	if r.Interval != nil {
		interval = types.Int64Value(int64(*r.Interval))
	}

	return types.ObjectValue(recurrenceAttrTypes(), map[string]attr.Value{
		"type":              types.StringValue(r.Type),
		"interval":          interval,
		"days_of_week":      dowList,
		"days_of_month":     domList,
		"week_of_the_month": wotmList,
	})
}
