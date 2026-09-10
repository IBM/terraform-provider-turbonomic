// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// settingsPolicyItemAttrTypes
// ---------------------------------------------------------------------------

func TestSettingsPolicyItemAttrTypes_Complete(t *testing.T) {
	m := settingsPolicyItemAttrTypes()
	expected := []string{
		"uuid", "name", "entity_type", "disabled",
		"default", "read_only", "note", "scope_uuids", "schedule_uuid",
	}
	for _, key := range expected {
		assert.Contains(t, m, key, "settingsPolicyItemAttrTypes should contain %q", key)
	}
	assert.Len(t, m, len(expected))
}

// ---------------------------------------------------------------------------
// buildSettingsPolicyItemObject
// ---------------------------------------------------------------------------

func TestBuildSettingsPolicyItemObject_AllFields(t *testing.T) {
	trueVal := true
	falseVal := false
	p := &settingsPolicyDTO{
		UUID:        strPtr("policy-uuid-1"),
		DisplayName: strPtr("vm-automation"),
		EntityType:  strPtr("VirtualMachine"),
		Disabled:    &falseVal,
		Default:     &falseVal,
		ReadOnly:    &trueVal,
		Note:        strPtr("managed by terraform"),
		Schedule:    &settingsScheduleDTO{UUID: "sched-uuid-1"},
		Scopes:      []settingsScopeDTO{{UUID: "scope-uuid-1"}, {UUID: "scope-uuid-2"}},
	}

	obj, diags := buildSettingsPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	attrs := obj.(types.Object).Attributes()
	assert.Equal(t, types.StringValue("policy-uuid-1"), attrs["uuid"])
	assert.Equal(t, types.StringValue("vm-automation"), attrs["name"])
	assert.Equal(t, types.StringValue("VirtualMachine"), attrs["entity_type"])
	assert.Equal(t, types.BoolValue(false), attrs["disabled"])
	assert.Equal(t, types.BoolValue(false), attrs["default"])
	assert.Equal(t, types.BoolValue(true), attrs["read_only"])
	assert.Equal(t, types.StringValue("managed by terraform"), attrs["note"])
	assert.Equal(t, types.StringValue("sched-uuid-1"), attrs["schedule_uuid"])

	scopeUUIDs := attrs["scope_uuids"].(types.List)
	assert.Equal(t, 2, len(scopeUUIDs.Elements()))
}

func TestBuildSettingsPolicyItemObject_NullOptionals(t *testing.T) {
	// A minimal DTO with only uuid populated.
	p := &settingsPolicyDTO{
		UUID: strPtr("policy-uuid-2"),
	}

	obj, diags := buildSettingsPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	attrs := obj.(types.Object).Attributes()
	assert.Equal(t, types.StringValue("policy-uuid-2"), attrs["uuid"])
	assert.True(t, attrs["name"].(types.String).IsNull())
	assert.True(t, attrs["entity_type"].(types.String).IsNull())
	assert.True(t, attrs["disabled"].(types.Bool).IsNull())
	assert.True(t, attrs["default"].(types.Bool).IsNull())
	assert.True(t, attrs["read_only"].(types.Bool).IsNull())
	assert.True(t, attrs["note"].(types.String).IsNull())
	assert.True(t, attrs["schedule_uuid"].(types.String).IsNull())

	scopeUUIDs := attrs["scope_uuids"].(types.List)
	assert.False(t, scopeUUIDs.IsNull())
	assert.Equal(t, 0, len(scopeUUIDs.Elements()))
}

func TestBuildSettingsPolicyItemObject_NoSchedule(t *testing.T) {
	p := &settingsPolicyDTO{
		UUID:        strPtr("policy-uuid-3"),
		DisplayName: strPtr("global-defaults"),
		EntityType:  strPtr("PhysicalMachine"),
	}

	obj, diags := buildSettingsPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	attrs := obj.(types.Object).Attributes()
	assert.True(t, attrs["schedule_uuid"].(types.String).IsNull())
}

func TestBuildSettingsPolicyItemObject_GlobalPolicy(t *testing.T) {
	// Global policy: no scopes - scope_uuids should be an empty list.
	p := &settingsPolicyDTO{
		UUID:       strPtr("policy-uuid-4"),
		EntityType: strPtr("Container"),
		Scopes:     []settingsScopeDTO{},
	}

	obj, diags := buildSettingsPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	attrs := obj.(types.Object).Attributes()
	scopeUUIDs := attrs["scope_uuids"].(types.List)
	assert.False(t, scopeUUIDs.IsNull())
	assert.Equal(t, 0, len(scopeUUIDs.Elements()))
}

func TestBuildSettingsPolicyItemObject_MultipleScopes(t *testing.T) {
	p := &settingsPolicyDTO{
		UUID:   strPtr("policy-uuid-5"),
		Scopes: []settingsScopeDTO{{UUID: "g1"}, {UUID: "g2"}, {UUID: "g3"}},
	}

	obj, diags := buildSettingsPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	attrs := obj.(types.Object).Attributes()
	scopeUUIDs := attrs["scope_uuids"].(types.List)
	assert.Equal(t, 3, len(scopeUUIDs.Elements()))
}
