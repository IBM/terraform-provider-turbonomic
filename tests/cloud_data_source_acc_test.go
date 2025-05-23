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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Test turbonomic_cloud_data_source
func TestAccCloudDataSource(t *testing.T) {
	for _, tc := range []struct {
		name                string
		testEntity          string
		testEntityType      string
		entityName          string
		entityType          string
		currentInstanceType string
		newInstanceType     string
		defaultSize         string
	}{
		{
			name:                "Valid VM Recommendation",
			testEntity:          vmName,
			testEntityType:      "VirtualMachine",
			entityName:          vmName,
			entityType:          "VirtualMachine",
			currentInstanceType: vmCurrSize,
			newInstanceType:     vmNewSize,
			defaultSize:         vmNewSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_cloud_entity_recommendation" "test" {
							entity_name  = "` + tc.testEntity + `"
							entity_type  = "` + tc.testEntityType + `"
							default_size = "` + tc.defaultSize + `"
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_name", tc.entityName),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_type", tc.entityType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "current_instance_type", tc.currentInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "new_instance_type", tc.newInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "default_size", tc.defaultSize),
						),
					},
				},
			})
		})
	}
}

func TestAccCloudDataSourceOAuth(t *testing.T) {
	for _, tc := range []struct {
		name                string
		testEntity          string
		testEntityType      string
		entityName          string
		entityType          string
		currentInstanceType string
		newInstanceType     string
		defaultSize         string
	}{
		{
			name:                "Valid VM Recommendation",
			testEntity:          vmName,
			testEntityType:      "VirtualMachine",
			entityName:          vmName,
			entityType:          "VirtualMachine",
			currentInstanceType: vmCurrSize,
			newInstanceType:     vmNewSize,
			defaultSize:         vmNewSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfigoAuth +
							`data "turbonomic_cloud_entity_recommendation" "test" {
							entity_name  = "` + tc.testEntity + `"
							entity_type  = "` + tc.testEntityType + `"
							default_size = "` + tc.defaultSize + `"
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_name", tc.entityName),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_type", tc.entityType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "current_instance_type", tc.currentInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "new_instance_type", tc.newInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "default_size", tc.defaultSize),
						),
					},
				},
			})
		})
	}
}

func init() {
	os.Setenv("TF_ACC", "1")
}
