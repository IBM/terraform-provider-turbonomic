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

package logger

import (
	"reflect"
	"testing"
)

func Test_argsToMap(t *testing.T) {
	tests := []struct {
		name string
		args []any
		want map[string]any
	}{
		{

			name: "Valid key-value pairs",
			args: []any{"name", "John", "age", 30},
			want: map[string]any{"name": "John", "age": 30},
		},
		{
			name: "Odd number of arguments",
			args: []any{"name", "Bob", "age"},
			want: map[string]any{"name": "Bob"},
		},
		{
			name: "Nil value",
			args: []any{},
			want: map[string]any{},
		},
		{
			name: "Non-string key is ignored",
			args: []any{123, "value", "valid", true},
			want: map[string]any{"valid": true},
		},
		{
			name: "Multiple invalid keys",
			args: []any{true, "yes", 42, "answer", "valid", "entry"},
			want: map[string]any{"valid": "entry"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := argsToMap(tt.args...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("argsToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
