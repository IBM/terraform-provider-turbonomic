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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRawValueToString verifies conversion from json.RawMessage to plain string.
func TestRawValueToString(t *testing.T) {
	tests := []struct {
		name  string
		input json.RawMessage
		want  string
	}{
		{
			name:  "JSON string unquoted",
			input: json.RawMessage(`"RECOMMEND"`),
			want:  "RECOMMEND",
		},
		{
			name:  "JSON string AUTOMATIC",
			input: json.RawMessage(`"AUTOMATIC"`),
			want:  "AUTOMATIC",
		},
		{
			name:  "JSON string MANUAL",
			input: json.RawMessage(`"MANUAL"`),
			want:  "MANUAL",
		},
		{
			name:  "JSON string DISABLED",
			input: json.RawMessage(`"DISABLED"`),
			want:  "DISABLED",
		},
		{
			name:  "JSON number stays as string",
			input: json.RawMessage(`90.0`),
			want:  "90.0",
		},
		{
			name:  "JSON integer",
			input: json.RawMessage(`42`),
			want:  "42",
		},
		{
			name:  "JSON bool true",
			input: json.RawMessage(`true`),
			want:  "true",
		},
		{
			name:  "JSON bool false",
			input: json.RawMessage(`false`),
			want:  "false",
		},
		{
			name:  "nil input returns empty string",
			input: nil,
			want:  "",
		},
		{
			name:  "empty input returns empty string",
			input: json.RawMessage{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rawValueToString(tt.input)
			if got != tt.want {
				t.Errorf("rawValueToString(%s) = %q, want %q", string(tt.input), got, tt.want)
			}
		})
	}
}

// TestStringToRawValue verifies conversion from plain string to json.RawMessage.
func TestStringToRawValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // expected JSON bytes
	}{
		{
			name:  "plain string gets quoted",
			input: "RECOMMEND",
			want:  `"RECOMMEND"`,
		},
		{
			name:  "AUTOMATIC gets quoted",
			input: "AUTOMATIC",
			want:  `"AUTOMATIC"`,
		},
		{
			name:  "numeric string used as-is",
			input: "90.0",
			want:  `90.0`,
		},
		{
			name:  "integer string used as-is",
			input: "42",
			want:  `42`,
		},
		{
			name:  "bool true used as-is",
			input: "true",
			want:  `true`,
		},
		{
			name:  "bool false used as-is",
			input: "false",
			want:  `false`,
		},
		{
			name:  "empty string returns nil",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringToRawValue(tt.input)
			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("stringToRawValue(%q) = %s, want %s", tt.input, gotStr, tt.want)
			}
		})
	}
}

// TestRawValueRoundTrip verifies that stringToRawValue → rawValueToString is an identity.
func TestRawValueRoundTrip(t *testing.T) {
	values := []string{"RECOMMEND", "AUTOMATIC", "MANUAL", "DISABLED", "90.0", "true", "false", "42"}
	for _, v := range values {
		t.Run(v, func(t *testing.T) {
			raw := stringToRawValue(v)
			got := rawValueToString(raw)
			if got != v {
				t.Errorf("round-trip(%q): got %q", v, got)
			}
		})
	}
}

// TestPolicyHTTPDelete_404Tolerant verifies that policyHTTPDelete treats a 404
// response as success (resource already gone) rather than returning an error.
func TestPolicyHTTPDelete_404Tolerant(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "200 OK", statusCode: http.StatusOK, wantErr: false},
		{name: "204 No Content", statusCode: http.StatusNoContent, wantErr: false},
		{name: "404 Not Found - already gone", statusCode: http.StatusNotFound, wantErr: false},
		{name: "403 Forbidden - still errors", statusCode: http.StatusForbidden, wantErr: true},
		{name: "500 Internal Server Error - still errors", statusCode: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer srv.Close()

			err := policyHTTPDelete(t.Context(), srv.Client(), srv.URL, "/test/path")
			if (err != nil) != tt.wantErr {
				t.Errorf("policyHTTPDelete status %d: got err=%v, wantErr=%v", tt.statusCode, err, tt.wantErr)
			}
		})
	}
}
