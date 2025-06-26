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
	"slices"

	turboclient "github.com/IBM/turbonomic-go-client"
)

// Tag key and value for "optimized by" tag
const (
	OptimizedByTagName  = "turbonomic_optimized_by"
	OptimizedByTagValue = "turbonomic-terraform-provider"
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
				return fmt.Errorf("Unable to tag an entity in Turbonomic: %v", err)
			}
		}
	}
	return nil
}
