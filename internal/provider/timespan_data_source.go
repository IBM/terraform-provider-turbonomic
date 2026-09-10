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

var _ datasource.DataSource = &timespanDataSource{}
var _ datasource.DataSourceWithConfigure = &timespanDataSource{}

// NewTimespanDataSource returns a new instance of the turbonomic_timespan data source.
func NewTimespanDataSource() datasource.DataSource {
	return &timespanDataSource{}
}

type timespanDataSource struct {
	v2Client *v2.Client
}

// timespanDataSourceModel is the root Terraform state model.
type timespanDataSourceModel struct {
	DisplayNameFilter types.String `tfsdk:"display_name"`
	Timespans         types.List   `tfsdk:"timespans"`
}

// timespanDTO is the local wire DTO for a timespan schedule returned by GET /schedules/timespans.
type timespanDTO struct {
	UUID        *string `json:"uuid,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	Description *string `json:"description,omitempty"`
	TimeZone    *string `json:"timeZone,omitempty"`
}

// timespanItemAttrTypes returns the canonical attr.Type map for a single timespan item.
func timespanItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uuid":         types.StringType,
		"display_name": types.StringType,
		"description":  types.StringType,
		"time_zone":    types.StringType,
	}
}

func (d *timespanDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_timespan"
}

func (d *timespanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	timespanItemSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the timespan schedule.",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable name of the timespan schedule.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the timespan schedule.",
			},
			"time_zone": schema.StringAttribute{
				Computed:    true,
				Description: "IANA timezone used for the time spans in this schedule (e.g. America/New_York).",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns Turbonomic time-span schedules (parking schedules), optionally filtered by display name. " +
			"Reference a timespan UUID in turbonomic_parking_policy via the attach_schedule block.",
		Attributes: map[string]schema.Attribute{
			"display_name": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to timespans whose display name matches exactly. Omit to return all timespans.",
			},
			"timespans": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "List of timespan schedules matching the supplied filter. Empty when no timespans match.",
				NestedObject: timespanItemSchema,
			},
		},
	}
}

func (d *timespanDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The v2 client is required for timespan data source operations but is not available.",
		)
	}
}

func (d *timespanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config timespanDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	displayNameFilter := config.DisplayNameFilter.ValueString()
	tflog.Debug(ctx, "reading timespan data source", map[string]interface{}{"display_name_filter": displayNameFilter})

	httpClient := d.v2Client.GetHTTPClient()
	baseURL := d.v2Client.GetBaseURL()
	url := baseURL + "/schedules/timespans"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("error building timespans request", err.Error())
		return
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("error fetching timespans", err.Error())
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"unexpected status fetching timespans",
			fmt.Sprintf("GET %s returned HTTP %d", url, httpResp.StatusCode),
		)
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("error reading timespans response body", err.Error())
		return
	}

	var allTimespans []timespanDTO
	if err := json.Unmarshal(body, &allTimespans); err != nil {
		resp.Diagnostics.AddError("error parsing timespans response", err.Error())
		return
	}

	itemType := types.ObjectType{AttrTypes: timespanItemAttrTypes()}
	var elements []attr.Value

	for i := range allTimespans {
		t := &allTimespans[i]
		if displayNameFilter != "" {
			if t.DisplayName == nil || *t.DisplayName != displayNameFilter {
				continue
			}
		}
		obj, diags := buildTimespanItemObject(t)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elements = append(elements, obj)
	}

	if elements == nil {
		elements = []attr.Value{}
	}

	timespansList, diags := types.ListValue(itemType, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "timespan data source read complete", map[string]interface{}{
		"total_fetched": len(allTimespans),
		"matched":       len(elements),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &timespanDataSourceModel{
		DisplayNameFilter: config.DisplayNameFilter,
		Timespans:         timespansList,
	})...)
}

// buildTimespanItemObject constructs a types.Object for a single timespanDTO.
func buildTimespanItemObject(t *timespanDTO) (attr.Value, diag.Diagnostics) {
	uuid := types.StringNull()
	if t.UUID != nil {
		uuid = types.StringPointerValue(t.UUID)
	}
	displayName := types.StringNull()
	if t.DisplayName != nil {
		displayName = types.StringPointerValue(t.DisplayName)
	}
	description := types.StringNull()
	if t.Description != nil {
		description = types.StringPointerValue(t.Description)
	}
	timeZone := types.StringNull()
	if t.TimeZone != nil {
		timeZone = types.StringPointerValue(t.TimeZone)
	}

	return types.ObjectValue(timespanItemAttrTypes(), map[string]attr.Value{
		"uuid":         uuid,
		"display_name": displayName,
		"description":  description,
		"time_zone":    timeZone,
	})
}
