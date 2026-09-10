// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ── roleAttrTypes ───────────────────────────────────────────────────────────

func TestRoleAttrTypes_Complete(t *testing.T) {
	attrTypes := roleAttrTypes()
	expected := []string{"uuid", "name", "display_name", "description"}
	for _, k := range expected {
		if _, ok := attrTypes[k]; !ok {
			t.Errorf("missing key %q in roleAttrTypes", k)
		}
	}
	if len(attrTypes) != len(expected) {
		t.Errorf("roleAttrTypes has %d keys, want %d", len(attrTypes), len(expected))
	}
}

// ── userItemAttrTypes ────────────────────────────────────────────────────────

func TestUserItemAttrTypes_Complete(t *testing.T) {
	attrTypes := userItemAttrTypes()
	expected := []string{"uuid", "username", "display_name", "login_provider", "type", "roles", "scope_group_uuids"}
	for _, k := range expected {
		if _, ok := attrTypes[k]; !ok {
			t.Errorf("missing key %q in userItemAttrTypes", k)
		}
	}
	if len(attrTypes) != len(expected) {
		t.Errorf("userItemAttrTypes has %d keys, want %d", len(attrTypes), len(expected))
	}
}

// ── buildRoleObject ──────────────────────────────────────────────────────────

func TestBuildRoleObject_AllFields(t *testing.T) {
	uuid := "role-uuid-1"
	name := "ADMINISTRATOR"
	dn := "Administrator"
	desc := "Full access"
	r := &roleDTO{
		Uuid:        &uuid,
		Name:        &name,
		DisplayName: &dn,
		Description: &desc,
	}
	obj, diags := buildRoleObject(r)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	attrs := obj.(types.Object).Attributes()

	if attrs["uuid"].(types.String).ValueString() != "role-uuid-1" {
		t.Errorf("expected uuid role-uuid-1, got %v", attrs["uuid"])
	}
	if attrs["name"].(types.String).ValueString() != "ADMINISTRATOR" {
		t.Errorf("expected name ADMINISTRATOR, got %v", attrs["name"])
	}
	if attrs["display_name"].(types.String).ValueString() != "Administrator" {
		t.Errorf("expected display_name Administrator, got %v", attrs["display_name"])
	}
	if attrs["description"].(types.String).ValueString() != "Full access" {
		t.Errorf("expected description 'Full access', got %v", attrs["description"])
	}
}

func TestBuildRoleObject_NilFields(t *testing.T) {
	r := &roleDTO{}
	obj, diags := buildRoleObject(r)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	attrs := obj.(types.Object).Attributes()

	for _, key := range []string{"uuid", "name", "display_name", "description"} {
		if !attrs[key].(types.String).IsNull() {
			t.Errorf("expected null %s for nil pointer, got %v", key, attrs[key])
		}
	}
}

// ── buildUserItemObject ──────────────────────────────────────────────────────

func TestBuildUserItemObject_AllFields(t *testing.T) {
	uuid := "user-uuid-1"
	username := "jdoe"
	dn := "John Doe"
	lp := "Local"
	ut := "DedicatedCustomer"
	scopeUUID := "group-uuid-1"
	roleUUID := "role-uuid-1"
	roleName := "OBSERVER"

	u := &userDTO{
		Uuid:          &uuid,
		Username:      &username,
		DisplayName:   &dn,
		LoginProvider: &lp,
		Type:          &ut,
		Roles: []roleDTO{
			{Uuid: &roleUUID, Name: &roleName},
		},
		Scope: []scopeEntry{
			{Uuid: &scopeUUID},
		},
	}

	obj, diags := buildUserItemObject(u)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	attrs := obj.(types.Object).Attributes()

	if attrs["uuid"].(types.String).ValueString() != "user-uuid-1" {
		t.Errorf("expected uuid user-uuid-1, got %v", attrs["uuid"])
	}
	if attrs["username"].(types.String).ValueString() != "jdoe" {
		t.Errorf("expected username jdoe, got %v", attrs["username"])
	}
	if attrs["display_name"].(types.String).ValueString() != "John Doe" {
		t.Errorf("expected display_name 'John Doe', got %v", attrs["display_name"])
	}
	if attrs["login_provider"].(types.String).ValueString() != "Local" {
		t.Errorf("expected login_provider Local, got %v", attrs["login_provider"])
	}
	if attrs["type"].(types.String).ValueString() != "DedicatedCustomer" {
		t.Errorf("expected type DedicatedCustomer, got %v", attrs["type"])
	}

	rolesList := attrs["roles"].(types.List)
	if len(rolesList.Elements()) != 1 {
		t.Fatalf("expected 1 role, got %d", len(rolesList.Elements()))
	}
	roleAttrs := rolesList.Elements()[0].(types.Object).Attributes()
	if roleAttrs["name"].(types.String).ValueString() != "OBSERVER" {
		t.Errorf("expected role name OBSERVER, got %v", roleAttrs["name"])
	}

	scopeList := attrs["scope_group_uuids"].(types.List)
	if len(scopeList.Elements()) != 1 {
		t.Fatalf("expected 1 scope group uuid, got %d", len(scopeList.Elements()))
	}
	if scopeList.Elements()[0].(types.String).ValueString() != "group-uuid-1" {
		t.Errorf("expected scope group-uuid-1, got %v", scopeList.Elements()[0])
	}
}

func TestBuildUserItemObject_NullOptionals(t *testing.T) {
	u := &userDTO{}
	obj, diags := buildUserItemObject(u)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	attrs := obj.(types.Object).Attributes()

	for _, key := range []string{"uuid", "username", "display_name", "login_provider", "type"} {
		if !attrs[key].(types.String).IsNull() {
			t.Errorf("expected null %s for nil pointer, got %v", key, attrs[key])
		}
	}

	// roles and scope_group_uuids should be empty lists, not null
	if len(attrs["roles"].(types.List).Elements()) != 0 {
		t.Errorf("expected empty roles list, got %v", attrs["roles"])
	}
	if len(attrs["scope_group_uuids"].(types.List).Elements()) != 0 {
		t.Errorf("expected empty scope_group_uuids list, got %v", attrs["scope_group_uuids"])
	}
}

func TestBuildUserItemObject_MultipleRoles(t *testing.T) {
	name1 := "ADMINISTRATOR"
	name2 := "ADVISOR"
	u := &userDTO{
		Roles: []roleDTO{
			{Name: &name1},
			{Name: &name2},
		},
	}
	obj, diags := buildUserItemObject(u)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	rolesList := obj.(types.Object).Attributes()["roles"].(types.List)
	if len(rolesList.Elements()) != 2 {
		t.Errorf("expected 2 roles, got %d", len(rolesList.Elements()))
	}
}

func TestBuildUserItemObject_ScopeNilUUID(t *testing.T) {
	// Scope entries with nil uuid should be skipped.
	u := &userDTO{
		Scope: []scopeEntry{
			{Uuid: nil},
			{Uuid: strPtr("group-uuid-2")},
		},
	}
	obj, diags := buildUserItemObject(u)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	scopeList := obj.(types.Object).Attributes()["scope_group_uuids"].(types.List)
	if len(scopeList.Elements()) != 1 {
		t.Errorf("expected 1 non-nil scope uuid, got %d", len(scopeList.Elements()))
	}
	if scopeList.Elements()[0].(types.String).ValueString() != "group-uuid-2" {
		t.Errorf("unexpected scope uuid: %v", scopeList.Elements()[0])
	}
}

// ── userHasRole ───────────────────────────────────────────────────────────────

func TestUserHasRole_Match(t *testing.T) {
	name := "ADMINISTRATOR"
	u := &userDTO{
		Roles: []roleDTO{{Name: &name}},
	}
	if !userHasRole(u, "ADMINISTRATOR") {
		t.Error("expected userHasRole to return true for matching role")
	}
}

func TestUserHasRole_NoMatch(t *testing.T) {
	name := "OBSERVER"
	u := &userDTO{
		Roles: []roleDTO{{Name: &name}},
	}
	if userHasRole(u, "ADMINISTRATOR") {
		t.Error("expected userHasRole to return false for non-matching role")
	}
}

func TestUserHasRole_Empty(t *testing.T) {
	u := &userDTO{}
	if userHasRole(u, "ADMINISTRATOR") {
		t.Error("expected userHasRole to return false for empty roles")
	}
}

func TestUserHasRole_NilRoleName(t *testing.T) {
	u := &userDTO{
		Roles: []roleDTO{{Name: nil}},
	}
	if userHasRole(u, "ADMINISTRATOR") {
		t.Error("expected userHasRole to return false for nil role name")
	}
}

// ── compile-time interface checks ────────────────────────────────────────────

var _ attr.Value = types.ObjectNull(userItemAttrTypes())
var _ attr.Value = types.ObjectNull(roleAttrTypes())

// strPtr is a test helper that returns a pointer to the given string value.
func strPtr(s string) *string { return &s }
