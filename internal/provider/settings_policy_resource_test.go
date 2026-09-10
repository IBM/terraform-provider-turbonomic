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
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newSettingsResource() *settingsPolicyResource {
	return &settingsPolicyResource{}
}

// buildTestSettingsManagersList builds a types.List of settingsManagerModel for use in plans.
func buildTestSettingsManagersList(t *testing.T, category string, settings []settingModel) types.List {
	t.Helper()
	ctx := context.Background()

	settingElems := make([]attr.Value, 0, len(settings))
	for _, s := range settings {
		obj, d := types.ObjectValue(settingAttrTypes, map[string]attr.Value{
			"uuid":  s.UUID,
			"value": s.Value,
		})
		if d.HasError() {
			t.Fatalf("failed to build setting object: %v", d)
		}
		settingElems = append(settingElems, obj)
	}
	settingsList, d := types.ListValue(settingObjType, settingElems)
	if d.HasError() {
		t.Fatalf("failed to build settings list: %v", d)
	}

	managerObj, d := types.ObjectValue(managerAttrTypes, map[string]attr.Value{
		"category": types.StringValue(category),
		"settings": settingsList,
	})
	if d.HasError() {
		t.Fatalf("failed to build manager object: %v", d)
	}

	managerList, d := types.ListValue(managerObjType, []attr.Value{managerObj})
	if d.HasError() {
		t.Fatalf("failed to build manager list: %v", d)
	}

	_ = ctx
	return managerList
}

// TestSettingsPolicy_BuildDTO_BasicFields verifies a minimal plan produces a correct DTO.
func TestSettingsPolicy_BuildDTO_BasicFields(t *testing.T) {
	r := newSettingsResource()

	managersList := buildTestSettingsManagersList(t, "automationmanager", []settingModel{
		{UUID: types.StringValue("moveVirtualMachine"), Value: types.StringValue("AUTOMATIC")},
	})

	plan := settingsPolicyResourceModel{
		Name:             types.StringValue("my-policy"),
		EntityType:       types.StringValue("VirtualMachine"),
		Disabled:         types.BoolNull(),
		ScopeUUIDs:       types.ListNull(types.StringType),
		Note:             types.StringNull(),
		ScheduleUUID:     types.StringNull(),
		SettingsManagers: managersList,
	}

	var d diag.Diagnostics
	dto := r.buildDTO(context.Background(), plan, &d)
	if d.HasError() {
		t.Fatalf("buildDTO returned diagnostics errors: %v", d)
	}

	if dto.DisplayName == nil || *dto.DisplayName != "my-policy" {
		t.Errorf("DisplayName: got %v, want %q", dto.DisplayName, "my-policy")
	}
	if dto.EntityType == nil || *dto.EntityType != "VirtualMachine" {
		t.Errorf("EntityType: got %v, want %q", dto.EntityType, "VirtualMachine")
	}
	if dto.Disabled != nil {
		t.Errorf("Disabled should be nil when not set, got %v", dto.Disabled)
	}
	if dto.Scopes != nil {
		t.Errorf("Scopes should be nil when scope_uuids not set, got %v", dto.Scopes)
	}
	if len(dto.SettingsManagers) != 1 {
		t.Fatalf("SettingsManagers length: got %d, want 1", len(dto.SettingsManagers))
	}

	mgr := dto.SettingsManagers[0]
	if mgr.UUID != "automationmanager" {
		t.Errorf("manager UUID: got %q, want %q", mgr.UUID, "automationmanager")
	}
	if len(mgr.Settings) != 1 {
		t.Fatalf("settings length: got %d, want 1", len(mgr.Settings))
	}
	if mgr.Settings[0].UUID != "moveVirtualMachine" {
		t.Errorf("setting UUID: got %q, want %q", mgr.Settings[0].UUID, "moveVirtualMachine")
	}

	// Value should be JSON-encoded as "AUTOMATIC" (quoted string)
	var decoded string
	if err := json.Unmarshal(mgr.Settings[0].Value, &decoded); err != nil {
		t.Fatalf("value unmarshal failed: %v", err)
	}
	if decoded != "AUTOMATIC" {
		t.Errorf("setting value: got %q, want %q", decoded, "AUTOMATIC")
	}
}

// TestSettingsPolicy_BuildDTO_WithScope verifies scope_uuids are converted to Scopes.
func TestSettingsPolicy_BuildDTO_WithScope(t *testing.T) {
	r := newSettingsResource()

	scopeList, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"group-uuid-1", "group-uuid-2"})

	managersList := buildTestSettingsManagersList(t, "automationmanager", []settingModel{
		{UUID: types.StringValue("moveVirtualMachine"), Value: types.StringValue("RECOMMEND")},
	})

	plan := settingsPolicyResourceModel{
		Name:             types.StringValue("scoped-policy"),
		EntityType:       types.StringValue("VirtualMachine"),
		ScopeUUIDs:       scopeList,
		SettingsManagers: managersList,
	}

	var d diag.Diagnostics
	dto := r.buildDTO(context.Background(), plan, &d)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	if len(dto.Scopes) != 2 {
		t.Fatalf("Scopes length: got %d, want 2", len(dto.Scopes))
	}
	if dto.Scopes[0].UUID != "group-uuid-1" {
		t.Errorf("Scopes[0]: got %q, want %q", dto.Scopes[0].UUID, "group-uuid-1")
	}
	if dto.Scopes[1].UUID != "group-uuid-2" {
		t.Errorf("Scopes[1]: got %q, want %q", dto.Scopes[1].UUID, "group-uuid-2")
	}
}

// TestSettingsPolicy_BuildDTO_NumericValue verifies numeric string values are sent
// as JSON numbers (not quoted strings).
func TestSettingsPolicy_BuildDTO_NumericValue(t *testing.T) {
	r := newSettingsResource()

	managersList := buildTestSettingsManagersList(t, "marketsettingsmanager", []settingModel{
		{UUID: types.StringValue("resizeTargetUtilizationVmem"), Value: types.StringValue("90.0")},
	})

	plan := settingsPolicyResourceModel{
		Name:             types.StringValue("numeric-policy"),
		EntityType:       types.StringValue("VirtualMachine"),
		ScopeUUIDs:       types.ListNull(types.StringType),
		SettingsManagers: managersList,
	}

	var d diag.Diagnostics
	dto := r.buildDTO(context.Background(), plan, &d)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	val := dto.SettingsManagers[0].Settings[0].Value
	// 90.0 is a valid JSON number - should be sent unquoted
	var f float64
	if err := json.Unmarshal(val, &f); err != nil {
		t.Fatalf("value should be a JSON number, got %s: %v", string(val), err)
	}
	if f != 90.0 {
		t.Errorf("numeric value: got %v, want 90.0", f)
	}
}

// TestSettingsPolicy_BuildDTO_FilterEmptySettings verifies that managers with no
// valued settings are excluded from the payload (mirrors UI filter behaviour).
func TestSettingsPolicy_BuildDTO_FilterEmptySettings(t *testing.T) {
	r := newSettingsResource()

	// Build a manager with one setting that has a null value - should be excluded.
	ctx := context.Background()
	nullSettingObj, _ := types.ObjectValue(settingAttrTypes, map[string]attr.Value{
		"uuid":  types.StringValue("someUuid"),
		"value": types.StringNull(),
	})
	nullSettingsList, _ := types.ListValue(settingObjType, []attr.Value{nullSettingObj})
	emptyManagerObj, _ := types.ObjectValue(managerAttrTypes, map[string]attr.Value{
		"category": types.StringValue("automationmanager"),
		"settings": nullSettingsList,
	})
	emptyManagerList, _ := types.ListValue(managerObjType, []attr.Value{emptyManagerObj})

	plan := settingsPolicyResourceModel{
		Name:             types.StringValue("empty-policy"),
		EntityType:       types.StringValue("VirtualMachine"),
		ScopeUUIDs:       types.ListNull(types.StringType),
		SettingsManagers: emptyManagerList,
	}
	_ = ctx

	var d diag.Diagnostics
	dto := r.buildDTO(context.Background(), plan, &d)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	if len(dto.SettingsManagers) != 0 {
		t.Errorf("SettingsManagers with no valued settings should be excluded, got %d managers", len(dto.SettingsManagers))
	}
}

// TestSettingsPolicy_MapDTOToModel_BasicFields verifies Read response mapping.
func TestSettingsPolicy_MapDTOToModel_BasicFields(t *testing.T) {
	r := newSettingsResource()

	uuid := "settings-policy-uuid"
	name := "my-settings-policy"
	entityType := "VirtualMachine"
	readOnly := false
	defaultPolicy := false

	dto := &settingsPolicyDTO{
		UUID:        &uuid,
		DisplayName: &name,
		EntityType:  &entityType,
		ReadOnly:    &readOnly,
		Default:     &defaultPolicy,
		SettingsManagers: []settingsManagerDTO{
			{
				UUID: "automationmanager",
				Settings: []settingItemDTO{
					{UUID: "moveVirtualMachine", Value: json.RawMessage(`"AUTOMATIC"`)},
					{UUID: "resizeVirtualMachine", Value: json.RawMessage(`"RECOMMEND"`)},
				},
			},
		},
	}

	model := &settingsPolicyResourceModel{}
	var d diag.Diagnostics
	r.mapDTOToModel(context.Background(), dto, model, &d)
	if d.HasError() {
		t.Fatalf("mapDTOToModel returned diagnostics errors: %v", d)
	}

	if model.UUID.ValueString() != uuid {
		t.Errorf("UUID: got %q, want %q", model.UUID.ValueString(), uuid)
	}
	if model.Name.ValueString() != name {
		t.Errorf("Name: got %q, want %q", model.Name.ValueString(), name)
	}
	if model.EntityType.ValueString() != entityType {
		t.Errorf("EntityType: got %q, want %q", model.EntityType.ValueString(), entityType)
	}
	if model.ReadOnly.ValueBool() != readOnly {
		t.Errorf("ReadOnly: got %v, want %v", model.ReadOnly.ValueBool(), readOnly)
	}
	if model.Default.ValueBool() != defaultPolicy {
		t.Errorf("Default: got %v, want %v", model.Default.ValueBool(), defaultPolicy)
	}

	// Check settings_managers
	if model.SettingsManagers.IsNull() {
		t.Fatal("SettingsManagers should not be null")
	}
	var managers []settingsManagerModel
	d = model.SettingsManagers.ElementsAs(context.Background(), &managers, false)
	if d.HasError() {
		t.Fatalf("ElementsAs on SettingsManagers: %v", d)
	}
	if len(managers) != 1 {
		t.Fatalf("SettingsManagers length: got %d, want 1", len(managers))
	}
	if managers[0].Category.ValueString() != "automationmanager" {
		t.Errorf("manager category: got %q, want %q", managers[0].Category.ValueString(), "automationmanager")
	}

	var settings []settingModel
	d = managers[0].Settings.ElementsAs(context.Background(), &settings, false)
	if d.HasError() {
		t.Fatalf("ElementsAs on Settings: %v", d)
	}
	if len(settings) != 2 {
		t.Fatalf("settings length: got %d, want 2", len(settings))
	}
	if settings[0].UUID.ValueString() != "moveVirtualMachine" {
		t.Errorf("setting[0] UUID: got %q, want %q", settings[0].UUID.ValueString(), "moveVirtualMachine")
	}
	if settings[0].Value.ValueString() != "AUTOMATIC" {
		t.Errorf("setting[0] value: got %q, want %q", settings[0].Value.ValueString(), "AUTOMATIC")
	}
	if settings[1].Value.ValueString() != "RECOMMEND" {
		t.Errorf("setting[1] value: got %q, want %q", settings[1].Value.ValueString(), "RECOMMEND")
	}
}

// TestSettingsPolicy_MapDTOToModel_ScopesExtracted verifies scopes are extracted as UUID strings.
func TestSettingsPolicy_MapDTOToModel_ScopesExtracted(t *testing.T) {
	r := newSettingsResource()

	uuid := "policy-uuid"
	name := "scoped-policy"
	entityType := "VirtualMachine"

	dto := &settingsPolicyDTO{
		UUID:        &uuid,
		DisplayName: &name,
		EntityType:  &entityType,
		Scopes: []settingsScopeDTO{
			{UUID: "group-uuid-1"},
			{UUID: "group-uuid-2"},
		},
	}

	model := &settingsPolicyResourceModel{}
	var d diag.Diagnostics
	r.mapDTOToModel(context.Background(), dto, model, &d)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	if model.ScopeUUIDs.IsNull() {
		t.Fatal("ScopeUUIDs should not be null")
	}

	var scopes []string
	d = model.ScopeUUIDs.ElementsAs(context.Background(), &scopes, false)
	if d.HasError() {
		t.Fatalf("ElementsAs on ScopeUUIDs: %v", d)
	}
	if len(scopes) != 2 {
		t.Fatalf("scopes length: got %d, want 2", len(scopes))
	}
	if scopes[0] != "group-uuid-1" {
		t.Errorf("scopes[0]: got %q, want %q", scopes[0], "group-uuid-1")
	}
	if scopes[1] != "group-uuid-2" {
		t.Errorf("scopes[1]: got %q, want %q", scopes[1], "group-uuid-2")
	}
}

// TestSettingsPolicy_MapDTOToModel_NumericValue verifies numeric values survive the round-trip.
func TestSettingsPolicy_MapDTOToModel_NumericValue(t *testing.T) {
	r := newSettingsResource()

	uuid := "policy-uuid"
	name := "numeric-policy"
	entityType := "VirtualMachine"

	dto := &settingsPolicyDTO{
		UUID:        &uuid,
		DisplayName: &name,
		EntityType:  &entityType,
		SettingsManagers: []settingsManagerDTO{
			{
				UUID: "marketsettingsmanager",
				Settings: []settingItemDTO{
					{UUID: "resizeTargetUtilizationVmem", Value: json.RawMessage(`90.0`)},
				},
			},
		},
	}

	model := &settingsPolicyResourceModel{}
	var d diag.Diagnostics
	r.mapDTOToModel(context.Background(), dto, model, &d)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	var managers []settingsManagerModel
	d = model.SettingsManagers.ElementsAs(context.Background(), &managers, false)
	if d.HasError() {
		t.Fatalf("ElementsAs failed: %v", d)
	}
	var settings []settingModel
	d = managers[0].Settings.ElementsAs(context.Background(), &settings, false)
	if d.HasError() {
		t.Fatalf("ElementsAs on settings failed: %v", d)
	}
	if settings[0].Value.ValueString() != "90.0" {
		t.Errorf("numeric value: got %q, want %q", settings[0].Value.ValueString(), "90.0")
	}
}

// TestSettingsPolicy_MapDTOToModel_EmptySettingsFiltered verifies that settings
// with no value are excluded from the model.
func TestSettingsPolicy_MapDTOToModel_EmptySettingsFiltered(t *testing.T) {
	r := newSettingsResource()

	uuid := "policy-uuid"
	name := "policy"
	entityType := "VirtualMachine"

	dto := &settingsPolicyDTO{
		UUID:        &uuid,
		DisplayName: &name,
		EntityType:  &entityType,
		SettingsManagers: []settingsManagerDTO{
			{
				UUID: "automationmanager",
				Settings: []settingItemDTO{
					{UUID: "moveVirtualMachine", Value: json.RawMessage(`"AUTOMATIC"`)},
					{UUID: "emptyValueSetting", Value: nil}, // no value - should be excluded
				},
			},
		},
	}

	model := &settingsPolicyResourceModel{}
	var d diag.Diagnostics
	r.mapDTOToModel(context.Background(), dto, model, &d)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	var managers []settingsManagerModel
	d = model.SettingsManagers.ElementsAs(context.Background(), &managers, false)
	if d.HasError() {
		t.Fatalf("ElementsAs failed: %v", d)
	}
	var settings []settingModel
	d = managers[0].Settings.ElementsAs(context.Background(), &settings, false)
	if d.HasError() {
		t.Fatalf("ElementsAs on settings failed: %v", d)
	}
	if len(settings) != 1 {
		t.Errorf("expected 1 valued setting, got %d (nil-value settings should be excluded)", len(settings))
	}
	if settings[0].UUID.ValueString() != "moveVirtualMachine" {
		t.Errorf("remaining setting UUID: got %q, want %q", settings[0].UUID.ValueString(), "moveVirtualMachine")
	}
}

// TestSettingsPolicy_MapDTOToModel_NullOptionals verifies null/missing optional fields.
func TestSettingsPolicy_MapDTOToModel_NullOptionals(t *testing.T) {
	r := newSettingsResource()

	uuid := "policy-uuid"
	name := "policy"
	entityType := "VirtualMachine"

	dto := &settingsPolicyDTO{
		UUID:        &uuid,
		DisplayName: &name,
		EntityType:  &entityType,
		// No Schedule, Note, Scopes, SettingsManagers, ReadOnly, Default
	}

	model := &settingsPolicyResourceModel{}
	var d diag.Diagnostics
	r.mapDTOToModel(context.Background(), dto, model, &d)
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}

	if !model.ScheduleUUID.IsNull() {
		t.Error("ScheduleUUID should be null when Schedule absent in DTO")
	}
	if !model.Note.IsNull() {
		t.Error("Note should be null when absent in DTO")
	}
	if !model.ScopeUUIDs.IsNull() {
		t.Error("ScopeUUIDs should be null when Scopes absent in DTO")
	}
	if !model.SettingsManagers.IsNull() {
		t.Error("SettingsManagers should be null when absent in DTO")
	}
	if !model.ReadOnly.IsNull() {
		t.Error("ReadOnly should be null when absent in DTO")
	}
	if !model.Default.IsNull() {
		t.Error("Default should be null when absent in DTO")
	}
	// The settings_policy resource sets Disabled to false (not null) when the
	// API omits the field - this prevents plan/state mismatch after create.
	if model.Disabled.IsNull() || model.Disabled.ValueBool() != false {
		t.Error("Disabled should be false (not null) when absent in DTO")
	}
}
