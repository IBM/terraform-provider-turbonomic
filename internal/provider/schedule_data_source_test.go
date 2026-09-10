// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuildRecurrenceObject_Nil(t *testing.T) {
	obj, diags := buildRecurrenceObject(nil)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if !obj.(types.Object).IsNull() {
		t.Errorf("expected null object for nil recurrence, got %v", obj)
	}
}

func TestBuildRecurrenceObject_Daily(t *testing.T) {
	interval := int32(2)
	r := &recurrenceDTO{
		Type:     "DAILY",
		Interval: &interval,
	}
	obj, diags := buildRecurrenceObject(r)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	to := obj.(types.Object)
	attrs := to.Attributes()

	if attrs["type"].(types.String).ValueString() != "DAILY" {
		t.Errorf("expected type DAILY, got %v", attrs["type"])
	}
	if attrs["interval"].(types.Int64).ValueInt64() != 2 {
		t.Errorf("expected interval 2, got %v", attrs["interval"])
	}
}

func TestBuildRecurrenceObject_Weekly(t *testing.T) {
	r := &recurrenceDTO{
		Type:       "WEEKLY",
		DaysOfWeek: []string{"Mon", "Wed", "Fri"},
	}
	obj, diags := buildRecurrenceObject(r)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	to := obj.(types.Object)
	attrs := to.Attributes()

	dowList := attrs["days_of_week"].(types.List)
	elems := dowList.Elements()
	if len(elems) != 3 {
		t.Fatalf("expected 3 days_of_week, got %d", len(elems))
	}
	if elems[0].(types.String).ValueString() != "Mon" {
		t.Errorf("expected Mon, got %v", elems[0])
	}
}

func TestBuildRecurrenceObject_Monthly(t *testing.T) {
	r := &recurrenceDTO{
		Type:        "MONTHLY",
		DaysOfMonth: []int32{1, 15},
	}
	obj, diags := buildRecurrenceObject(r)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	to := obj.(types.Object)
	attrs := to.Attributes()

	domList := attrs["days_of_month"].(types.List)
	elems := domList.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 days_of_month, got %d", len(elems))
	}
	if elems[1].(types.Int64).ValueInt64() != 15 {
		t.Errorf("expected 15, got %v", elems[1])
	}
}

func TestBuildScheduleItemObject_OneTime(t *testing.T) {
	uuid := "sched-uuid-1"
	name := "Maintenance Window"
	s := &scheduleDTO{
		Uuid:        &uuid,
		DisplayName: &name,
		StartDate:   "2024-01-15T09:00",
		StartTime:   "2024-01-15T09:00",
		EndTime:     "2024-01-15T17:00",
		Recurrence:  nil,
	}
	obj, diags := buildScheduleItemObject(s)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	to := obj.(types.Object)
	attrs := to.Attributes()

	if attrs["uuid"].(types.String).ValueString() != "sched-uuid-1" {
		t.Errorf("expected uuid sched-uuid-1, got %v", attrs["uuid"])
	}
	if attrs["display_name"].(types.String).ValueString() != "Maintenance Window" {
		t.Errorf("expected display_name 'Maintenance Window', got %v", attrs["display_name"])
	}
	if attrs["start_date"].(types.String).ValueString() != "2024-01-15T09:00" {
		t.Errorf("unexpected start_date: %v", attrs["start_date"])
	}
	// recurrence should be null object
	rec := attrs["recurrence"].(types.Object)
	if !rec.IsNull() {
		t.Errorf("expected null recurrence for one-time schedule, got %v", rec)
	}
}

func TestBuildScheduleItemObject_WithRecurrence(t *testing.T) {
	uuid := "sched-uuid-2"
	name := "Weekly Backup"
	tz := "America/New_York"
	nextOcc := "2024-01-22T09:00"
	nextTS := int64(1705914000000)
	s := &scheduleDTO{
		Uuid:                    &uuid,
		DisplayName:             &name,
		StartDate:               "2024-01-15T09:00",
		StartTime:               "2024-01-15T09:00",
		EndTime:                 "2024-01-15T17:00",
		TimeZone:                &tz,
		NextOccurrence:          &nextOcc,
		NextOccurrenceTimestamp: &nextTS,
		Recurrence: &recurrenceDTO{
			Type:       "WEEKLY",
			DaysOfWeek: []string{"Mon"},
		},
	}
	obj, diags := buildScheduleItemObject(s)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	to := obj.(types.Object)
	attrs := to.Attributes()

	if attrs["time_zone"].(types.String).ValueString() != "America/New_York" {
		t.Errorf("unexpected time_zone: %v", attrs["time_zone"])
	}
	if attrs["next_occurrence_timestamp"].(types.Int64).ValueInt64() != 1705914000000 {
		t.Errorf("unexpected next_occurrence_timestamp: %v", attrs["next_occurrence_timestamp"])
	}
	rec := attrs["recurrence"].(types.Object)
	if rec.IsNull() {
		t.Error("expected non-null recurrence")
	}
	recAttrs := rec.Attributes()
	if recAttrs["type"].(types.String).ValueString() != "WEEKLY" {
		t.Errorf("expected WEEKLY recurrence, got %v", recAttrs["type"])
	}
}

func TestBuildScheduleItemObject_NullOptionals(t *testing.T) {
	s := &scheduleDTO{
		StartDate: "2024-06-01T08:00",
		StartTime: "2024-06-01T08:00",
		EndTime:   "2024-06-01T12:00",
	}
	obj, diags := buildScheduleItemObject(s)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	to := obj.(types.Object)
	attrs := to.Attributes()

	if !attrs["uuid"].(types.String).IsNull() {
		t.Errorf("expected null uuid, got %v", attrs["uuid"])
	}
	if !attrs["end_date"].(types.String).IsNull() {
		t.Errorf("expected null end_date, got %v", attrs["end_date"])
	}
	if !attrs["next_occurrence"].(types.String).IsNull() {
		t.Errorf("expected null next_occurrence, got %v", attrs["next_occurrence"])
	}
	// remaining_time_active_ms should be 0 (not null) when field absent
	if attrs["remaining_time_active_ms"].(types.Int64).ValueInt64() != 0 {
		t.Errorf("expected remaining_time_active_ms=0, got %v", attrs["remaining_time_active_ms"])
	}
}

func TestScheduleItemAttrTypes_Complete(t *testing.T) {
	// Ensure scheduleItemAttrTypes has no missing keys vs what buildScheduleItemObject produces.
	attrTypes := scheduleItemAttrTypes()
	expectedKeys := []string{
		"uuid", "display_name", "start_date", "start_time", "end_time",
		"end_date", "time_zone", "next_occurrence", "next_occurrence_timestamp",
		"remaining_time_active_ms", "recurrence",
	}
	for _, k := range expectedKeys {
		if _, ok := attrTypes[k]; !ok {
			t.Errorf("missing key %q in scheduleItemAttrTypes", k)
		}
	}
	if len(attrTypes) != len(expectedKeys) {
		t.Errorf("attrTypes has %d keys, want %d", len(attrTypes), len(expectedKeys))
	}
}

// Ensure attr.Value interface is satisfied - compile-time check via blank assignment.
var _ attr.Value = types.ObjectNull(scheduleItemAttrTypes())
