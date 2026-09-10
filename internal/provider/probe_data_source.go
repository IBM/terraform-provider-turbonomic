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

var _ datasource.DataSource = &probeDataSource{}
var _ datasource.DataSourceWithConfigure = &probeDataSource{}

// NewProbeDataSource returns a new instance of the turbonomic_probe data source.
func NewProbeDataSource() datasource.DataSource {
	return &probeDataSource{}
}

type probeDataSource struct {
	v2Client *v2.Client
}

// probeDataSourceModel is the root Terraform state model.
type probeDataSourceModel struct {
	TypeFilter     types.String `tfsdk:"type"`
	CategoryFilter types.String `tfsdk:"category"`
	Probes         types.List   `tfsdk:"probes"`
}

// probeDTO is the local wire DTO for a probe returned by GET /probes.
// The API returns TargetApiDTO objects but we only need the probe-level fields.
type probeDTO struct {
	UUID        *string `json:"uuid,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	Type        *string `json:"type,omitempty"`
	Category    *string `json:"category,omitempty"`
}

// probeItemAttrTypes returns the canonical attr.Type map for a single probe item.
func probeItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uuid":         types.StringType,
		"display_name": types.StringType,
		"type":         types.StringType,
		"category":     types.StringType,
	}
}

func (d *probeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_probe"
}

func (d *probeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	probeItemSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the probe.",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable name of the probe.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Probe type identifier (e.g. Kubernetes, AWS, Azure, GCP, vCenter).",
			},
			"category": schema.StringAttribute{
				Computed:    true,
				Description: "Probe category (e.g. Cloud Native, Cloud Management, Hypervisor).",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns Turbonomic probes (target types), optionally filtered by type or category. " +
			"Use this data source to look up probe UUIDs before creating turbonomic_target resources.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to probes whose type matches exactly (e.g. Kubernetes, AWS). Omit to return all probe types.",
			},
			"category": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to probes in this category (e.g. Cloud Native). Omit to return all categories.",
			},
			"probes": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "List of probes matching the supplied filters. Empty when no probes match.",
				NestedObject: probeItemSchema,
			},
		},
	}
}

func (d *probeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The v2 client is required for probe data source operations but is not available.",
		)
	}
}

func (d *probeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config probeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	typeFilter := config.TypeFilter.ValueString()
	categoryFilter := config.CategoryFilter.ValueString()
	tflog.Debug(ctx, "reading probe data source", map[string]interface{}{
		"type_filter":     typeFilter,
		"category_filter": categoryFilter,
	})

	httpClient := d.v2Client.GetHTTPClient()
	baseURL := d.v2Client.GetBaseURL()
	url := baseURL + "/probes"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("error building probes request", err.Error())
		return
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("error fetching probes", err.Error())
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"unexpected status fetching probes",
			fmt.Sprintf("GET %s returned HTTP %d", url, httpResp.StatusCode),
		)
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("error reading probes response body", err.Error())
		return
	}

	var allProbes []probeDTO
	if err := json.Unmarshal(body, &allProbes); err != nil {
		resp.Diagnostics.AddError("error parsing probes response", err.Error())
		return
	}

	itemType := types.ObjectType{AttrTypes: probeItemAttrTypes()}
	var elements []attr.Value

	for i := range allProbes {
		p := &allProbes[i]
		if typeFilter != "" {
			if p.Type == nil || *p.Type != typeFilter {
				continue
			}
		}
		if categoryFilter != "" {
			if p.Category == nil || *p.Category != categoryFilter {
				continue
			}
		}
		obj, diags := buildProbeItemObject(p)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elements = append(elements, obj)
	}

	if elements == nil {
		elements = []attr.Value{}
	}

	probesList, diags := types.ListValue(itemType, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "probe data source read complete", map[string]interface{}{
		"total_fetched": len(allProbes),
		"matched":       len(elements),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &probeDataSourceModel{
		TypeFilter:     config.TypeFilter,
		CategoryFilter: config.CategoryFilter,
		Probes:         probesList,
	})...)
}

// buildProbeItemObject constructs a types.Object for a single probeDTO.
func buildProbeItemObject(p *probeDTO) (attr.Value, diag.Diagnostics) {
	uuid := types.StringNull()
	if p.UUID != nil {
		uuid = types.StringPointerValue(p.UUID)
	}
	displayName := types.StringNull()
	if p.DisplayName != nil {
		displayName = types.StringPointerValue(p.DisplayName)
	}
	probeType := types.StringNull()
	if p.Type != nil {
		probeType = types.StringPointerValue(p.Type)
	}
	category := types.StringNull()
	if p.Category != nil {
		category = types.StringPointerValue(p.Category)
	}

	return types.ObjectValue(probeItemAttrTypes(), map[string]attr.Value{
		"uuid":         uuid,
		"display_name": displayName,
		"type":         probeType,
		"category":     category,
	})
}
