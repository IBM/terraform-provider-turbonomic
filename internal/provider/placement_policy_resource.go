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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	v2 "github.com/IBM/turbonomic-go-client/v2"
)

var (
	_ resource.Resource                = &placementPolicyResource{}
	_ resource.ResourceWithConfigure   = &placementPolicyResource{}
	_ resource.ResourceWithImportState = &placementPolicyResource{}
	_ resource.ResourceWithModifyPlan  = &placementPolicyResource{}
)

// NewPlacementPolicyResource constructs a new turbonomic_placement_policy resource.
func NewPlacementPolicyResource() resource.Resource {
	return &placementPolicyResource{}
}

type placementPolicyResource struct {
	v2Client *v2.Client
}

// placementPolicyResourceModel holds the Terraform state for a placement policy.
type placementPolicyResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Type                     types.String `tfsdk:"type"`
	Enabled                  types.Bool   `tfsdk:"enabled"`
	BuyerGroupUUID           types.String `tfsdk:"buyer_group_uuid"`
	SellerGroupUUID          types.String `tfsdk:"seller_group_uuid"`
	MergeUUIDs               types.List   `tfsdk:"merge_uuids"`
	Capacity                 types.Int64  `tfsdk:"capacity"`
	ProviderEntityType       types.String `tfsdk:"provider_entity_type"`
	EnableCreateResourcePool types.Bool   `tfsdk:"enable_create_resource_pool"`
	MergeType                types.String `tfsdk:"merge_type"`
	UUID                     types.String `tfsdk:"uuid"`
}

type placementPolicyInputDTO struct {
	PolicyName               string   `json:"policyName"`
	Type                     string   `json:"type"`
	Enabled                  *bool    `json:"enabled,omitempty"`
	BuyerUUID                *string  `json:"buyerUuid,omitempty"`
	SellerUUID               *string  `json:"sellerUuid,omitempty"`
	MergeUUIDs               []string `json:"mergeUuids,omitempty"`
	Capacity                 *int64   `json:"capacity,omitempty"`
	ProviderEntityType       *string  `json:"providerEntityType,omitempty"`
	EnableCreateResourcePool *bool    `json:"enableCreateResourcePool,omitempty"`
	MergeType                *string  `json:"mergeType,omitempty"`
}

type baseRefDTO struct {
	UUID *string `json:"uuid,omitempty"`
}

// placementPolicyDTO mirrors PolicyApiDTO for Read responses.
// consumerGroup/providerGroup/mergeGroups are full objects - we extract .uuid only.
type placementPolicyDTO struct {
	UUID                     *string      `json:"uuid,omitempty"`
	Name                     *string      `json:"name,omitempty"`
	Type                     string       `json:"type"`
	Enabled                  *bool        `json:"enabled,omitempty"`
	Capacity                 *int64       `json:"capacity,omitempty"`
	MergeType                *string      `json:"mergeType,omitempty"`
	ConsumerGroup            *baseRefDTO  `json:"consumerGroup,omitempty"`
	ProviderGroup            *baseRefDTO  `json:"providerGroup,omitempty"`
	MergeGroups              []baseRefDTO `json:"mergeGroups,omitempty"`
	ProviderEntityType       *string      `json:"providerEntityType,omitempty"`
	EnableCreateResourcePool *bool        `json:"enableCreateResourcePool,omitempty"`
}

func (r *placementPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_placement_policy"
}

func (r *placementPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Turbonomic placement policy. Placement policies control where virtual machines can or cannot run relative to groups of hosts, clusters, or other VMs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal Terraform identifier, equal to the policy UUID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Display name of the placement policy.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Placement policy type. One of: BIND_TO_GROUP, MUST_NOT_RUN_TOGETHER, MUST_RUN_TOGETHER, MERGE, AT_MOST_N, AT_MOST_N_BOUND, BIND_TO_COMPLEMENTARY_GROUP, BIND_TO_GROUP_AND_LICENSE, EXCLUSIVE_BIND_TO_GROUP.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"BIND_TO_GROUP",
						"MUST_NOT_RUN_TOGETHER",
						"MUST_RUN_TOGETHER",
						"MERGE",
						"AT_MOST_N",
						"AT_MOST_N_BOUND",
						"BIND_TO_COMPLEMENTARY_GROUP",
						"BIND_TO_GROUP_AND_LICENSE",
						"EXCLUSIVE_BIND_TO_GROUP",
					),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the policy is enabled. Defaults to true.",
				Optional:    true,
			},
			"buyer_group_uuid": schema.StringAttribute{
				Description: "UUID of the consumer (buyer) group. Required for non-MERGE policy types.",
				Optional:    true,
			},
			"seller_group_uuid": schema.StringAttribute{
				Description: "UUID of the provider (seller) group. Required for non-MERGE policy types.",
				Optional:    true,
			},
			"merge_uuids": schema.ListAttribute{
				Description: "UUIDs of the groups to merge. Required for MERGE policy type.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"capacity": schema.Int64Attribute{
				Description: "Maximum number of consumer entities allowed per provider entity. Used with AT_MOST_N and AT_MOST_N_BOUND policy types.",
				Optional:    true,
			},
			"provider_entity_type": schema.StringAttribute{
				Description: "Entity type constraint on the provider group.",
				Optional:    true,
			},
			"enable_create_resource_pool": schema.BoolAttribute{
				Description: "When true, creates a resource pool at the destination during cross-cluster VM moves. Only applicable to MERGE policies with Cluster merge type.",
				Optional:    true,
			},
			"merge_type": schema.StringAttribute{
				Description: "Type of merge for MERGE policies. One of: Cluster, DataCenter, DesktopPool, Network, StorageCluster.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("Cluster", "DataCenter", "DesktopPool", "Network", "StorageCluster"),
				},
			},
			"uuid": schema.StringAttribute{
				Description: "UUID assigned by Turbonomic after creation.",
				Computed:    true,
			},
		},
	}
}

func (r *placementPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
		resp.Diagnostics.AddError("v2 client not available", "The v2 client is required for placement policy operations but is not available.")
		return
	}
}

func (r *placementPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan placementPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := r.buildInputDTO(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating placement policy", map[string]interface{}{"name": plan.Name.ValueString()})

	body, err := policyHTTPPost(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/markets/Market/policies", dto)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Placement Policy", err.Error())
		return
	}

	var created placementPolicyDTO
	if err := json.Unmarshal(body, &created); err != nil {
		resp.Diagnostics.AddError("Error Parsing Create Response", err.Error())
		return
	}
	if created.UUID == nil {
		resp.Diagnostics.AddError("Error Creating Placement Policy", "API response did not include a UUID.")
		return
	}

	plan.UUID = types.StringPointerValue(created.UUID)
	plan.ID = types.StringPointerValue(created.UUID)

	// Re-read to populate computed fields from authoritative API response.
	r.readIntoModel(ctx, *created.UUID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *placementPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state placementPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	if uuid == "" {
		uuid = state.ID.ValueString()
	}

	// 404 handling: readIntoModel sets UUID to null, we then remove from state.
	r.readIntoModel(ctx, uuid, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.UUID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *placementPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan placementPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state placementPolicyResourceModel
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

	tflog.Debug(ctx, "updating placement policy", map[string]interface{}{"uuid": uuid})

	if _, err := policyHTTPPut(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/markets/Market/policies/"+uuid, dto); err != nil {
		resp.Diagnostics.AddError("Error Updating Placement Policy", err.Error())
		return
	}

	r.readIntoModel(ctx, uuid, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *placementPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state placementPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()
	tflog.Debug(ctx, "deleting placement policy", map[string]interface{}{"uuid": uuid})
	if err := policyHTTPDelete(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/markets/Market/policies/"+uuid); err != nil {
		resp.Diagnostics.AddError("Error Deleting Placement Policy", err.Error())
	}
}

func (r *placementPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ModifyPlan enforces cross-field conditionality at plan time:
//   - MERGE policies: merge_uuids required; seller_group_uuid, buyer_group_uuid must not be set
//   - Non-MERGE policies: buyer_group_uuid and seller_group_uuid required; merge_uuids, merge_type,
//     enable_create_resource_pool must not be set
func (r *placementPolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan placementPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// type may be unknown during initial plan (shouldn't happen since it's Required, but guard anyway).
	if plan.Type.IsUnknown() {
		return
	}

	resp.Diagnostics.Append(validatePlacementPolicyPlan(plan)...)
}

// validatePlacementPolicyPlan is the pure validation logic extracted from ModifyPlan
// so it can be unit-tested without a live framework Plan object.
func validatePlacementPolicyPlan(plan placementPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	policyType := plan.Type.ValueString()
	isMerge := policyType == "MERGE"

	if isMerge {
		if plan.MergeUUIDs.IsNull() || plan.MergeUUIDs.IsUnknown() || len(plan.MergeUUIDs.Elements()) == 0 {
			diags.AddAttributeError(
				path.Root("merge_uuids"),
				"merge_uuids required for MERGE policy",
				"MERGE placement policies require merge_uuids to contain at least one group UUID.",
			)
		}
		if !plan.SellerGroupUUID.IsNull() && !plan.SellerGroupUUID.IsUnknown() {
			diags.AddAttributeError(
				path.Root("seller_group_uuid"),
				"seller_group_uuid not allowed for MERGE policy",
				"seller_group_uuid is only valid for non-MERGE placement policies. Remove it when type is MERGE.",
			)
		}
		if !plan.BuyerGroupUUID.IsNull() && !plan.BuyerGroupUUID.IsUnknown() {
			diags.AddAttributeError(
				path.Root("buyer_group_uuid"),
				"buyer_group_uuid not allowed for MERGE policy",
				"buyer_group_uuid is only valid for non-MERGE placement policies. Remove it when type is MERGE.",
			)
		}
	} else {
		// Non-MERGE: buyer_group_uuid is required.
		// IsUnknown means the value is known-after-apply (e.g. referencing a group resource
		// being created in the same plan) - skip the check in that case.
		if plan.BuyerGroupUUID.IsNull() {
			diags.AddAttributeError(
				path.Root("buyer_group_uuid"),
				"buyer_group_uuid required for non-MERGE policy",
				fmt.Sprintf("buyer_group_uuid is required when policy type is %q.", policyType),
			)
		}
		if plan.SellerGroupUUID.IsNull() {
			diags.AddAttributeError(
				path.Root("seller_group_uuid"),
				"seller_group_uuid required for non-MERGE policy",
				fmt.Sprintf("seller_group_uuid is required when policy type is %q.", policyType),
			)
		}
		if !plan.MergeUUIDs.IsNull() && !plan.MergeUUIDs.IsUnknown() && len(plan.MergeUUIDs.Elements()) > 0 {
			diags.AddAttributeError(
				path.Root("merge_uuids"),
				"merge_uuids not allowed for non-MERGE policy",
				fmt.Sprintf("merge_uuids is only valid when type is MERGE, but type is %q.", policyType),
			)
		}
		if !plan.MergeType.IsNull() && !plan.MergeType.IsUnknown() {
			diags.AddAttributeError(
				path.Root("merge_type"),
				"merge_type not allowed for non-MERGE policy",
				fmt.Sprintf("merge_type is only valid when type is MERGE, but type is %q.", policyType),
			)
		}
		if !plan.EnableCreateResourcePool.IsNull() && !plan.EnableCreateResourcePool.IsUnknown() {
			diags.AddAttributeError(
				path.Root("enable_create_resource_pool"),
				"enable_create_resource_pool not allowed for non-MERGE policy",
				fmt.Sprintf("enable_create_resource_pool is only valid when type is MERGE, but type is %q.", policyType),
			)
		}
	}

	return diags
}

// readIntoModel fetches a placement policy by UUID and populates model.
// On 404, sets model.UUID to null so the caller can remove from state.
// NOTE: Read uses GET /policies/{uuid} - NOT /markets/Market/policies/{uuid}
func (r *placementPolicyResource) readIntoModel(ctx context.Context, uuid string, model *placementPolicyResourceModel, diags *diag.Diagnostics) {
	body, statusCode, err := policyHTTPGet(ctx, r.v2Client.GetHTTPClient(), r.v2Client.GetBaseURL(), "/policies/"+uuid)
	if err != nil {
		diags.AddError("Error Reading Placement Policy", err.Error())
		return
	}
	if statusCode == http.StatusNotFound {
		model.UUID = types.StringNull()
		return
	}

	var dto placementPolicyDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		diags.AddError("Error Parsing Placement Policy Response", err.Error())
		return
	}
	r.mapDTOToModel(&dto, model, diags)
}

func (r *placementPolicyResource) buildInputDTO(ctx context.Context, plan placementPolicyResourceModel, diags *diag.Diagnostics) placementPolicyInputDTO {
	dto := placementPolicyInputDTO{
		PolicyName: plan.Name.ValueString(),
		Type:       plan.Type.ValueString(),
	}
	setIfKnown(&dto.Enabled, plan.Enabled, types.Bool.ValueBool)
	setIfKnown(&dto.BuyerUUID, plan.BuyerGroupUUID, types.String.ValueString)
	setIfKnown(&dto.SellerUUID, plan.SellerGroupUUID, types.String.ValueString)
	if !plan.MergeUUIDs.IsNull() && !plan.MergeUUIDs.IsUnknown() {
		var uuids []string
		diags.Append(plan.MergeUUIDs.ElementsAs(ctx, &uuids, false)...)
		dto.MergeUUIDs = uuids
	}
	setIfKnown(&dto.Capacity, plan.Capacity, types.Int64.ValueInt64)
	setIfKnown(&dto.ProviderEntityType, plan.ProviderEntityType, types.String.ValueString)
	setIfKnown(&dto.EnableCreateResourcePool, plan.EnableCreateResourcePool, types.Bool.ValueBool)
	setIfKnown(&dto.MergeType, plan.MergeType, types.String.ValueString)
	return dto
}

func (r *placementPolicyResource) mapDTOToModel(dto *placementPolicyDTO, model *placementPolicyResourceModel, diags *diag.Diagnostics) {
	if dto.UUID != nil {
		model.UUID = types.StringPointerValue(dto.UUID)
		model.ID = types.StringPointerValue(dto.UUID)
	}
	if dto.Name != nil {
		model.Name = types.StringPointerValue(dto.Name)
	}
	model.Type = types.StringValue(dto.Type)
	if dto.Enabled != nil {
		model.Enabled = types.BoolPointerValue(dto.Enabled)
	}
	if dto.Capacity != nil {
		model.Capacity = types.Int64PointerValue(dto.Capacity)
	} else {
		model.Capacity = types.Int64Null()
	}
	if dto.ProviderEntityType != nil {
		model.ProviderEntityType = types.StringPointerValue(dto.ProviderEntityType)
	} else {
		model.ProviderEntityType = types.StringNull()
	}
	if dto.EnableCreateResourcePool != nil {
		model.EnableCreateResourcePool = types.BoolPointerValue(dto.EnableCreateResourcePool)
	} else {
		model.EnableCreateResourcePool = types.BoolNull()
	}
	if dto.MergeType != nil {
		model.MergeType = types.StringPointerValue(dto.MergeType)
	} else {
		model.MergeType = types.StringNull()
	}
	// buyer/seller: extract uuid from nested ConsumerGroup/ProviderGroup objects
	if dto.ConsumerGroup != nil && dto.ConsumerGroup.UUID != nil {
		model.BuyerGroupUUID = types.StringPointerValue(dto.ConsumerGroup.UUID)
	} else {
		model.BuyerGroupUUID = types.StringNull()
	}
	if dto.ProviderGroup != nil && dto.ProviderGroup.UUID != nil {
		model.SellerGroupUUID = types.StringPointerValue(dto.ProviderGroup.UUID)
	} else {
		model.SellerGroupUUID = types.StringNull()
	}
	// merge_uuids: extract uuid from each MergeGroups element
	if len(dto.MergeGroups) > 0 {
		elems := make([]attr.Value, 0, len(dto.MergeGroups))
		for _, g := range dto.MergeGroups {
			if g.UUID != nil {
				elems = append(elems, types.StringPointerValue(g.UUID))
			}
		}
		var d diag.Diagnostics
		model.MergeUUIDs, d = types.ListValue(types.StringType, elems)
		diags.Append(d...)
	} else {
		model.MergeUUIDs = types.ListNull(types.StringType)
	}
}
