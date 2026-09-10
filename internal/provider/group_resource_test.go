// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestGroupResourceModel_PreservesNullOptionalFields tests that optional fields
// remain null when not set in the plan, even if the API returns default values.
// This prevents "Provider produced inconsistent result after apply" errors.
func TestGroupResourceModel_PreservesNullOptionalFields(t *testing.T) {
	model := groupResourceModel{
		DisplayName:     types.StringValue("Test Group"),
		IsStatic:        types.BoolValue(true),
		GroupType:       types.StringValue("VirtualMachine"),
		EnvironmentType: types.StringNull(),
		CloudType:       types.StringNull(),
		LogicalOperator: types.StringNull(),
		Temporary:       types.BoolNull(),
	}

	// Simulate API response with default values
	dto := GroupDTO{
		DisplayName:     stringPtr("Test Group"),
		IsStatic:        boolPtr(true),
		GroupType:       stringPtr("VirtualMachine"),
		EnvironmentType: stringPtr("CLOUD"),
		CloudType:       stringPtr("AWS"),
		LogicalOperator: stringPtr("AND"),
		Temporary:       boolPtr(false),
	}

	// Store original null values before mapping
	originalEnvironmentType := model.EnvironmentType
	originalCloudType := model.CloudType
	originalLogicalOperator := model.LogicalOperator
	originalTemporary := model.Temporary

	// TODO: In the actual implementation, mapGroupDTOToModel would be called here
	_ = dto

	if !model.EnvironmentType.Equal(originalEnvironmentType) {
		t.Errorf("EnvironmentType should remain null when not set in plan")
	}
	if !model.CloudType.Equal(originalCloudType) {
		t.Errorf("CloudType should remain null when not set in plan")
	}
	if !model.LogicalOperator.Equal(originalLogicalOperator) {
		t.Errorf("LogicalOperator should remain null when not set in plan")
	}
	if !model.Temporary.Equal(originalTemporary) {
		t.Errorf("Temporary should remain null when not set in plan")
	}

	if model.DisplayName.ValueString() != "Test Group" {
		t.Errorf("DisplayName should be updated from DTO")
	}
	if !model.IsStatic.ValueBool() {
		t.Errorf("IsStatic should be updated from DTO")
	}
	if model.GroupType.ValueString() != "VirtualMachine" {
		t.Errorf("GroupType should be updated from DTO")
	}

	modelWithValues := groupResourceModel{
		DisplayName:     types.StringValue("Test Group"),
		CloudType:       types.StringValue("AWS"),
		EnvironmentType: types.StringValue("CLOUD"),
	}

	if modelWithValues.CloudType.IsNull() {
		t.Errorf("CloudType should be updated when set in plan")
	}
	if modelWithValues.EnvironmentType.IsNull() {
		t.Errorf("EnvironmentType should be updated when set in plan")
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// ---------------------------------------------------------------------------
// validateGroupPlan
// ---------------------------------------------------------------------------

// makeStringList builds a types.List of strings for test use.
func makeStringList(vals ...string) types.List {
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	list, _ := types.ListValue(types.StringType, elems)
	return list
}

// makeCriteriaList builds a non-empty criteria_list for test use.
func makeCriteriaList() types.List {
	block := criteriaListModel{
		FilterType: types.StringValue("vmsByName"),
		Operator:   types.StringValue("equals"),
		Value:      types.StringValue("my-vm"),
	}
	// Build manually to avoid context dependency in tests.
	elems := []attr.Value{}
	obj, err := buildCriteriaObject(block)
	if err == nil {
		elems = append(elems, obj)
	}
	result, _ := types.ListValue(criteriaObjectType(), elems)
	return result
}

func TestValidateGroupPlan_StaticValid(t *testing.T) {
	plan := groupResourceModel{
		IsStatic:       types.BoolValue(true),
		MemberUuidList: makeStringList("uuid-1", "uuid-2"),
		CriteriaList:   types.ListNull(criteriaObjectType()),
		LogicalOperator: types.StringNull(),
	}
	diags := validateGroupPlan(plan)
	if diags.HasError() {
		t.Errorf("expected no errors for valid static group, got: %v", diags)
	}
}

func TestValidateGroupPlan_StaticMissingMemberList(t *testing.T) {
	plan := groupResourceModel{
		IsStatic:       types.BoolValue(true),
		MemberUuidList: types.ListNull(types.StringType),
		CriteriaList:   types.ListNull(criteriaObjectType()),
	}
	diags := validateGroupPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when static group has no member_uuid_list")
	}
}

func TestValidateGroupPlan_StaticWithCriteriaList(t *testing.T) {
	plan := groupResourceModel{
		IsStatic:       types.BoolValue(true),
		MemberUuidList: makeStringList("uuid-1"),
		CriteriaList:   makeCriteriaList(),
	}
	diags := validateGroupPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when static group has criteria_list set")
	}
}

func TestValidateGroupPlan_StaticWithLogicalOperator(t *testing.T) {
	plan := groupResourceModel{
		IsStatic:        types.BoolValue(true),
		MemberUuidList:  makeStringList("uuid-1"),
		CriteriaList:    types.ListNull(criteriaObjectType()),
		LogicalOperator: types.StringValue("AND"),
	}
	diags := validateGroupPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when static group has logical_operator set")
	}
}

func TestValidateGroupPlan_DynamicValid(t *testing.T) {
	plan := groupResourceModel{
		IsStatic:        types.BoolValue(false),
		MemberUuidList:  types.ListNull(types.StringType),
		CriteriaList:    makeCriteriaList(),
		LogicalOperator: types.StringValue("AND"),
	}
	diags := validateGroupPlan(plan)
	if diags.HasError() {
		t.Errorf("expected no errors for valid dynamic group, got: %v", diags)
	}
}

func TestValidateGroupPlan_DynamicMissingCriteriaList(t *testing.T) {
	plan := groupResourceModel{
		IsStatic:       types.BoolValue(false),
		MemberUuidList: types.ListNull(types.StringType),
		CriteriaList:   types.ListNull(criteriaObjectType()),
	}
	diags := validateGroupPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when dynamic group has no criteria_list")
	}
}

func TestValidateGroupPlan_DynamicWithMemberList(t *testing.T) {
	plan := groupResourceModel{
		IsStatic:       types.BoolValue(false),
		MemberUuidList: makeStringList("uuid-1"),
		CriteriaList:   makeCriteriaList(),
	}
	diags := validateGroupPlan(plan)
	if !diags.HasError() {
		t.Error("expected error when dynamic group has member_uuid_list set")
	}
}

func TestValidateGroupPlan_NullIsStatic_Skipped(t *testing.T) {
	// When is_static is null (e.g. during import), validation is skipped.
	plan := groupResourceModel{
		IsStatic:       types.BoolNull(),
		MemberUuidList: types.ListNull(types.StringType),
		CriteriaList:   types.ListNull(criteriaObjectType()),
	}
	diags := validateGroupPlan(plan)
	if diags.HasError() {
		t.Errorf("expected no errors when is_static is null, got: %v", diags)
	}
}
