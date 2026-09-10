// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// placementPolicyItemAttrTypes
// ---------------------------------------------------------------------------

func TestPlacementPolicyItemAttrTypes_Complete(t *testing.T) {
	m := placementPolicyItemAttrTypes()
	expected := []string{
		"uuid", "name", "type", "enabled",
		"buyer_group_uuid", "seller_group_uuid",
		"merge_uuids", "merge_type", "capacity",
		"provider_entity_type", "enable_create_resource_pool",
	}
	for _, key := range expected {
		assert.Contains(t, m, key, "placementPolicyItemAttrTypes should contain %q", key)
	}
	assert.Len(t, m, len(expected))
}

// ---------------------------------------------------------------------------
// buildPlacementPolicyItemObject
// ---------------------------------------------------------------------------

func TestBuildPlacementPolicyItemObject_NonMerge(t *testing.T) {
	buyerUUID := "buyer-uuid-1"
	sellerUUID := "seller-uuid-1"
	p := &placementPolicyDTO{
		UUID:    strPtr("policy-uuid-1"),
		Name:    strPtr("bind-prod-vms"),
		Type:    "BIND_TO_GROUP",
		Enabled: boolPtr(true),
		ConsumerGroup: &baseRefDTO{UUID: &buyerUUID},
		ProviderGroup: &baseRefDTO{UUID: &sellerUUID},
	}

	obj, diags := buildPlacementPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	// Decode back via ObjectValue.
	attrs := obj.(types.Object).Attributes()
	assert.Equal(t, types.StringValue("policy-uuid-1"), attrs["uuid"])
	assert.Equal(t, types.StringValue("bind-prod-vms"), attrs["name"])
	assert.Equal(t, types.StringValue("BIND_TO_GROUP"), attrs["type"])
	assert.Equal(t, types.BoolValue(true), attrs["enabled"])
	assert.Equal(t, types.StringValue("buyer-uuid-1"), attrs["buyer_group_uuid"])
	assert.Equal(t, types.StringValue("seller-uuid-1"), attrs["seller_group_uuid"])
	assert.True(t, attrs["merge_type"].(types.String).IsNull())
	assert.True(t, attrs["capacity"].(types.Int64).IsNull())

	// merge_uuids should be an empty list (not null).
	mergeUUIDs := attrs["merge_uuids"].(types.List)
	assert.False(t, mergeUUIDs.IsNull())
	assert.Equal(t, 0, len(mergeUUIDs.Elements()))
}

func TestBuildPlacementPolicyItemObject_Merge(t *testing.T) {
	m1, m2 := "merge-uuid-1", "merge-uuid-2"
	mt := "Cluster"
	p := &placementPolicyDTO{
		UUID:      strPtr("policy-uuid-2"),
		Name:      strPtr("merge-clusters"),
		Type:      "MERGE",
		Enabled:   boolPtr(true),
		MergeType: &mt,
		MergeGroups: []baseRefDTO{
			{UUID: &m1},
			{UUID: &m2},
		},
	}

	obj, diags := buildPlacementPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	attrs := obj.(types.Object).Attributes()
	assert.Equal(t, types.StringValue("MERGE"), attrs["type"])
	assert.Equal(t, types.StringValue("Cluster"), attrs["merge_type"])
	assert.True(t, attrs["buyer_group_uuid"].(types.String).IsNull())
	assert.True(t, attrs["seller_group_uuid"].(types.String).IsNull())

	mergeUUIDs := attrs["merge_uuids"].(types.List)
	assert.Equal(t, 2, len(mergeUUIDs.Elements()))
}

func TestBuildPlacementPolicyItemObject_AtMostN(t *testing.T) {
	cap := int64(3)
	pet := "PhysicalMachine"
	p := &placementPolicyDTO{
		UUID:               strPtr("policy-uuid-3"),
		Name:               strPtr("at-most-3"),
		Type:               "AT_MOST_N",
		Enabled:            boolPtr(false),
		Capacity:           &cap,
		ProviderEntityType: &pet,
		ConsumerGroup:      &baseRefDTO{UUID: strPtr("buyer-uuid")},
		ProviderGroup:      &baseRefDTO{UUID: strPtr("seller-uuid")},
	}

	obj, diags := buildPlacementPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	attrs := obj.(types.Object).Attributes()
	assert.Equal(t, types.BoolValue(false), attrs["enabled"])
	assert.Equal(t, types.Int64Value(3), attrs["capacity"])
	assert.Equal(t, types.StringValue("PhysicalMachine"), attrs["provider_entity_type"])
}

func TestBuildPlacementPolicyItemObject_NullOptionals(t *testing.T) {
	// A minimal DTO with only required fields populated.
	p := &placementPolicyDTO{
		Type: "MUST_NOT_RUN_TOGETHER",
	}

	obj, diags := buildPlacementPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	attrs := obj.(types.Object).Attributes()
	assert.True(t, attrs["uuid"].(types.String).IsNull())
	assert.True(t, attrs["name"].(types.String).IsNull())
	assert.True(t, attrs["enabled"].(types.Bool).IsNull())
	assert.True(t, attrs["buyer_group_uuid"].(types.String).IsNull())
	assert.True(t, attrs["seller_group_uuid"].(types.String).IsNull())
	assert.True(t, attrs["merge_type"].(types.String).IsNull())
	assert.True(t, attrs["capacity"].(types.Int64).IsNull())
	assert.True(t, attrs["provider_entity_type"].(types.String).IsNull())
	assert.True(t, attrs["enable_create_resource_pool"].(types.Bool).IsNull())
	assert.Equal(t, types.StringValue("MUST_NOT_RUN_TOGETHER"), attrs["type"])
}

func TestBuildPlacementPolicyItemObject_EnableCreateResourcePool(t *testing.T) {
	ecrp := true
	mt := "Cluster"
	m1 := "merge-uuid-1"
	p := &placementPolicyDTO{
		UUID:                     strPtr("policy-uuid-4"),
		Name:                     strPtr("merge-with-pool"),
		Type:                     "MERGE",
		Enabled:                  boolPtr(true),
		MergeType:                &mt,
		EnableCreateResourcePool: &ecrp,
		MergeGroups:              []baseRefDTO{{UUID: &m1}},
	}

	obj, diags := buildPlacementPolicyItemObject(p)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	attrs := obj.(types.Object).Attributes()
	assert.Equal(t, types.BoolValue(true), attrs["enable_create_resource_pool"])
}
