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

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Test turbonomic_aws_instance_data_source
func TestAccAWSInstanceDataSource(t *testing.T) {
	for _, tc := range []struct {
		name           string
		testEntity     string
		testEntityType string
		entityName     string
		entityType     string
		currentType    string
		newType        string
		defaultType    string
	}{
		{
			name:           "Valid VM Recommendation",
			testEntity:     awsVMName,
			testEntityType: "VirtualMachine",
			entityName:     awsVMName,
			entityType:     "VirtualMachine",
			currentType:    awsVMCurrentType,
			newType:        awsVMNewType,
			defaultType:    awsVMDefaultType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_aws_instance" "test" {
							entity_name  = "` + tc.testEntity + `"
							default_instance_type = "` + tc.defaultType + `"
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_aws_instance.test", "entity_name", tc.entityName),
							resource.TestCheckResourceAttr("data.turbonomic_aws_instance.test", "entity_type", tc.entityType),
							resource.TestCheckResourceAttr("data.turbonomic_aws_instance.test", "current_instance_type", tc.currentType),
							resource.TestCheckResourceAttr("data.turbonomic_aws_instance.test", "new_instance_type", tc.newType),
							resource.TestCheckResourceAttr("data.turbonomic_aws_instance.test", "default_instance_type", tc.defaultType),
						),
					},
				},
			})
		})
	}
}
