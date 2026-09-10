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
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newPlacementResource() *placementPolicyResource {
	return &placementPolicyResource{}
}

// TestPlacementPolicy_BuildInputDTO_BasicBindToGroup verifies the DTO built from
// a BIND_TO_GROUP plan has the expected fields.
func TestPlacementPolicy_BuildInputDTO_BasicBindToGroup(t *testing.T) {
	r := newPlacementResource()

	buyerUUID := "buyer-uuid-123"
	sellerUUID := "seller-uuid-456"

	plan := placementPolicyResourceModel{
		Name:            types.StringValue("my-policy"),
		Type:            types.StringValue("BIND_TO_GROUP"),
		Enabled:         types.BoolValue(true),
		BuyerGroupUUID:  types.StringValue(buyerUUID),
		SellerGroupUUID: types.StringValue(sellerUUID),
		MergeUUIDs:      types.ListNull(types.StringType),
		Capacity:        types.Int64Null(),
		MergeType:       types.StringNull(),
	}

	var d diag.Diagnostics
	dto := r.buildInputDTO(context.Background(), plan, &d)
	if d.HasError() {
		t.Fatalf("buildInputDTO returned diagnostics errors: %v", d)
	}

	if dto.PolicyName != "my-policy" {
		t.Errorf("PolicyName: got %q, want %q", dto.PolicyName, "my-policy")
	}
	if dto.Type != "BIND_TO_GROUP" {
		t.Errorf("Type: got %q, want %q", dto.Type, "BIND_TO_GROUP")
	}
	if dto.Enabled == nil || !*dto.Enabled {
		t.Error("Enabled should be true")
	}
	if dto.BuyerUUID == nil || *dto.BuyerUUID != buyerUUID {
		t.Errorf("BuyerUUID: got %v, want %q", dto.BuyerUUID, buyerUUID)
	}
	if dto.SellerUUID == nil || *dto.SellerUUID != sellerUUID {
		t.Errorf("SellerUUID: got %v, want %q", dto.SellerUUID, sellerUUID)
	}
	if dto.MergeUUIDs != nil {
		t.Errorf("MergeUUIDs should be nil for BIND_TO_GROUP, got %v", dto.MergeUUIDs)
	}
	if dto.Capacity != nil {
		t.Errorf("Capacity should be nil, got %v", dto.Capacity)
	}
}

// TestPlacementPolicy_BuildInputDTO_Merge verifies MERGE-type DTO construction.
func TestPlacementPolicy_BuildInputDTO_Merge(t *testing.T) {
	r := newPlacementResource()

	mergeList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"uuid-a", "uuid-b"})

	plan := placementPolicyResourceModel{
		Name:       types.StringValue("merge-policy"),
		Type:       types.StringValue("MERGE"),
		Enabled:    types.BoolNull(),
		MergeUUIDs: mergeList,
		MergeType:  types.StringValue("Cluster"),
	}

	var d diag.Diagnostics
	dto := r.buildInputDTO(context.Background(), plan, &d)
	if d.HasError() {
		t.Fatalf("buildInputDTO returned diagnostics errors: %v", d)
	}

	if dto.Type != "MERGE" {
		t.Errorf("Type: got %q, want %q", dto.Type, "MERGE")
	}
	if len(dto.MergeUUIDs) != 2 {
		t.Errorf("MergeUUIDs length: got %d, want 2", len(dto.MergeUUIDs))
	}
	if dto.MergeType == nil || *dto.MergeType != "Cluster" {
		t.Errorf("MergeType: got %v, want %q", dto.MergeType, "Cluster")
	}
	if dto.Enabled != nil {
		t.Errorf("Enabled should be nil when not set, got %v", dto.Enabled)
	}
	if dto.BuyerUUID != nil {
		t.Errorf("BuyerUUID should be nil for MERGE, got %v", dto.BuyerUUID)
	}
}

// TestPlacementPolicy_BuildInputDTO_AtMostN verifies AT_MOST_N capacity is set.
func TestPlacementPolicy_BuildInputDTO_AtMostN(t *testing.T) {
	r := newPlacementResource()

	plan := placementPolicyResourceModel{
		Name:            types.StringValue("at-most-n"),
		Type:            types.StringValue("AT_MOST_N"),
		BuyerGroupUUID:  types.StringValue("buyer"),
		SellerGroupUUID: types.StringValue("seller"),
		Capacity:        types.Int64Value(5),
		MergeUUIDs:      types.ListNull(types.StringType),
	}

	var d diag.Diagnostics
	dto := r.buildInputDTO(context.Background(), plan, &d)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	if dto.Capacity == nil || *dto.Capacity != 5 {
		t.Errorf("Capacity: got %v, want 5", dto.Capacity)
	}
}

// TestPlacementPolicy_MapDTOToModel_BasicFields verifies Read response mapping.
func TestPlacementPolicy_MapDTOToModel_BasicFields(t *testing.T) {
	r := newPlacementResource()

	uuid := "policy-uuid-789"
	name := "my-policy"
	policyType := "BIND_TO_GROUP"
	enabled := true
	buyerUUID := "buyer-uuid-123"
	sellerUUID := "seller-uuid-456"

	dto := &placementPolicyDTO{
		UUID:    &uuid,
		Name:    &name,
		Type:    policyType,
		Enabled: &enabled,
		ConsumerGroup: &baseRefDTO{
			UUID: &buyerUUID,
		},
		ProviderGroup: &baseRefDTO{
			UUID: &sellerUUID,
		},
	}

	model := &placementPolicyResourceModel{}
	var d diag.Diagnostics
	r.mapDTOToModel(dto, model, &d)
	if d.HasError() {
		t.Fatalf("mapDTOToModel returned diagnostics errors: %v", d)
	}

	if model.UUID.ValueString() != uuid {
		t.Errorf("UUID: got %q, want %q", model.UUID.ValueString(), uuid)
	}
	if model.ID.ValueString() != uuid {
		t.Errorf("ID: got %q, want %q", model.ID.ValueString(), uuid)
	}
	if model.Name.ValueString() != name {
		t.Errorf("Name: got %q, want %q", model.Name.ValueString(), name)
	}
	if model.Type.ValueString() != policyType {
		t.Errorf("Type: got %q, want %q", model.Type.ValueString(), policyType)
	}
	if model.Enabled.ValueBool() != enabled {
		t.Errorf("Enabled: got %v, want %v", model.Enabled.ValueBool(), enabled)
	}
	if model.BuyerGroupUUID.ValueString() != buyerUUID {
		t.Errorf("BuyerGroupUUID: got %q, want %q", model.BuyerGroupUUID.ValueString(), buyerUUID)
	}
	if model.SellerGroupUUID.ValueString() != sellerUUID {
		t.Errorf("SellerGroupUUID: got %q, want %q", model.SellerGroupUUID.ValueString(), sellerUUID)
	}
}

// TestPlacementPolicy_MapDTOToModel_MergeGroups verifies merge UUIDs are extracted.
func TestPlacementPolicy_MapDTOToModel_MergeGroups(t *testing.T) {
	r := newPlacementResource()

	uuid := "merge-policy-uuid"
	policyType := "MERGE"
	uuidA := "group-a"
	uuidB := "group-b"

	dto := &placementPolicyDTO{
		UUID: &uuid,
		Type: policyType,
		MergeGroups: []baseRefDTO{
			{UUID: &uuidA},
			{UUID: &uuidB},
		},
	}

	model := &placementPolicyResourceModel{}
	var d diag.Diagnostics
	r.mapDTOToModel(dto, model, &d)
	if d.HasError() {
		t.Fatalf("mapDTOToModel returned diagnostics errors: %v", d)
	}

	if model.MergeUUIDs.IsNull() {
		t.Fatal("MergeUUIDs should not be null")
	}
	elems := model.MergeUUIDs.Elements()
	if len(elems) != 2 {
		t.Fatalf("MergeUUIDs length: got %d, want 2", len(elems))
	}
}

// TestPlacementPolicy_MapDTOToModel_NilGroupsBecomesNull verifies optional fields
// that are absent in the API response are set to null in the model.
func TestPlacementPolicy_MapDTOToModel_NilGroupsBecomesNull(t *testing.T) {
	r := newPlacementResource()

	uuid := "policy-uuid"
	dto := &placementPolicyDTO{
		UUID: &uuid,
		Type: "MUST_NOT_RUN_TOGETHER",
		// No ConsumerGroup, ProviderGroup, or MergeGroups
	}

	model := &placementPolicyResourceModel{}
	var d diag.Diagnostics
	r.mapDTOToModel(dto, model, &d)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	if !model.BuyerGroupUUID.IsNull() {
		t.Error("BuyerGroupUUID should be null when ConsumerGroup is absent")
	}
	if !model.SellerGroupUUID.IsNull() {
		t.Error("SellerGroupUUID should be null when ProviderGroup is absent")
	}
	if !model.MergeUUIDs.IsNull() {
		t.Error("MergeUUIDs should be null when MergeGroups is empty")
	}
	if !model.Capacity.IsNull() {
		t.Error("Capacity should be null when absent in DTO")
	}
}

// ── validatePlacementPolicyPlan ──────────────────────────────────────────────

func TestValidatePlacementPolicyPlan_MergeValid(t *testing.T) {
	mergeList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"uuid-a"})
	plan := placementPolicyResourceModel{
		Type:       types.StringValue("MERGE"),
		MergeUUIDs: mergeList,
		MergeType:  types.StringValue("Cluster"),
		// buyer/seller not set
		BuyerGroupUUID:  types.StringNull(),
		SellerGroupUUID: types.StringNull(),
	}
	diags := validatePlacementPolicyPlan(plan)
	if diags.HasError() {
		t.Errorf("expected no errors for valid MERGE plan, got: %v", diags)
	}
}

func TestValidatePlacementPolicyPlan_MergeNoMergeUUIDs(t *testing.T) {
	plan := placementPolicyResourceModel{
		Type:            types.StringValue("MERGE"),
		MergeUUIDs:      types.ListNull(types.StringType),
		BuyerGroupUUID:  types.StringNull(),
		SellerGroupUUID: types.StringNull(),
	}
	diags := validatePlacementPolicyPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when MERGE policy has no merge_uuids")
	}
}

func TestValidatePlacementPolicyPlan_MergeWithSellerUUID(t *testing.T) {
	mergeList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"uuid-a"})
	plan := placementPolicyResourceModel{
		Type:            types.StringValue("MERGE"),
		MergeUUIDs:      mergeList,
		BuyerGroupUUID:  types.StringNull(),
		SellerGroupUUID: types.StringValue("seller-123"), // not allowed
	}
	diags := validatePlacementPolicyPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when MERGE policy sets seller_group_uuid")
	}
}

func TestValidatePlacementPolicyPlan_MergeWithBuyerUUID(t *testing.T) {
	mergeList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"uuid-a"})
	plan := placementPolicyResourceModel{
		Type:            types.StringValue("MERGE"),
		MergeUUIDs:      mergeList,
		BuyerGroupUUID:  types.StringValue("buyer-123"), // not allowed
		SellerGroupUUID: types.StringNull(),
	}
	diags := validatePlacementPolicyPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when MERGE policy sets buyer_group_uuid")
	}
}

func TestValidatePlacementPolicyPlan_BindToGroupValid(t *testing.T) {
	plan := placementPolicyResourceModel{
		Type:            types.StringValue("BIND_TO_GROUP"),
		BuyerGroupUUID:  types.StringValue("buyer-uuid"),
		SellerGroupUUID: types.StringValue("seller-uuid"),
		MergeUUIDs:      types.ListNull(types.StringType),
		MergeType:       types.StringNull(),
		EnableCreateResourcePool: types.BoolNull(),
	}
	diags := validatePlacementPolicyPlan(plan)
	if diags.HasError() {
		t.Errorf("expected no errors for valid BIND_TO_GROUP plan, got: %v", diags)
	}
}

func TestValidatePlacementPolicyPlan_NonMergeNoBuyer(t *testing.T) {
	plan := placementPolicyResourceModel{
		Type:            types.StringValue("BIND_TO_GROUP"),
		BuyerGroupUUID:  types.StringNull(), // missing
		SellerGroupUUID: types.StringValue("seller-uuid"),
		MergeUUIDs:      types.ListNull(types.StringType),
	}
	diags := validatePlacementPolicyPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when non-MERGE policy is missing buyer_group_uuid")
	}
}

func TestValidatePlacementPolicyPlan_NonMergeNoSeller(t *testing.T) {
	plan := placementPolicyResourceModel{
		Type:            types.StringValue("MUST_NOT_RUN_TOGETHER"),
		BuyerGroupUUID:  types.StringValue("buyer-uuid"),
		SellerGroupUUID: types.StringNull(), // missing
		MergeUUIDs:      types.ListNull(types.StringType),
	}
	diags := validatePlacementPolicyPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when non-MERGE policy is missing seller_group_uuid")
	}
}

func TestValidatePlacementPolicyPlan_NonMergeWithMergeUUIDs(t *testing.T) {
	mergeList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"uuid-a"})
	plan := placementPolicyResourceModel{
		Type:            types.StringValue("BIND_TO_GROUP"),
		BuyerGroupUUID:  types.StringValue("buyer-uuid"),
		SellerGroupUUID: types.StringValue("seller-uuid"),
		MergeUUIDs:      mergeList, // not allowed
	}
	diags := validatePlacementPolicyPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when non-MERGE policy sets merge_uuids")
	}
}

func TestValidatePlacementPolicyPlan_NonMergeWithMergeType(t *testing.T) {
	plan := placementPolicyResourceModel{
		Type:            types.StringValue("BIND_TO_GROUP"),
		BuyerGroupUUID:  types.StringValue("buyer-uuid"),
		SellerGroupUUID: types.StringValue("seller-uuid"),
		MergeUUIDs:      types.ListNull(types.StringType),
		MergeType:       types.StringValue("Cluster"), // not allowed
	}
	diags := validatePlacementPolicyPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when non-MERGE policy sets merge_type")
	}
}

func TestValidatePlacementPolicyPlan_NonMergeWithEnableCreateResourcePool(t *testing.T) {
	plan := placementPolicyResourceModel{
		Type:                     types.StringValue("BIND_TO_GROUP"),
		BuyerGroupUUID:           types.StringValue("buyer-uuid"),
		SellerGroupUUID:          types.StringValue("seller-uuid"),
		MergeUUIDs:               types.ListNull(types.StringType),
		MergeType:                types.StringNull(),
		EnableCreateResourcePool: types.BoolValue(true), // not allowed
	}
	diags := validatePlacementPolicyPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when non-MERGE policy sets enable_create_resource_pool")
	}
}
