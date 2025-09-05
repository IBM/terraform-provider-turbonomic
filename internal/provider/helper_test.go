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
	"testing"
)

func TestConvertMiBtoGiB(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		expected int64
	}{
		{
			name:     "Zero value",
			input:    0,
			expected: 0,
		},
		{
			name:     "Exact GiB value",
			input:    1024,
			expected: 1,
		},
		{
			name:     "Multiple GiB value",
			input:    2048,
			expected: 2,
		},
		{
			name:     "Round down",
			input:    1023,
			expected: 1, // 1023/1024 = 0.999, rounds to 1
		},
		{
			name:     "Round up",
			input:    1025,
			expected: 1, // 1025/1024 = 1.001, rounds to 1
		},
		{
			name:     "Round to nearest",
			input:    1536, // 1.5 GiB
			expected: 2,    // 1536/1024 = 1.5, rounds to 2
		},
		{
			name:     "Large value",
			input:    10240,
			expected: 10,
		},
		{
			name:     "Fractional value below 1 GiB",
			input:    512,
			expected: 1, // 512/1024 = 0.5, rounds to 1
		},
		{
			name:     "Fractional value below 0.5 GiB",
			input:    400,
			expected: 0, // 400/1024 = 0.39, rounds to 0
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertMiBtoGiB(tc.input)
			if result != tc.expected {
				t.Errorf("convertMiBtoGiB(%f) = %d; want %d", tc.input, result, tc.expected)
			}
		})
	}
}

func TestConvertKibitToMiBps(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		expected float64
	}{
		{
			name:     "Zero value",
			input:    0,
			expected: 0,
		},
		{
			name:     "Exact MiB/s value",
			input:    8192, // 1 MiB/s = 8192 Kibit/s
			expected: 1,
		},
		{
			name:     "Multiple MiB/s value",
			input:    16384, // 2 MiB/s = 16384 Kibit/s
			expected: 2,
		},
		{
			name:     "Round down",
			input:    8191,
			expected: 1, // 8191/8192 = 0.999, rounds to 1
		},
		{
			name:     "Round up",
			input:    8193,
			expected: 1, // 8193/8192 = 1.0001, rounds to 1
		},
		{
			name:     "Round to nearest",
			input:    12288, // 1.5 MiB/s
			expected: 2,     // 12288/8192 = 1.5, rounds to 2
		},
		{
			name:     "Large value",
			input:    81920, // 10 MiB/s
			expected: 10,
		},
		{
			name:     "Fractional value below 1 MiB/s",
			input:    4096, // 0.5 MiB/s
			expected: 1,    // 4096/8192 = 0.5, rounds to 1
		},
		{
			name:     "Fractional value below 0.5 MiB/s",
			input:    3000,
			expected: 0, // 3000/8192 = 0.366, rounds to 0
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertKibitToMiBps(tc.input)
			if result != tc.expected {
				t.Errorf("convertKibitToMiBps(%f) = %f; want %f", tc.input, result, tc.expected)
			}
		})
	}
}
