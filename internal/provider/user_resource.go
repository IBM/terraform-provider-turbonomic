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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

// NewUserResource returns a new turbonomic_user resource instance.
func NewUserResource() resource.Resource {
	return &userResource{}
}

type userResource struct {
	v2Client *v2.Client
}

// userResourceModel is the Terraform state model for turbonomic_user.
type userResourceModel struct {
	ID              types.String `tfsdk:"id"`
	UUID            types.String `tfsdk:"uuid"`
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password"`
	DisplayName     types.String `tfsdk:"display_name"`
	LoginProvider   types.String `tfsdk:"login_provider"`
	Type            types.String `tfsdk:"type"`
	RoleNames       types.List   `tfsdk:"role_names"`
	ScopeGroupUUIDs types.List   `tfsdk:"scope_group_uuids"`
	ShowSharedSC    types.Bool   `tfsdk:"show_shared_user_sc"`
}

// userWriteDTO is the wire DTO for create/update (matches UserApiDTO).
type userWriteDTO struct {
	Username         string       `json:"username"`
	Password         *string      `json:"password,omitempty"`
	DisplayName      *string      `json:"displayName,omitempty"`
	LoginProvider    *string      `json:"loginProvider,omitempty"`
	Type             *string      `json:"type,omitempty"`
	Roles            []roleRefDTO `json:"roles,omitempty"`
	Scope            []scopeEntry `json:"scope,omitempty"`
	ShowSharedUserSC *bool        `json:"showSharedUserSC,omitempty"`
}

// roleRefDTO is used to reference a role by name in userWriteDTO.
type roleRefDTO struct {
	Name string `json:"name"`
}

// userReadDTO is the wire DTO for read responses (superset of write).
type userReadDTO struct {
	UUID             *string      `json:"uuid,omitempty"`
	Username         *string      `json:"username,omitempty"`
	DisplayName      *string      `json:"displayName,omitempty"`
	LoginProvider    *string      `json:"loginProvider,omitempty"`
	Type             *string      `json:"type,omitempty"`
	Roles            []roleDTO    `json:"roles,omitempty"`
	Scope            []scopeEntry `json:"scope,omitempty"`
	ShowSharedUserSC *bool        `json:"showSharedUserSC,omitempty"`
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a local Turbonomic user account including role assignments and group scope. " +
			"LDAP users are managed externally and should be read via the turbonomic_user data source. " +
			"The password attribute is write-only and never stored in state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal Terraform identifier, equal to the user UUID.",
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
			"username": schema.StringAttribute{
				Description: "Login username. Must be unique. Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{ // pragma: allowlist secret
				Description: "Password for the user account. Write-only - never stored in state. " +
					"Required on create for LOCAL users. Ignored for LDAP users.",
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Full display name of the user.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"login_provider": schema.StringAttribute{
				Description: "Authentication provider for the user: LOCAL or LDAP. Defaults to LOCAL.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("LOCAL", "LDAP"),
				},
			},
			"type": schema.StringAttribute{
				Description: "User account type: DedicatedCustomer or SharedCustomer. Defaults to DedicatedCustomer.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("DedicatedCustomer", "SharedCustomer"),
				},
			},
			"role_names": schema.ListAttribute{
				Description: "List of role names to assign to the user (e.g. [\"OBSERVER\", \"AUTOMATOR\"]). " +
					"See turbonomic_role data source for available roles.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"scope_group_uuids": schema.ListAttribute{
				Description: "List of group UUIDs that define the user's scope. Empty for unscoped (global) users.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"show_shared_user_sc": schema.BoolAttribute{
				Description: "Whether to show the shared-user supply chain. Defaults to false.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		resp.Diagnostics.AddError("v2 client not available", "The v2 client is required for user operations but is not available.")
	}
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := r.buildWriteDTO(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating user", map[string]interface{}{"username": plan.Username.ValueString()})

	body, err := policyHTTPPost(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/users", dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating User", err.Error())
		return
	}

	var created userReadDTO
	if err := json.Unmarshal(body, &created); err != nil {
		resp.Diagnostics.AddError("Error Parsing Create Response", err.Error())
		return
	}
	if created.UUID == nil {
		resp.Diagnostics.AddError("Error Creating User", "API response did not include a UUID.")
		return
	}

	r.mapDTOToModel(&created, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		uuid = state.ID.ValueString()
	}

	body, statusCode, err := policyHTTPGet(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/users/"+uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading User", err.Error())
		return
	}
	if statusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	var dto userReadDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		resp.Diagnostics.AddError("Error Parsing User Response", err.Error())
		return
	}

	r.mapDTOToModel(&dto, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.UUID = state.UUID
	plan.ID = state.ID
	uuid := state.UUID.ValueString()

	dto := r.buildWriteDTO(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating user", map[string]interface{}{"uuid": uuid})

	body, err := policyHTTPPut(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/users/"+uuid, dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating User", err.Error())
		return
	}

	var updated userReadDTO
	if err := json.Unmarshal(body, &updated); err != nil {
		resp.Diagnostics.AddError("Error Parsing Update Response", err.Error())
		return
	}

	r.mapDTOToModel(&updated, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()
	tflog.Debug(ctx, "deleting user", map[string]interface{}{"uuid": uuid})
	if err := policyHTTPDelete(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/users/"+uuid); err != nil {
		resp.Diagnostics.AddError("Error Deleting User", err.Error())
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *userResource) buildWriteDTO(ctx context.Context, plan userResourceModel, diags *diag.Diagnostics) userWriteDTO {
	dto := userWriteDTO{
		Username: plan.Username.ValueString(),
	}

	setIfKnown(&dto.Password, plan.Password, types.String.ValueString)       // pragma: allowlist secret
	setIfKnown(&dto.DisplayName, plan.DisplayName, types.String.ValueString)
	setIfKnown(&dto.LoginProvider, plan.LoginProvider, types.String.ValueString)
	setIfKnown(&dto.Type, plan.Type, types.String.ValueString)
	setIfKnown(&dto.ShowSharedUserSC, plan.ShowSharedSC, types.Bool.ValueBool)

	if !plan.RoleNames.IsNull() && !plan.RoleNames.IsUnknown() {
		var names []string
		diags.Append(plan.RoleNames.ElementsAs(ctx, &names, false)...)
		if !diags.HasError() {
			for _, n := range names {
				dto.Roles = append(dto.Roles, roleRefDTO{Name: n})
			}
		}
	}

	if !plan.ScopeGroupUUIDs.IsNull() && !plan.ScopeGroupUUIDs.IsUnknown() {
		var uuids []string
		diags.Append(plan.ScopeGroupUUIDs.ElementsAs(ctx, &uuids, false)...)
		if !diags.HasError() {
			for _, u := range uuids {
				uu := u
				dto.Scope = append(dto.Scope, scopeEntry{Uuid: &uu})
			}
		}
	}

	return dto
}

func (r *userResource) mapDTOToModel(dto *userReadDTO, model *userResourceModel, diags *diag.Diagnostics) {
	if dto.UUID != nil {
		model.UUID = types.StringPointerValue(dto.UUID)
		model.ID = types.StringPointerValue(dto.UUID)
	}
	if dto.Username != nil {
		model.Username = types.StringPointerValue(dto.Username)
	}
	// The API echoes back the username when no explicit display_name is set.
	// Only store if a real distinct display name was provided; otherwise preserve the plan
	// value (which is null when display_name was omitted in config).
	if dto.DisplayName != nil && *dto.DisplayName != model.Username.ValueString() {
		model.DisplayName = types.StringPointerValue(dto.DisplayName)
	} else if model.DisplayName.IsUnknown() {
		// Plan had no display_name - clear it to null so state is consistent.
		model.DisplayName = types.StringNull()
	}
	if dto.LoginProvider != nil {
		model.LoginProvider = types.StringPointerValue(dto.LoginProvider)
	} else {
		model.LoginProvider = types.StringNull()
	}
	if dto.Type != nil {
		model.Type = types.StringPointerValue(dto.Type)
	} else {
		model.Type = types.StringNull()
	}
	if dto.ShowSharedUserSC != nil {
		model.ShowSharedSC = types.BoolPointerValue(dto.ShowSharedUserSC)
	} else {
		model.ShowSharedSC = types.BoolValue(false)
	}

	roleElems := make([]attr.Value, 0, len(dto.Roles))
	for _, r := range dto.Roles {
		if r.Name != nil {
			roleElems = append(roleElems, types.StringPointerValue(r.Name))
		}
	}
	roleList, d := types.ListValue(types.StringType, roleElems)
	diags.Append(d...)
	if !d.HasError() {
		model.RoleNames = roleList
	}

	scopeElems := make([]attr.Value, 0, len(dto.Scope))
	for _, s := range dto.Scope {
		if s.Uuid != nil {
			scopeElems = append(scopeElems, types.StringPointerValue(s.Uuid))
		}
	}
	scopeList, d := types.ListValue(types.StringType, scopeElems)
	diags.Append(d...)
	if !d.HasError() {
		model.ScopeGroupUUIDs = scopeList
	}

	// password is write-only - never overwrite from the API response.
}
