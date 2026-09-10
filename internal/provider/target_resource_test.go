// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/IBM/turbonomic-go-client/api/generated"
)

// TestBuildInputFields verifies that buildInputFields merges regular and
// write-only maps into a flat []InputFieldApiDTO.
func TestBuildInputFields(t *testing.T) {
	ctx := context.Background()

	regular, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"displayName": types.StringValue("My Target"),
		"hostname":    types.StringValue("host.example.com"),
	})
	writeOnly, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"apiToken": types.StringValue("secret-value"),
	})

	fields, err := buildInputFields(ctx, regular, writeOnly)
	if err != nil {
		t.Fatalf("buildInputFields returned unexpected error: %v", err)
	}

	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	byName := make(map[string]string, len(fields))
	for _, f := range fields {
		if f.Name != nil && f.Value != nil {
			byName[*f.Name] = *f.Value
		}
	}

	if byName["displayName"] != "My Target" {
		t.Errorf("displayName: got %q, want %q", byName["displayName"], "My Target")
	}
	if byName["hostname"] != "host.example.com" {
		t.Errorf("hostname: got %q, want %q", byName["hostname"], "host.example.com")
	}
	if byName["apiToken"] != "secret-value" {
		t.Errorf("apiToken: got %q, want %q", byName["apiToken"], "secret-value")
	}
}

// TestBuildInputFields_NullMaps verifies that null maps produce an empty slice
// without error.
func TestBuildInputFields_NullMaps(t *testing.T) {
	fields, err := buildInputFields(context.Background(), types.MapNull(types.StringType), types.MapNull(types.StringType))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 0 {
		t.Errorf("expected 0 fields, got %d", len(fields))
	}
}

// TestMapTargetResponseToModel_NonSecretFieldUpdated verifies that a non-secret
// inputField returned by the API updates its matching key in the state map.
func TestMapTargetResponseToModel_NonSecretFieldUpdated(t *testing.T) {
	isSecret := false
	updatedValue := "Updated Target Name"
	name := "displayName"

	planMap, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"displayName": types.StringValue("Old Name"),
	})

	model := targetResourceModel{
		InputFields: planMap,
	}

	resp := &generated.TargetApiDTO{
		Type: "AWS",
		InputFields: &[]generated.InputFieldApiDTO{
			{Name: &name, Value: &updatedValue, IsSecret: &isSecret},
		},
	}

	mapTargetResponseToModel(context.Background(), resp, &model)

	elems := make(map[string]string)
	_ = model.InputFields.ElementsAs(context.Background(), &elems, false)

	if elems["displayName"] != "Updated Target Name" {
		t.Errorf("displayName: got %q, want %q", elems["displayName"], "Updated Target Name")
	}
}

// TestMapTargetResponseToModel_SecretFieldNotUpdated verifies that a secret
// inputField returned by the API does NOT overwrite the state map value.
func TestMapTargetResponseToModel_SecretFieldNotUpdated(t *testing.T) {
	isSecret := true
	apiValue := "should-not-appear"
	name := "password"

	planMap, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"password": types.StringValue("original-plan-value"),
	})

	model := targetResourceModel{
		InputFields: planMap,
	}

	resp := &generated.TargetApiDTO{
		Type: "vCenter",
		InputFields: &[]generated.InputFieldApiDTO{
			{Name: &name, Value: &apiValue, IsSecret: &isSecret},
		},
	}

	mapTargetResponseToModel(context.Background(), resp, &model)

	elems := make(map[string]string)
	_ = model.InputFields.ElementsAs(context.Background(), &elems, false)

	if elems["password"] != "original-plan-value" {
		t.Errorf("password should be preserved as plan value, got %q", elems["password"])
	}
}

// TestMapTargetResponseToModel_NewKeyNotAdded verifies that an inputField key
// returned by the API that was NOT in the plan map is not added to the state.
func TestMapTargetResponseToModel_NewKeyNotAdded(t *testing.T) {
	isSecret := false
	apiValue := "some-default"
	name := "optionalField"

	planMap, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"displayName": types.StringValue("My Target"),
	})

	model := targetResourceModel{
		InputFields: planMap,
	}

	resp := &generated.TargetApiDTO{
		Type: "Kubernetes",
		InputFields: &[]generated.InputFieldApiDTO{
			{Name: &name, Value: &apiValue, IsSecret: &isSecret},
		},
	}

	mapTargetResponseToModel(context.Background(), resp, &model)

	elems := make(map[string]string)
	_ = model.InputFields.ElementsAs(context.Background(), &elems, false)

	if _, exists := elems["optionalField"]; exists {
		t.Errorf("optionalField should not be added to state - it was not in the plan")
	}
	if len(elems) != 1 {
		t.Errorf("state map should still have 1 key, got %d", len(elems))
	}
}

// TestMapTargetResponseToModel_HealthFields verifies that health summary fields
// are populated from the API response.
func TestMapTargetResponseToModel_HealthFields(t *testing.T) {
	ts := "2024-01-15T10:30:00Z"
	errTxt := "connection refused"
	healthState := generated.HealthStateNORMAL
	rollup := generated.HealthStateMAJOR

	model := targetResourceModel{
		InputFields: types.MapNull(types.StringType),
	}

	resp := &generated.TargetApiDTO{
		Type: "AWS",
		HealthSummary: &generated.TargetHealthSummaryApiDTO{
			HealthState:                   healthState,
			RollupState:                   rollup,
			TimeOfLastSuccessfulDiscovery: &ts,
		},
		Health: &generated.TargetHealthApiDTO{
			ErrorText:                &errTxt,
			HealthState:              healthState,
			HealthClassDiscriminator: "TargetHealthApiDTO",
			HealthCategory:           generated.HealthCategoryTARGET,
			RollupState:              rollup,
			TargetName:               "my-target",
			TargetStatusSubcategory:  generated.TargetStatusSubcategoryVALIDATION,
			Uuid:                     "abc-123",
		},
	}

	mapTargetResponseToModel(context.Background(), resp, &model)

	if model.HealthState.ValueString() != "NORMAL" {
		t.Errorf("health_state: got %q, want %q", model.HealthState.ValueString(), "NORMAL")
	}
	if model.HealthRollupState.ValueString() != "MAJOR" {
		t.Errorf("health_rollup_state: got %q, want %q", model.HealthRollupState.ValueString(), "MAJOR")
	}
	if model.LastSuccessfulDiscovery.ValueString() != ts {
		t.Errorf("last_successful_discovery: got %q, want %q", model.LastSuccessfulDiscovery.ValueString(), ts)
	}
	if model.HealthErrorText.ValueString() != errTxt {
		t.Errorf("health_error_text: got %q, want %q", model.HealthErrorText.ValueString(), errTxt)
	}
}

// TestMapTargetResponseToModel_NilHealthSummary verifies that nil healthSummary
// results in null computed health fields (no panic).
func TestMapTargetResponseToModel_NilHealthSummary(t *testing.T) {
	model := targetResourceModel{
		InputFields: types.MapNull(types.StringType),
	}

	resp := &generated.TargetApiDTO{
		Type:          "Azure",
		HealthSummary: nil,
		Health:        nil,
	}

	mapTargetResponseToModel(context.Background(), resp, &model)

	if !model.HealthState.IsNull() {
		t.Errorf("health_state should be null when healthSummary is absent")
	}
	if !model.HealthRollupState.IsNull() {
		t.Errorf("health_rollup_state should be null when healthSummary is absent")
	}
	if !model.HealthErrorText.IsNull() {
		t.Errorf("health_error_text should be null when health is absent")
	}
	if !model.LastSuccessfulDiscovery.IsNull() {
		t.Errorf("last_successful_discovery should be null when healthSummary is absent")
	}
}

// TestIsNotFoundError verifies the 404 detection helper.
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantYes bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantYes: false,
		},
		{
			name:    "404 error message",
			err:     fmt.Errorf("get target failed with status 404: not found"),
			wantYes: true,
		},
		{
			name:    "500 error message",
			err:     fmt.Errorf("get target failed with status 500: internal error"),
			wantYes: false,
		},
		{
			name:    "unrelated error",
			err:     errors.New("connection refused"),
			wantYes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNotFoundError(tt.err)
			if got != tt.wantYes {
				t.Errorf("isNotFoundError(%v) = %v, want %v", tt.err, got, tt.wantYes)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// mapTargetResponseToModel - UUID and audit fields
// ---------------------------------------------------------------------------

// TestMapTargetResponseToModel_UUIDAuditFields verifies that UUID and audit
// fields (LastEditTime, LastEditUser, IsProbeRegistered) are correctly mapped
// from the API response into the model.
func TestMapTargetResponseToModel_UUIDAuditFields(t *testing.T) {
	uuid := "target-uuid-123"
	editTime := "2024-06-01T12:00:00Z"
	editUser := "admin"
	isProbeRegistered := true

	model := targetResourceModel{
		InputFields: types.MapNull(types.StringType),
	}

	resp := &generated.TargetApiDTO{
		Type:              "AWS",
		Uuid:              &uuid,
		LastEditTime:      &editTime,
		LastEditUser:      &editUser,
		IsProbeRegistered: &isProbeRegistered,
	}

	mapTargetResponseToModel(context.Background(), resp, &model)

	if model.UUID.ValueString() != uuid {
		t.Errorf("uuid: got %q, want %q", model.UUID.ValueString(), uuid)
	}
	if model.LastEditTime.ValueString() != editTime {
		t.Errorf("last_edit_time: got %q, want %q", model.LastEditTime.ValueString(), editTime)
	}
	if model.LastEditUser.ValueString() != editUser {
		t.Errorf("last_edit_user: got %q, want %q", model.LastEditUser.ValueString(), editUser)
	}
	if !model.IsProbeRegistered.ValueBool() {
		t.Error("is_probe_registered should be true")
	}
}

// TestMapTargetResponseToModel_NilAuditFields verifies that nil LastEditTime
// and LastEditUser become null in the model (no panic, no empty string).
func TestMapTargetResponseToModel_NilAuditFields(t *testing.T) {
	uuid := "target-uuid-456"
	model := targetResourceModel{
		InputFields: types.MapNull(types.StringType),
	}

	resp := &generated.TargetApiDTO{
		Type:         "Azure",
		Uuid:         &uuid,
		LastEditTime: nil,
		LastEditUser: nil,
	}

	mapTargetResponseToModel(context.Background(), resp, &model)

	if !model.LastEditTime.IsNull() {
		t.Errorf("last_edit_time should be null when API returns nil, got %q", model.LastEditTime.ValueString())
	}
	if !model.LastEditUser.IsNull() {
		t.Errorf("last_edit_user should be null when API returns nil, got %q", model.LastEditUser.ValueString())
	}
}

// TestMapTargetResponseToModel_NilResponse verifies that a nil TargetApiDTO
// does not panic and leaves the model unchanged.
func TestMapTargetResponseToModel_NilResponse(t *testing.T) {
	model := targetResourceModel{
		UUID:        types.StringValue("existing-uuid"),
		InputFields: types.MapNull(types.StringType),
	}

	// Must not panic
	mapTargetResponseToModel(context.Background(), nil, &model)

	if model.UUID.ValueString() != "existing-uuid" {
		t.Error("UUID should be unchanged when resp is nil")
	}
}

// TestMapTargetResponseToModel_NullInputFieldsInModel verifies that when
// model.InputFields is null (e.g. not configured), API input fields do not
// cause a panic and the field stays null.
func TestMapTargetResponseToModel_NullInputFieldsInModel(t *testing.T) {
	isSecret := false
	apiValue := "some-value"
	name := "displayName"

	model := targetResourceModel{
		InputFields: types.MapNull(types.StringType), // not configured
	}

	resp := &generated.TargetApiDTO{
		Type: "vCenter",
		InputFields: &[]generated.InputFieldApiDTO{
			{Name: &name, Value: &apiValue, IsSecret: &isSecret},
		},
	}

	// Must not panic and must leave InputFields null
	mapTargetResponseToModel(context.Background(), resp, &model)

	if !model.InputFields.IsNull() {
		t.Error("input_fields should remain null when model had no input_fields configured")
	}
}

// ---------------------------------------------------------------------------
// buildInputFields - edge cases
// ---------------------------------------------------------------------------

// TestBuildInputFields_WriteOnlyOnly verifies that a null regular map and a
// populated write-only map produce only the write-only fields.
func TestBuildInputFields_WriteOnlyOnly(t *testing.T) {
	writeOnly, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"hcpToken": types.StringValue("vault-token"),
	})

	fields, err := buildInputFields(context.Background(), types.MapNull(types.StringType), writeOnly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Name == nil || *fields[0].Name != "hcpToken" {
		t.Errorf("expected field name %q, got %v", "hcpToken", fields[0].Name)
	}
	if fields[0].Value == nil || *fields[0].Value != "vault-token" {
		t.Errorf("expected field value %q, got %v", "vault-token", fields[0].Value)
	}
}

// TestBuildInputFields_RegularOnly verifies that a null write-only map and a
// populated regular map produce only the regular fields.
func TestBuildInputFields_RegularOnly(t *testing.T) {
	regular, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"hostname": types.StringValue("turbo.example.com"),
	})

	fields, err := buildInputFields(context.Background(), regular, types.MapNull(types.StringType))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Name == nil || *fields[0].Name != "hostname" {
		t.Errorf("expected field name %q, got %v", "hostname", fields[0].Name)
	}
}

// ---------------------------------------------------------------------------
// InputFieldsWoVersion - preserved through model round-trip
// ---------------------------------------------------------------------------

// TestMapTargetResponseToModel_WoVersionPreserved verifies that
// InputFieldsWoVersion set in the plan model is not cleared by
// mapTargetResponseToModel (the function must never touch it).
func TestMapTargetResponseToModel_WoVersionPreserved(t *testing.T) {
	uuid := "uuid-999"
	model := targetResourceModel{
		InputFields:          types.MapNull(types.StringType),
		InputFieldsWoVersion: types.Int64Value(3),
	}

	resp := &generated.TargetApiDTO{
		Type: "Kubernetes",
		Uuid: &uuid,
	}

	mapTargetResponseToModel(context.Background(), resp, &model)

	if model.InputFieldsWoVersion.ValueInt64() != 3 {
		t.Errorf("input_fields_wo_version should remain 3 after mapping, got %d",
			model.InputFieldsWoVersion.ValueInt64())
	}
}

// TestMapTargetResponseToModel_WoVersionZeroDefault verifies that a zero
// version (default when not set) is also preserved unchanged.
func TestMapTargetResponseToModel_WoVersionZeroDefault(t *testing.T) {
	uuid := "uuid-000"
	model := targetResourceModel{
		InputFields:          types.MapNull(types.StringType),
		InputFieldsWoVersion: types.Int64Value(0),
	}

	resp := &generated.TargetApiDTO{
		Type: "AWS",
		Uuid: &uuid,
	}

	mapTargetResponseToModel(context.Background(), resp, &model)

	if model.InputFieldsWoVersion.ValueInt64() != 0 {
		t.Errorf("input_fields_wo_version should remain 0, got %d",
			model.InputFieldsWoVersion.ValueInt64())
	}
}

// ---------------------------------------------------------------------------
// ptrToString / ptrToBool helpers
// ---------------------------------------------------------------------------

func TestPtrToString(t *testing.T) {
	s := "hello"
	if got := ptrToString(&s); got != "hello" {
		t.Errorf("ptrToString(&s): got %q, want %q", got, "hello")
	}
	if got := ptrToString(nil); got != "" {
		t.Errorf("ptrToString(nil): got %q, want empty string", got)
	}
}

func TestPtrToBool(t *testing.T) {
	b := true
	if got := ptrToBool(&b); !got {
		t.Error("ptrToBool(&true): got false, want true")
	}
	if got := ptrToBool(nil); got {
		t.Error("ptrToBool(nil): got true, want false")
	}
}
