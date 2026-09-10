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

var _ datasource.DataSource = &placementPolicyDataSource{}
var _ datasource.DataSourceWithConfigure = &placementPolicyDataSource{}

// NewPlacementPolicyDataSource returns a new instance of the turbonomic_placement_policy data source.
func NewPlacementPolicyDataSource() datasource.DataSource {
	return &placementPolicyDataSource{}
}

type placementPolicyDataSource struct {
	v2Client *v2.Client
}

// placementPolicyDataSourceModel is the root Terraform state model.
type placementPolicyDataSourceModel struct {
	NameFilter    types.String `tfsdk:"name"`
	TypeFilter    types.String `tfsdk:"type"`
	EnabledFilter types.Bool   `tfsdk:"enabled"`
	Policies      types.List   `tfsdk:"policies"`
}

// placementPolicyItemAttrTypes returns the canonical attr.Type map for a single policy item.
func placementPolicyItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uuid":                       types.StringType,
		"name":                       types.StringType,
		"type":                       types.StringType,
		"enabled":                    types.BoolType,
		"buyer_group_uuid":           types.StringType,
		"seller_group_uuid":          types.StringType,
		"merge_uuids":                types.ListType{ElemType: types.StringType},
		"merge_type":                 types.StringType,
		"capacity":                   types.Int64Type,
		"provider_entity_type":       types.StringType,
		"enable_create_resource_pool": types.BoolType,
	}
}

func (d *placementPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_placement_policy"
}

func (d *placementPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	policyItemSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The UUID of the placement policy.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Display name of the placement policy.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Placement policy type (e.g. BIND_TO_GROUP, MERGE, AT_MOST_N).",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the policy is enabled.",
			},
			"buyer_group_uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the consumer (buyer) group. Set for non-MERGE policies.",
			},
			"seller_group_uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the provider (seller) group. Set for non-MERGE policies.",
			},
			"merge_uuids": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "UUIDs of the groups being merged. Set for MERGE policies.",
			},
			"merge_type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of merge (e.g. Cluster, DataCenter). Set for MERGE policies.",
			},
			"capacity": schema.Int64Attribute{
				Computed:    true,
				Description: "Maximum consumer entities per provider entity. Set for AT_MOST_N and AT_MOST_N_BOUND policies.",
			},
			"provider_entity_type": schema.StringAttribute{
				Computed:    true,
				Description: "Entity type constraint on the provider group.",
			},
			"enable_create_resource_pool": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether a resource pool is created at the destination during cross-cluster moves.",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns all Turbonomic placement policies, optionally filtered by name, type, or enabled state. " +
			"Use this data source to look up policy UUIDs for use in other resources.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to policies whose name matches this value exactly. Omit to return all policies.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to policies of this type (e.g. BIND_TO_GROUP, MERGE). Omit to return all types.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Filter results to enabled (true) or disabled (false) policies. Omit to return both.",
			},
			"policies": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "List of placement policies matching the supplied filters. Empty when no policies match.",
				NestedObject: policyItemSchema,
			},
		},
	}
}

func (d *placementPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The v2 client is required for placement policy data source operations but is not available.",
		)
	}
}

func (d *placementPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config placementPolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameFilter := config.NameFilter.ValueString()
	typeFilter := config.TypeFilter.ValueString()

	tflog.Debug(ctx, "reading placement policy data source", map[string]interface{}{
		"name_filter": nameFilter,
		"type_filter": typeFilter,
	})

	// Fetch all placement policies.
	// NOTE: The list endpoint is /markets/Market/policies (same as Create/Update/Delete).
	//       The per-UUID Read endpoint is /policies/{uuid} (no /markets/ prefix).
	httpClient := d.v2Client.GetHTTPClient()
	baseURL := d.v2Client.GetBaseURL()
	url := baseURL + "/markets/Market/policies"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("error building placement policies request", err.Error())
		return
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("error fetching placement policies", err.Error())
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"unexpected status fetching placement policies",
			fmt.Sprintf("GET %s returned HTTP %d", url, httpResp.StatusCode),
		)
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("error reading placement policies response body", err.Error())
		return
	}

	var allPolicies []placementPolicyDTO
	if err := json.Unmarshal(body, &allPolicies); err != nil {
		resp.Diagnostics.AddError("error parsing placement policies response", err.Error())
		return
	}

	// Apply filters and build result objects.
	itemType := types.ObjectType{AttrTypes: placementPolicyItemAttrTypes()}
	var elements []attr.Value

	for i := range allPolicies {
		p := &allPolicies[i]
		if nameFilter != "" {
			if p.Name == nil || *p.Name != nameFilter {
				continue
			}
		}
		if typeFilter != "" && p.Type != typeFilter {
			continue
		}
		if !config.EnabledFilter.IsNull() {
			want := config.EnabledFilter.ValueBool()
			if p.Enabled == nil || *p.Enabled != want {
				continue
			}
		}
		obj, diags := buildPlacementPolicyItemObject(p)
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

	tflog.Debug(ctx, "placement policy data source read complete", map[string]interface{}{
		"total_fetched": len(allPolicies),
		"matched":       len(elements),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &placementPolicyDataSourceModel{
		NameFilter:    config.NameFilter,
		TypeFilter:    config.TypeFilter,
		EnabledFilter: config.EnabledFilter,
		Policies:      policiesList,
	})...)
}

// buildPlacementPolicyItemObject constructs a types.Object for a single placementPolicyDTO.
func buildPlacementPolicyItemObject(p *placementPolicyDTO) (attr.Value, diag.Diagnostics) {
	uuid := types.StringNull()
	if p.UUID != nil {
		uuid = types.StringPointerValue(p.UUID)
	}
	name := types.StringNull()
	if p.Name != nil {
		name = types.StringPointerValue(p.Name)
	}
	enabled := types.BoolNull()
	if p.Enabled != nil {
		enabled = types.BoolPointerValue(p.Enabled)
	}
	capacity := types.Int64Null()
	if p.Capacity != nil {
		capacity = types.Int64PointerValue(p.Capacity)
	}
	providerEntityType := types.StringNull()
	if p.ProviderEntityType != nil {
		providerEntityType = types.StringPointerValue(p.ProviderEntityType)
	}
	enableCreateResourcePool := types.BoolNull()
	if p.EnableCreateResourcePool != nil {
		enableCreateResourcePool = types.BoolPointerValue(p.EnableCreateResourcePool)
	}
	mergeType := types.StringNull()
	if p.MergeType != nil {
		mergeType = types.StringPointerValue(p.MergeType)
	}
	buyerGroupUUID := types.StringNull()
	if p.ConsumerGroup != nil && p.ConsumerGroup.UUID != nil {
		buyerGroupUUID = types.StringPointerValue(p.ConsumerGroup.UUID)
	}
	sellerGroupUUID := types.StringNull()
	if p.ProviderGroup != nil && p.ProviderGroup.UUID != nil {
		sellerGroupUUID = types.StringPointerValue(p.ProviderGroup.UUID)
	}

	// merge_uuids: extract uuid from each MergeGroups element.
	var mergeUUIDs types.List
	if len(p.MergeGroups) > 0 {
		elems := make([]attr.Value, 0, len(p.MergeGroups))
		for _, g := range p.MergeGroups {
			if g.UUID != nil {
				elems = append(elems, types.StringPointerValue(g.UUID))
			}
		}
		var d diag.Diagnostics
		mergeUUIDs, d = types.ListValue(types.StringType, elems)
		if d.HasError() {
			return nil, d
		}
	} else {
		mergeUUIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	obj, diags := types.ObjectValue(placementPolicyItemAttrTypes(), map[string]attr.Value{
		"uuid":                        uuid,
		"name":                        name,
		"type":                        types.StringValue(p.Type),
		"enabled":                     enabled,
		"buyer_group_uuid":            buyerGroupUUID,
		"seller_group_uuid":           sellerGroupUUID,
		"merge_uuids":                 mergeUUIDs,
		"merge_type":                  mergeType,
		"capacity":                    capacity,
		"provider_entity_type":        providerEntityType,
		"enable_create_resource_pool": enableCreateResourcePool,
	})
	return obj, diags
}
