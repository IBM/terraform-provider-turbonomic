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
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type TfLogAdapter struct{}

func (l *TfLogAdapter) Info(ctx context.Context, msg string, args ...any) {
	tflog.Info(ctx, msg, argsToMap(args...))
}

func (l *TfLogAdapter) Debug(ctx context.Context, msg string, args ...any) {
	tflog.Debug(ctx, msg, argsToMap(args...))
}

func (l *TfLogAdapter) Error(ctx context.Context, msg string, args ...any) {
	tflog.Error(ctx, msg, argsToMap(args...))
}

// Helper to convert slog-style args to map[string]interface{}
func argsToMap(args ...any) map[string]any {
	m := make(map[string]any)
	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		m[key] = args[i+1]
	}
	return m
}
