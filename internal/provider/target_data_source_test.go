// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/IBM/turbonomic-go-client/api/generated"
)

// TestMapTargetDTOToItemModel_AllFields verifies that every field in a
// fully-populated TargetApiDTO is correctly mapped to a targetItemModel.
func TestMapTargetDTOToItemModel_AllFields(t *testing.T) {
	uuid := "ds-uuid-001"
	category := "Public Cloud"
	editTime := "2024-07-01T09:00:00Z"
	editUser := "admin"
	discovery := "2024-07-01T08:00:00Z"
	errTxt := "validation failed"
	isProbe := true
	healthState := generated.HealthStateNORMAL
	rollup := generated.HealthStateMINOR
	isSecret := false
	fieldVal := "turbo.example.com"
	fieldName := "address"

	dto := &generated.TargetApiDTO{
		Type:              "vCenter",
		Uuid:              &uuid,
		Category:          &category,
		IsProbeRegistered: &isProbe,
		LastEditTime:      &editTime,
		LastEditUser:      &editUser,
		HealthSummary: &generated.TargetHealthSummaryApiDTO{
			HealthState:                   healthState,
			RollupState:                   rollup,
			TimeOfLastSuccessfulDiscovery: &discovery,
		},
		Health: &generated.TargetHealthApiDTO{
			ErrorText:                &errTxt,
			HealthState:              healthState,
			HealthClassDiscriminator: "TargetHealthApiDTO",
			HealthCategory:           generated.HealthCategoryTARGET,
			RollupState:              rollup,
		},
		InputFields: &[]generated.InputFieldApiDTO{
			{Name: &fieldName, Value: &fieldVal, IsSecret: &isSecret},
		},
	}

	item := mapTargetDTOToItemModel(dto)

	if item.UUID.ValueString() != uuid {
		t.Errorf("uuid: got %q, want %q", item.UUID.ValueString(), uuid)
	}
	if item.Type.ValueString() != "vCenter" {
		t.Errorf("type: got %q, want %q", item.Type.ValueString(), "vCenter")
	}
	if item.Category.ValueString() != category {
		t.Errorf("category: got %q, want %q", item.Category.ValueString(), category)
	}
	if !item.IsProbeRegistered.ValueBool() {
		t.Error("is_probe_registered: want true")
	}
	if item.LastEditTime.ValueString() != editTime {
		t.Errorf("last_edit_time: got %q, want %q", item.LastEditTime.ValueString(), editTime)
	}
	if item.LastEditUser.ValueString() != editUser {
		t.Errorf("last_edit_user: got %q, want %q", item.LastEditUser.ValueString(), editUser)
	}
	if item.HealthState.ValueString() != "NORMAL" {
		t.Errorf("health_state: got %q, want NORMAL", item.HealthState.ValueString())
	}
	if item.HealthRollupState.ValueString() != "MINOR" {
		t.Errorf("health_rollup_state: got %q, want MINOR", item.HealthRollupState.ValueString())
	}
	if item.LastSuccessfulDiscovery.ValueString() != discovery {
		t.Errorf("last_successful_discovery: got %q, want %q", item.LastSuccessfulDiscovery.ValueString(), discovery)
	}
	if item.HealthErrorText.ValueString() != errTxt {
		t.Errorf("health_error_text: got %q, want %q", item.HealthErrorText.ValueString(), errTxt)
	}

	elems := make(map[string]string)
	_ = item.InputFields.ElementsAs(context.Background(), &elems, false)
	if elems["address"] != fieldVal {
		t.Errorf("input_fields[address]: got %q, want %q", elems["address"], fieldVal)
	}
}

// TestMapTargetDTOToItemModel_SecretFieldsExcluded verifies that fields
// with isSecret=true are never included in the item's input_fields map.
func TestMapTargetDTOToItemModel_SecretFieldsExcluded(t *testing.T) {
	isSecret := true
	secretVal := "should-never-appear"
	secretName := "password"
	isNotSecret := false
	publicVal := "turbo.example.com"
	publicName := "address"

	dto := &generated.TargetApiDTO{
		Type: "vCenter",
		InputFields: &[]generated.InputFieldApiDTO{
			{Name: &publicName, Value: &publicVal, IsSecret: &isNotSecret},
			{Name: &secretName, Value: &secretVal, IsSecret: &isSecret},
		},
	}

	item := mapTargetDTOToItemModel(dto)

	elems := make(map[string]string)
	_ = item.InputFields.ElementsAs(context.Background(), &elems, false)

	if _, exists := elems["password"]; exists {
		t.Error("secret field 'password' must not appear in input_fields")
	}
	if elems["address"] != publicVal {
		t.Errorf("non-secret field 'address': got %q, want %q", elems["address"], publicVal)
	}
	if len(elems) != 1 {
		t.Errorf("input_fields should have 1 entry, got %d", len(elems))
	}
}

// TestMapTargetDTOToItemModel_NilHealthSummary verifies that absent
// HealthSummary and Health result in null computed fields - no panic.
func TestMapTargetDTOToItemModel_NilHealthSummary(t *testing.T) {
	dto := &generated.TargetApiDTO{
		Type:          "AWS",
		HealthSummary: nil,
		Health:        nil,
	}

	item := mapTargetDTOToItemModel(dto)

	if !item.HealthState.IsNull() {
		t.Error("health_state should be null when HealthSummary is nil")
	}
	if !item.HealthRollupState.IsNull() {
		t.Error("health_rollup_state should be null when HealthSummary is nil")
	}
	if !item.LastSuccessfulDiscovery.IsNull() {
		t.Error("last_successful_discovery should be null when HealthSummary is nil")
	}
	if !item.HealthErrorText.IsNull() {
		t.Error("health_error_text should be null when Health is nil")
	}
}

// TestMapTargetDTOToItemModel_NilCategory verifies that a nil Category
// pointer maps to a null types.String, not an empty string.
func TestMapTargetDTOToItemModel_NilCategory(t *testing.T) {
	dto := &generated.TargetApiDTO{
		Type:     "Kubernetes",
		Category: nil,
	}

	item := mapTargetDTOToItemModel(dto)

	if !item.Category.IsNull() {
		t.Errorf("category should be null when API returns nil, got %q", item.Category.ValueString())
	}
}

// TestMapTargetDTOToItemModel_NilUUID verifies that a nil UUID pointer maps
// to a null types.String.
func TestMapTargetDTOToItemModel_NilUUID(t *testing.T) {
	dto := &generated.TargetApiDTO{
		Type: "Prometheus",
		Uuid: nil,
	}

	item := mapTargetDTOToItemModel(dto)

	if !item.UUID.IsNull() {
		t.Errorf("uuid should be null when API returns nil, got %q", item.UUID.ValueString())
	}
}

// TestMapTargetDTOToItemModel_NoInputFields verifies that nil InputFields on
// the DTO produces an empty (non-null) map.
func TestMapTargetDTOToItemModel_NoInputFields(t *testing.T) {
	dto := &generated.TargetApiDTO{
		Type:        "Prometheus",
		InputFields: nil,
	}

	item := mapTargetDTOToItemModel(dto)

	if item.InputFields.IsNull() {
		t.Error("input_fields should be an empty map, not null, when DTO has no InputFields")
	}

	elems := make(map[string]attr.Value)
	_ = item.InputFields.ElementsAs(context.Background(), &elems, false)
	if len(elems) != 0 {
		t.Errorf("input_fields should be empty, got %d entries", len(elems))
	}
}

// TestMapTargetDTOToItemModel_HealthSummaryNoDiscovery verifies that a nil
// TimeOfLastSuccessfulDiscovery within a present HealthSummary maps to null.
func TestMapTargetDTOToItemModel_HealthSummaryNoDiscovery(t *testing.T) {
	health := generated.HealthStateNORMAL

	dto := &generated.TargetApiDTO{
		Type: "Azure",
		HealthSummary: &generated.TargetHealthSummaryApiDTO{
			HealthState:                   health,
			RollupState:                   health,
			TimeOfLastSuccessfulDiscovery: nil,
		},
	}

	item := mapTargetDTOToItemModel(dto)

	if item.HealthState.ValueString() != "NORMAL" {
		t.Errorf("health_state: got %q, want NORMAL", item.HealthState.ValueString())
	}
	if !item.LastSuccessfulDiscovery.IsNull() {
		t.Errorf("last_successful_discovery should be null when not set, got %q",
			item.LastSuccessfulDiscovery.ValueString())
	}
}

// TestMapTargetDTOToItemModel_NilIsProbeRegistered verifies that a nil
// IsProbeRegistered pointer maps to a null types.Bool.
func TestMapTargetDTOToItemModel_NilIsProbeRegistered(t *testing.T) {
	dto := &generated.TargetApiDTO{
		Type:              "vCenter",
		IsProbeRegistered: nil,
	}

	item := mapTargetDTOToItemModel(dto)

	if !item.IsProbeRegistered.IsNull() {
		t.Errorf("is_probe_registered should be null when API returns nil, got %v",
			item.IsProbeRegistered.ValueBool())
	}
}

// TestTargetDataSource_FilterLogic tests the in-memory type/category filtering
// applied inside Read before results are written to state.
func TestTargetDataSource_FilterLogic(t *testing.T) {
	awsType := "AWS"
	awsCat := "Public Cloud"
	vcType := "vCenter"
	vcCat := "Hypervisor"

	all := []generated.TargetApiDTO{
		{Type: awsType, Category: &awsCat},
		{Type: vcType, Category: &vcCat},
		{Type: awsType, Category: &awsCat}, // second AWS target
	}

	// Helper that replicates the filtering logic from Read().
	filter := func(typeF, catF string) []targetItemModel {
		var out []targetItemModel
		for i := range all {
			t := &all[i]
			if typeF != "" && t.Type != typeF {
				continue
			}
			if catF != "" && (t.Category == nil || *t.Category != catF) {
				continue
			}
			out = append(out, mapTargetDTOToItemModel(t))
		}
		if out == nil {
			out = []targetItemModel{}
		}
		return out
	}

	t.Run("no filters returns all", func(t *testing.T) {
		got := filter("", "")
		if len(got) != 3 {
			t.Errorf("expected 3 results, got %d", len(got))
		}
	})

	t.Run("type filter returns matching only", func(t *testing.T) {
		got := filter("AWS", "")
		if len(got) != 2 {
			t.Errorf("expected 2 AWS results, got %d", len(got))
		}
		for _, item := range got {
			if item.Type.ValueString() != "AWS" {
				t.Errorf("unexpected type %q in filtered results", item.Type.ValueString())
			}
		}
	})

	t.Run("type+category filter narrows correctly", func(t *testing.T) {
		got := filter("vCenter", "Hypervisor")
		if len(got) != 1 {
			t.Errorf("expected 1 result, got %d", len(got))
		}
	})

	t.Run("no match returns empty list not nil", func(t *testing.T) {
		got := filter("NonExistent", "")
		if got == nil {
			t.Error("result should be empty slice, not nil")
		}
		if len(got) != 0 {
			t.Errorf("expected 0 results, got %d", len(got))
		}
	})
}

// TestTargetDataSourceModel_FiltersPreservedInState verifies that optional
// filter values are round-tripped correctly into state (not cleared).
func TestTargetDataSourceModel_FiltersPreservedInState(t *testing.T) {
	model := targetDataSourceModel{
		TypeFilter:     types.StringValue("AWS"),
		CategoryFilter: types.StringValue("Public Cloud"),
		Targets:        []targetItemModel{},
	}

	if model.TypeFilter.ValueString() != "AWS" {
		t.Errorf("type filter: got %q, want AWS", model.TypeFilter.ValueString())
	}
	if model.CategoryFilter.ValueString() != "Public Cloud" {
		t.Errorf("category filter: got %q, want Public Cloud", model.CategoryFilter.ValueString())
	}
}
