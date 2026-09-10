// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// criteriaAttrTypes / criteriaObjectType
// ---------------------------------------------------------------------------

func TestCriteriaAttrTypes_Complete(t *testing.T) {
	m := criteriaAttrTypes()
	expected := []string{"filter_entity", "filter_field", "filter_type", "operator", "value", "case_sensitive"}
	for _, key := range expected {
		assert.Contains(t, m, key, "criteriaAttrTypes should contain %q", key)
	}
	assert.Len(t, m, len(expected))
}

// ---------------------------------------------------------------------------
// resolveCriteriaToList helpers
// ---------------------------------------------------------------------------

func mustBuildTestCriteriaList(t *testing.T, blocks []criteriaListModel) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(context.Background(), criteriaObjectType(), blocks)
	if diags.HasError() {
		t.Fatalf("building test criteria list: %v", diags)
	}
	return list
}

func mustDecodeCriteriaList(t *testing.T, list types.List) []criteriaListModel {
	t.Helper()
	var out []criteriaListModel
	if diags := list.ElementsAs(context.Background(), &out, false); diags.HasError() {
		t.Fatalf("decoding criteria list: %v", diags)
	}
	return out
}

// ---------------------------------------------------------------------------
// resolveCriteriaToList
// ---------------------------------------------------------------------------

func TestResolveCriteriaToList_NullInput(t *testing.T) {
	result, diags := resolveCriteriaToList(context.Background(), types.ListNull(criteriaObjectType()))
	assert.False(t, diags.HasError())
	assert.Equal(t, 0, len(result.Elements()))
}

func TestResolveCriteriaToList_EmptyList(t *testing.T) {
	list := types.ListValueMust(criteriaObjectType(), nil)
	result, diags := resolveCriteriaToList(context.Background(), list)
	assert.False(t, diags.HasError())
	assert.Equal(t, 0, len(result.Elements()))
}

func TestResolveCriteriaToList_ShorthandResolved(t *testing.T) {
	// filter_entity=vm + filter_field=name should resolve filter_type=vmsByName
	blocks := []criteriaListModel{
		{
			FilterEntity:  types.StringValue("vm"),
			FilterField:   types.StringValue("name"),
			FilterType:    types.StringNull(),
			Operator:      types.StringValue("equals"),
			Value:         types.StringValue("my-vm"),
			CaseSensitive: types.BoolNull(),
		},
	}
	list := mustBuildTestCriteriaList(t, blocks)
	result, diags := resolveCriteriaToList(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("resolveCriteriaToList: %v", diags)
	}
	assert.Equal(t, 1, len(result.Elements()))

	resolved := mustDecodeCriteriaList(t, result)
	assert.Equal(t, "vmsByName", resolved[0].FilterType.ValueString())
}

func TestResolveCriteriaToList_ExplicitFilterTypePreserved(t *testing.T) {
	// If filter_type is already set it must not be overwritten by shorthand lookup.
	blocks := []criteriaListModel{
		{
			FilterEntity:  types.StringNull(),
			FilterField:   types.StringNull(),
			FilterType:    types.StringValue("customFilterType"),
			Operator:      types.StringValue("equals"),
			Value:         types.StringValue("foo"),
			CaseSensitive: types.BoolNull(),
		},
	}
	list := mustBuildTestCriteriaList(t, blocks)
	result, diags := resolveCriteriaToList(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("resolveCriteriaToList: %v", diags)
	}
	resolved := mustDecodeCriteriaList(t, result)
	assert.Equal(t, "customFilterType", resolved[0].FilterType.ValueString())
}

func TestResolveCriteriaToList_MultipleBlocks(t *testing.T) {
	blocks := []criteriaListModel{
		{
			FilterEntity:  types.StringValue("vm"),
			FilterField:   types.StringValue("name"),
			FilterType:    types.StringNull(),
			Operator:      types.StringValue("regex"),
			Value:         types.StringValue("^prod-"),
			CaseSensitive: types.BoolNull(),
		},
		{
			FilterEntity:  types.StringValue("vm"),
			FilterField:   types.StringValue("guest_os"),
			FilterType:    types.StringNull(),
			Operator:      types.StringValue("equals"),
			Value:         types.StringValue("Linux"),
			CaseSensitive: types.BoolNull(),
		},
	}
	list := mustBuildTestCriteriaList(t, blocks)
	result, diags := resolveCriteriaToList(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("resolveCriteriaToList: %v", diags)
	}
	assert.Equal(t, 2, len(result.Elements()))

	resolved := mustDecodeCriteriaList(t, result)
	assert.Equal(t, "vmsByName", resolved[0].FilterType.ValueString())
	assert.Equal(t, "vmsByGuestName", resolved[1].FilterType.ValueString())
}

func TestResolveCriteriaToList_UnknownShorthandKeepsNullFilterType(t *testing.T) {
	// An unknown filter_field → shorthand lookup fails → filter_type stays null.
	// This is not an error at this layer; validation happens elsewhere.
	blocks := []criteriaListModel{
		{
			FilterEntity:  types.StringValue("vm"),
			FilterField:   types.StringValue("nonexistent"),
			FilterType:    types.StringNull(),
			Operator:      types.StringValue("equals"),
			Value:         types.StringValue("x"),
			CaseSensitive: types.BoolNull(),
		},
	}
	list := mustBuildTestCriteriaList(t, blocks)
	result, diags := resolveCriteriaToList(context.Background(), list)
	assert.False(t, diags.HasError())

	resolved := mustDecodeCriteriaList(t, result)
	assert.True(t, resolved[0].FilterType.IsNull())
}

func TestResolveCriteriaToList_ValuesPreserved(t *testing.T) {
	blocks := []criteriaListModel{
		{
			FilterEntity:  types.StringValue("pm"),
			FilterField:   types.StringValue("name"),
			FilterType:    types.StringNull(),
			Operator:      types.StringValue("regex"),
			Value:         types.StringValue("^esxi-"),
			CaseSensitive: types.BoolValue(true),
		},
	}
	list := mustBuildTestCriteriaList(t, blocks)
	result, diags := resolveCriteriaToList(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("resolveCriteriaToList: %v", diags)
	}
	resolved := mustDecodeCriteriaList(t, result)
	assert.Equal(t, "pmsByName", resolved[0].FilterType.ValueString())
	assert.Equal(t, "regex", resolved[0].Operator.ValueString())
	assert.Equal(t, "^esxi-", resolved[0].Value.ValueString())
	assert.Equal(t, true, resolved[0].CaseSensitive.ValueBool())
}

// ---------------------------------------------------------------------------
// buildCriteriaObject
// ---------------------------------------------------------------------------

func TestBuildCriteriaObject_AllFields(t *testing.T) {
	m := criteriaListModel{
		FilterEntity:  types.StringValue("vm"),
		FilterField:   types.StringValue("name"),
		FilterType:    types.StringValue("vmsByName"),
		Operator:      types.StringValue("equals"),
		Value:         types.StringValue("my-vm"),
		CaseSensitive: types.BoolValue(true),
	}
	obj, err := buildCriteriaObject(m)
	assert.NoError(t, err)
	assert.False(t, obj.IsNull())
	assert.False(t, obj.IsUnknown())

	attrs := obj.Attributes()
	assert.Equal(t, types.StringValue("vm"), attrs["filter_entity"])
	assert.Equal(t, types.StringValue("name"), attrs["filter_field"])
	assert.Equal(t, types.StringValue("vmsByName"), attrs["filter_type"])
	assert.Equal(t, types.StringValue("equals"), attrs["operator"])
	assert.Equal(t, types.StringValue("my-vm"), attrs["value"])
	assert.Equal(t, types.BoolValue(true), attrs["case_sensitive"])
}

func TestBuildCriteriaObject_NullOptionals(t *testing.T) {
	m := criteriaListModel{
		FilterEntity:  types.StringNull(),
		FilterField:   types.StringNull(),
		FilterType:    types.StringValue("vmsByName"),
		Operator:      types.StringValue("equals"),
		Value:         types.StringNull(),
		CaseSensitive: types.BoolNull(),
	}
	obj, err := buildCriteriaObject(m)
	assert.NoError(t, err)
	attrs := obj.Attributes()
	assert.True(t, attrs["filter_entity"].(types.String).IsNull())
	assert.True(t, attrs["value"].(types.String).IsNull())
	assert.True(t, attrs["case_sensitive"].(types.Bool).IsNull())
}
