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

import "context"

type LoggerCustom interface {
	Info(ctx context.Context, msg string, args ...any)
	Debug(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
}

type LoggerConfig struct {
	Logger LoggerCustom
	Ctx    context.Context
}

type LoggingOption func(*LoggerConfig)

func WithLogger(logger LoggerCustom) LoggingOption {
	return func(l *LoggerConfig) {
		l.Logger = logger
	}
}

func WithContext(ctx context.Context) LoggingOption {
	return func(l *LoggerConfig) {
		l.Ctx = ctx
	}
}

func SetLogConfig(options []LoggingOption) LoggerConfig {

	logConfig := LoggerConfig{}
	for _, opt := range options {
		opt(&logConfig)
	}

	if logConfig.Logger == nil {
		logConfig.Logger = NewSlogLogger()
	}

	if logConfig.Ctx == nil {
		logConfig.Ctx = context.Background()
	}

	return logConfig
}
