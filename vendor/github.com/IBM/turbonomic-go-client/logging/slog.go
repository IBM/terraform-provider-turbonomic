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

// For Integrations tests, the ActionTests struct is referenced from testdata.go which needs to be created based on testdata.go.template.
// Integrations tests will only run if the environment variable `INTEGRATION` is set.

package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type SlogLogger struct {
	logger *slog.Logger
}

func NewSlogLogger() *SlogLogger {
	var level = new(slog.LevelVar)
	logLevel := strings.ToUpper(os.Getenv("T8C_LOG"))
	switch logLevel {
	case "DEBUG":
		level.Set(slog.LevelDebug)
	case "INFO":
		level.Set(slog.LevelInfo)
	case "WARN":
		level.Set(slog.LevelWarn)
	case "ERROR":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	slog.SetDefault(slog.New(handler))
	// return slog.Default()

	return &SlogLogger{
		logger: slog.Default(),
	}
}

func (l *SlogLogger) Info(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

func (l *SlogLogger) Debug(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

func (l *SlogLogger) Error(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}
