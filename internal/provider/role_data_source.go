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

var _ datasource.DataSource = &roleDataSource{}
var _ datasource.DataSourceWithConfigure = &roleDataSource{}

// NewRoleDataSource returns a new instance of the turbonomic_role data source.
func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

type roleDataSource struct {
	v2Client *v2.Client
}

// roleDataSourceModel is the root Terraform state model.
type roleDataSourceModel struct {
	NameFilter types.String `tfsdk:"name"`
	Roles      types.List   `tfsdk:"roles"`
}

// roleAPIDTO is the local wire DTO for a role returned by GET /roles.
type roleAPIDTO struct {
	UUID        *string `json:"uuid,omitempty"`
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	Description *string `json:"description,omitempty"`
}

// roleItemAttrTypes returns the canonical attr.Type map for a single role item.
func roleItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uuid":         types.StringType,
		"name":         types.StringType,
		"display_name": types.StringType,
		"description":  types.StringType,
	}
}

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	roleItemSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the role.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Internal role name (e.g. ADMINISTRATOR, OBSERVER, AUTOMATOR).",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable display name of the role.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the role.",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns Turbonomic roles, optionally filtered by name. " +
			"Use this data source to look up role names for use in turbonomic_user resources.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to the role whose name matches exactly (e.g. ADMINISTRATOR). Omit to return all roles.",
			},
			"roles": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "List of roles matching the supplied filter. Empty when no roles match.",
				NestedObject: roleItemSchema,
			},
		},
	}
}

func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The v2 client is required for role data source operations but is not available.",
		)
	}
}

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameFilter := config.NameFilter.ValueString()
	tflog.Debug(ctx, "reading role data source", map[string]interface{}{"name_filter": nameFilter})

	httpClient := d.v2Client.GetHTTPClient()
	baseURL := d.v2Client.GetBaseURL()
	url := baseURL + "/roles"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("error building roles request", err.Error())
		return
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("error fetching roles", err.Error())
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"unexpected status fetching roles",
			fmt.Sprintf("GET %s returned HTTP %d", url, httpResp.StatusCode),
		)
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("error reading roles response body", err.Error())
		return
	}

	var allRoles []roleAPIDTO
	if err := json.Unmarshal(body, &allRoles); err != nil {
		resp.Diagnostics.AddError("error parsing roles response", err.Error())
		return
	}

	itemType := types.ObjectType{AttrTypes: roleItemAttrTypes()}
	var elements []attr.Value

	for i := range allRoles {
		r := &allRoles[i]
		if nameFilter != "" {
			if r.Name == nil || *r.Name != nameFilter {
				continue
			}
		}
		obj, diags := buildRoleItemObject(r)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elements = append(elements, obj)
	}

	if elements == nil {
		elements = []attr.Value{}
	}

	rolesList, diags := types.ListValue(itemType, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "role data source read complete", map[string]interface{}{
		"total_fetched": len(allRoles),
		"matched":       len(elements),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &roleDataSourceModel{
		NameFilter: config.NameFilter,
		Roles:      rolesList,
	})...)
}

// buildRoleItemObject constructs a types.Object for a single roleAPIDTO.
func buildRoleItemObject(r *roleAPIDTO) (attr.Value, diag.Diagnostics) {
	uuid := types.StringNull()
	if r.UUID != nil {
		uuid = types.StringPointerValue(r.UUID)
	}
	name := types.StringNull()
	if r.Name != nil {
		name = types.StringPointerValue(r.Name)
	}
	displayName := types.StringNull()
	if r.DisplayName != nil {
		displayName = types.StringPointerValue(r.DisplayName)
	}
	description := types.StringNull()
	if r.Description != nil {
		description = types.StringPointerValue(r.Description)
	}

	return types.ObjectValue(roleItemAttrTypes(), map[string]attr.Value{
		"uuid":         uuid,
		"name":         name,
		"display_name": displayName,
		"description":  description,
	})
}
