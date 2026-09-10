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

var _ datasource.DataSource = &settingsPolicyDataSource{}
var _ datasource.DataSourceWithConfigure = &settingsPolicyDataSource{}

// NewSettingsPolicyDataSource returns a new instance of the turbonomic_settings_policy data source.
func NewSettingsPolicyDataSource() datasource.DataSource {
	return &settingsPolicyDataSource{}
}

type settingsPolicyDataSource struct {
	v2Client *v2.Client
}

// settingsPolicyDataSourceModel is the root Terraform state model.
type settingsPolicyDataSourceModel struct {
	NameFilter       types.String `tfsdk:"name"`
	EntityTypeFilter types.String `tfsdk:"entity_type"`
	DisabledFilter   types.Bool   `tfsdk:"disabled"`
	Policies         types.List   `tfsdk:"policies"`
}

// settingsPolicyItemAttrTypes returns the canonical attr.Type map for a single policy item.
func settingsPolicyItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uuid":          types.StringType,
		"name":          types.StringType,
		"entity_type":   types.StringType,
		"disabled":      types.BoolType,
		"default":       types.BoolType,
		"read_only":     types.BoolType,
		"note":          types.StringType,
		"scope_uuids":   types.ListType{ElemType: types.StringType},
		"schedule_uuid": types.StringType,
	}
}

func (d *settingsPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings_policy"
}

func (d *settingsPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	policyItemSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The UUID of the settings policy.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Display name of the settings policy.",
			},
			"entity_type": schema.StringAttribute{
				Computed:    true,
				Description: "Entity type this policy targets (e.g. VirtualMachine, PhysicalMachine).",
			},
			"disabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the policy is disabled.",
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "True when this is a default (system-provided) policy.",
			},
			"read_only": schema.BoolAttribute{
				Computed:    true,
				Description: "True when the policy is system-managed and cannot be modified.",
			},
			"note": schema.StringAttribute{
				Computed:    true,
				Description: "Free-text note attached to the policy.",
			},
			"scope_uuids": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "UUIDs of the groups this policy is scoped to. Empty list means the policy applies globally.",
			},
			"schedule_uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the schedule attached to this policy, if any.",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns all Turbonomic settings (automation) policies, optionally filtered by name, entity type, or disabled state. " +
			"Use this data source to look up policy UUIDs for use in other resources.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to policies whose name matches this value exactly. Omit to return all policies.",
			},
			"entity_type": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to policies targeting this entity type (e.g. VirtualMachine). Omit to return all entity types.",
			},
			"disabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Filter results to disabled (true) or enabled (false) policies. Omit to return both.",
			},
			"policies": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "List of settings policies matching the supplied filters. Empty when no policies match.",
				NestedObject: policyItemSchema,
			},
		},
	}
}

func (d *settingsPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The v2 client is required for settings policy data source operations but is not available.",
		)
	}
}

func (d *settingsPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config settingsPolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameFilter := config.NameFilter.ValueString()
	entityTypeFilter := config.EntityTypeFilter.ValueString()

	tflog.Debug(ctx, "reading settings policy data source", map[string]interface{}{
		"name_filter":        nameFilter,
		"entity_type_filter": entityTypeFilter,
	})

	httpClient := d.v2Client.GetHTTPClient()
	baseURL := d.v2Client.GetBaseURL()
	url := baseURL + "/settingspolicies"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("error building settings policies request", err.Error())
		return
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("error fetching settings policies", err.Error())
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"unexpected status fetching settings policies",
			fmt.Sprintf("GET %s returned HTTP %d", url, httpResp.StatusCode),
		)
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("error reading settings policies response body", err.Error())
		return
	}

	var allPolicies []settingsPolicyDTO
	if err := json.Unmarshal(body, &allPolicies); err != nil {
		resp.Diagnostics.AddError("error parsing settings policies response", err.Error())
		return
	}

	// Apply filters and build result objects.
	itemType := types.ObjectType{AttrTypes: settingsPolicyItemAttrTypes()}
	var elements []attr.Value

	for i := range allPolicies {
		p := &allPolicies[i]
		if nameFilter != "" {
			if p.DisplayName == nil || *p.DisplayName != nameFilter {
				continue
			}
		}
		if entityTypeFilter != "" {
			if p.EntityType == nil || *p.EntityType != entityTypeFilter {
				continue
			}
		}
		if !config.DisabledFilter.IsNull() {
			want := config.DisabledFilter.ValueBool()
			if p.Disabled == nil || *p.Disabled != want {
				continue
			}
		}
		obj, diags := buildSettingsPolicyItemObject(p)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elements = append(elements, obj)
	}

	if elements == nil {
		elements = []attr.Value{}
	}

	policiesList, diags := types.ListValue(itemType, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "settings policy data source read complete", map[string]interface{}{
		"total_fetched": len(allPolicies),
		"matched":       len(elements),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &settingsPolicyDataSourceModel{
		NameFilter:       config.NameFilter,
		EntityTypeFilter: config.EntityTypeFilter,
		DisabledFilter:   config.DisabledFilter,
		Policies:         policiesList,
	})...)
}

// buildSettingsPolicyItemObject constructs a types.Object for a single settingsPolicyDTO.
func buildSettingsPolicyItemObject(p *settingsPolicyDTO) (attr.Value, diag.Diagnostics) {
	uuid := types.StringNull()
	if p.UUID != nil {
		uuid = types.StringPointerValue(p.UUID)
	}
	name := types.StringNull()
	if p.DisplayName != nil {
		name = types.StringPointerValue(p.DisplayName)
	}
	entityType := types.StringNull()
	if p.EntityType != nil {
		entityType = types.StringPointerValue(p.EntityType)
	}
	disabled := types.BoolNull()
	if p.Disabled != nil {
		disabled = types.BoolPointerValue(p.Disabled)
	}
	isDefault := types.BoolNull()
	if p.Default != nil {
		isDefault = types.BoolPointerValue(p.Default)
	}
	readOnly := types.BoolNull()
	if p.ReadOnly != nil {
		readOnly = types.BoolPointerValue(p.ReadOnly)
	}
	note := types.StringNull()
	if p.Note != nil {
		note = types.StringPointerValue(p.Note)
	}
	scheduleUUID := types.StringNull()
	if p.Schedule != nil {
		scheduleUUID = types.StringValue(p.Schedule.UUID)
	}

	// scope_uuids: extract uuid from each Scopes element.
	scopeElems := make([]attr.Value, 0, len(p.Scopes))
	for _, s := range p.Scopes {
		scopeElems = append(scopeElems, types.StringValue(s.UUID))
	}
	scopeUUIDs, scopeDiags := types.ListValue(types.StringType, scopeElems)
	if scopeDiags.HasError() {
		return nil, scopeDiags
	}

	obj, diags := types.ObjectValue(settingsPolicyItemAttrTypes(), map[string]attr.Value{
		"uuid":          uuid,
		"name":          name,
		"entity_type":   entityType,
		"disabled":      disabled,
		"default":       isDefault,
		"read_only":     readOnly,
		"note":          note,
		"scope_uuids":   scopeUUIDs,
		"schedule_uuid": scheduleUUID,
	})
	return obj, diags
}
