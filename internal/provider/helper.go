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
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	turboclient "github.com/IBM/turbonomic-go-client"
)

// Tag key and value for "optimized by" tag
const (
	OptimizedByTagName      = "turbonomic_optimized_by"
	OptimizedByTagValue     = "turbonomic-terraform-provider"
	TagAlredyExistsErrorMsg = "INVALID_ARGUMENT: Trying to insert a tag with a key that already exists: turbonomic_optimized_by"
)

func TagEntity(client *turboclient.Client, uuid string) error {
	// tag VM entity with "optimized by" tag if not already tagged
	if len(uuid) > 0 {
		entityTagsReq := turboclient.EntityRequest{
			Uuid: uuid}

		entityTags, err := client.GetEntityTags(entityTagsReq)
		if err != nil {
			return fmt.Errorf("Unable to retrieve entity tags from Turbonomic: %v", err)
		}

		var alreadyTagged bool = false
		for _, item := range entityTags {
			if item.Key == OptimizedByTagName {
				if slices.Contains(item.Values, OptimizedByTagValue) {
					alreadyTagged = true
					break
				}
			}
		}

		if !alreadyTagged {
			tagEntityReq := turboclient.TagEntityRequest{
				Uuid: uuid,
				Tags: []turboclient.Tag{{
					Key:    OptimizedByTagName,
					Values: []string{OptimizedByTagValue}}}}

			_, err := client.TagEntity(tagEntityReq)
			if err != nil {
				if strings.Contains(err.Error(), TagAlredyExistsErrorMsg) {
					return nil
				}
				return fmt.Errorf("Unable to tag an entity in Turbonomic: %v", err)
			}
		}
	}
	return nil
}

// applyDefaultIfEmpty returns the default value if the provided string field is null or empty,
// otherwise returns the original field value.
//
// Parameters:
//   - field: The original string value to check
//   - def: The default value to use if the original is null or empty
//
// Returns:
//   - The original value if it's not null or empty, otherwise the default value
func applyDefaultIfEmpty(field, def types.String) types.String {
	if (field.IsNull() || len(field.ValueString()) == 0) && !def.IsNull() {
		return def
	}
	return field
}

// Nullable is an interface for types that can be null
type Nullable interface {
	IsNull() bool
}

// StringValue is an interface for string types that can provide their string value
type StringValue interface {
	ValueString() string
}

// applyDefaultIfEmptyGeneric is a generic function that returns the default value if the provided field is null,
// otherwise returns the original field value. For string types, it also checks if the string is empty.
//
// Parameters:
//   - field: The original value to check (must implement Nullable)
//   - def: The default value to use if the original is null
//
// Returns:
//   - The original value if it's not null (and not empty for strings), otherwise the default value
func applyDefaultIfEmptyGeneric[T Nullable](field, def T) T {
	// Special handling for string types
	if strField, ok := any(field).(StringValue); ok {
		// For string types, check if it's null OR empty
		if field.IsNull() || len(strField.ValueString()) == 0 {
			// Only apply default if it's not null
			if !def.IsNull() {
				return def
			}
		}
		return field
	}

	// For non-string types, just check if it's null
	if field.IsNull() {
		return def
	}
	return field
}

// convertKbitToMiBps converts a value from Kibit/sec to MiB/sec and rounds the result
func convertKibitToMiBps(value float64) float64 {
	return math.Round(value / 8192)
}

// convertMBtoGiB converts a value from MB to GiB and rounds to the nearest integer
func convertMiBtoGiB(value float64) int64 {
	return int64(math.Round(value / 1024))
}
