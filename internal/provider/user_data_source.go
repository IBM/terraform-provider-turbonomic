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

var _ datasource.DataSource = &userDataSource{}
var _ datasource.DataSourceWithConfigure = &userDataSource{}

// NewUserDataSource returns a new instance of the turbonomic_user data source.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

type userDataSource struct {
	v2Client *v2.Client
}

// userDataSourceModel is the root Terraform state model.
type userDataSourceModel struct {
	UsernameFilter      types.String `tfsdk:"username"`
	RoleFilter          types.String `tfsdk:"role"`
	LoginProviderFilter types.String `tfsdk:"login_provider"`
	Users               types.List   `tfsdk:"users"`
}

// userDTO is a local wire DTO for JSON decoding.
// password and authToken are intentionally omitted - never written to state.
type userDTO struct {
	Uuid          *string      `json:"uuid,omitempty"`
	DisplayName   *string      `json:"displayName,omitempty"`
	Username      *string      `json:"username,omitempty"`
	LoginProvider *string      `json:"loginProvider,omitempty"`
	Type          *string      `json:"type,omitempty"`
	Roles         []roleDTO    `json:"roles,omitempty"`
	Scope         []scopeEntry `json:"scope,omitempty"`
}

// roleDTO is a local wire DTO for a user role.
type roleDTO struct {
	Uuid        *string `json:"uuid,omitempty"`
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	Description *string `json:"description,omitempty"`
}

// scopeEntry is used only to extract the UUID from the GroupApiDTO scope entries.
type scopeEntry struct {
	Uuid *string `json:"uuid,omitempty"`
}

// roleAttrTypes returns the canonical attr.Type map for a role sub-object.
func roleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uuid":         types.StringType,
		"name":         types.StringType,
		"display_name": types.StringType,
		"description":  types.StringType,
	}
}

// userItemAttrTypes returns the canonical attr.Type map for a single user item.
func userItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uuid":              types.StringType,
		"username":          types.StringType,
		"display_name":      types.StringType,
		"login_provider":    types.StringType,
		"type":              types.StringType,
		"roles":             types.ListType{ElemType: types.ObjectType{AttrTypes: roleAttrTypes()}},
		"scope_group_uuids": types.ListType{ElemType: types.StringType},
	}
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	roleSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The UUID of the role.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Role name enum value (e.g. ADMINISTRATOR, OBSERVER, AUTOMATOR).",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable name of the role.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the role.",
			},
		},
	}

	userItemSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The UUID of the user.",
			},
			"username": schema.StringAttribute{
				Computed:    true,
				Description: "Login username.",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Full display name of the user.",
			},
			"login_provider": schema.StringAttribute{
				Computed:    true,
				Description: "Authentication provider for the user: Local or LDAP.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "User account type: DedicatedCustomer or SharedCustomer.",
			},
			"roles": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "Roles assigned to the user.",
				NestedObject: roleSchema,
			},
			"scope_group_uuids": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "UUIDs of the group scopes assigned to the user. Empty for unscoped users.",
			},
		},
	}

	resp.Schema = schema.Schema{
		Description: "Returns Turbonomic users, optionally filtered by username, role, or login provider. " +
			"Requires ADMINISTRATOR or SITE_ADMIN privileges. " +
			"Sensitive fields (password, authToken) are never written to state.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to users whose username matches this value exactly.",
			},
			"role": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to users who have a role with this name (e.g. ADMINISTRATOR, OBSERVER).",
			},
			"login_provider": schema.StringAttribute{
				Optional:    true,
				Description: "Filter results to users with this login provider: Local or LDAP.",
			},
			"users": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "List of users matching the supplied filters. Empty when no users match.",
				NestedObject: userItemSchema,
			},
		},
	}
}

func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The v2 client is required for user data source operations but is not available.",
		)
	}
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	usernameFilter := config.UsernameFilter.ValueString()
	roleFilter := config.RoleFilter.ValueString()
	loginProviderFilter := config.LoginProviderFilter.ValueString()

	tflog.Debug(ctx, "reading user data source", map[string]interface{}{
		"username_filter":       usernameFilter,
		"role_filter":           roleFilter,
		"login_provider_filter": loginProviderFilter,
	})

	// Fetch all users via raw HTTP (same pattern as schedule / policy resources).
	httpClient := d.v2Client.GetHTTPClient()
	baseURL := d.v2Client.GetBaseURL()
	url := baseURL + "/users"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("error building users request", err.Error())
		return
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("error fetching users", err.Error())
		return
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"unexpected status fetching users",
			fmt.Sprintf("GET %s returned HTTP %d. ADMINISTRATOR or SITE_ADMIN role is required.", url, httpResp.StatusCode),
		)
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("error reading users response body", err.Error())
		return
	}

	var allUsers []userDTO
	if err := json.Unmarshal(body, &allUsers); err != nil {
		resp.Diagnostics.AddError("error parsing users response", err.Error())
		return
	}

	// Apply filters and build result objects.
	itemType := types.ObjectType{AttrTypes: userItemAttrTypes()}
	var elements []attr.Value

	for i := range allUsers {
		u := &allUsers[i]

		if usernameFilter != "" {
			if u.Username == nil || *u.Username != usernameFilter {
				continue
			}
		}
		if loginProviderFilter != "" {
			if u.LoginProvider == nil || *u.LoginProvider != loginProviderFilter {
				continue
			}
		}
		if roleFilter != "" {
			if !userHasRole(u, roleFilter) {
				continue
			}
		}

		obj, diags := buildUserItemObject(u)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elements = append(elements, obj)
	}

	if elements == nil {
		elements = []attr.Value{}
	}

	usersList, diags := types.ListValue(itemType, elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "user data source read complete", map[string]interface{}{
		"total_fetched": len(allUsers),
		"matched":       len(elements),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &userDataSourceModel{
		UsernameFilter:      config.UsernameFilter,
		RoleFilter:          config.RoleFilter,
		LoginProviderFilter: config.LoginProviderFilter,
		Users:               usersList,
	})...)
}

// userHasRole reports whether the user has at least one role matching the given name.
func userHasRole(u *userDTO, roleName string) bool {
	for _, r := range u.Roles {
		if r.Name != nil && *r.Name == roleName {
			return true
		}
	}
	return false
}

// buildUserItemObject constructs a types.Object for a single userDTO.
func buildUserItemObject(u *userDTO) (attr.Value, diag.Diagnostics) {
	uuid := types.StringNull()
	if u.Uuid != nil {
		uuid = types.StringPointerValue(u.Uuid)
	}
	username := types.StringNull()
	if u.Username != nil {
		username = types.StringPointerValue(u.Username)
	}
	displayName := types.StringNull()
	if u.DisplayName != nil {
		displayName = types.StringPointerValue(u.DisplayName)
	}
	loginProvider := types.StringNull()
	if u.LoginProvider != nil {
		loginProvider = types.StringPointerValue(u.LoginProvider)
	}
	userType := types.StringNull()
	if u.Type != nil {
		userType = types.StringPointerValue(u.Type)
	}

	// Build roles list.
	roleObjType := types.ObjectType{AttrTypes: roleAttrTypes()}
	roleElems := make([]attr.Value, len(u.Roles))
	for i, r := range u.Roles {
		roleObj, diags := buildRoleObject(&r)
		if diags.HasError() {
			return nil, diags
		}
		roleElems[i] = roleObj
	}
	rolesList, diags := types.ListValue(roleObjType, roleElems)
	if diags.HasError() {
		return nil, diags
	}

	// Build scope_group_uuids list - extract uuid only from scope GroupApiDTO objects.
	scopeElems := make([]attr.Value, 0, len(u.Scope))
	for _, s := range u.Scope {
		if s.Uuid != nil {
			scopeElems = append(scopeElems, types.StringPointerValue(s.Uuid))
		}
	}
	scopeList, diags := types.ListValue(types.StringType, scopeElems)
	if diags.HasError() {
		return nil, diags
	}

	return types.ObjectValue(userItemAttrTypes(), map[string]attr.Value{
		"uuid":              uuid,
		"username":          username,
		"display_name":      displayName,
		"login_provider":    loginProvider,
		"type":              userType,
		"roles":             rolesList,
		"scope_group_uuids": scopeList,
	})
}

// buildRoleObject constructs a types.Object for a single roleDTO.
func buildRoleObject(r *roleDTO) (attr.Value, diag.Diagnostics) {
	uuid := types.StringNull()
	if r.Uuid != nil {
		uuid = types.StringPointerValue(r.Uuid)
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

	return types.ObjectValue(roleAttrTypes(), map[string]attr.Value{
		"uuid":         uuid,
		"name":         name,
		"display_name": displayName,
		"description":  description,
	})
}
