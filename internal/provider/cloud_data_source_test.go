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
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	searchRespFileLoc      = "./testdata/search_success_response.json"
	actionRespFileLoc      = "./testdata/action_success_response.json"
	actionEmptyRespFileLoc = "./testdata/action_empty_response.json"
	config                 = `provider "turbonomic" {
	username = "administrator"
	password = "12345"
	hostname = "%s"
	skipverify = true
	}
	`
	vmName     = "testVM"
	vmCurrSize = "t2.micro"
	vmNewSize  = "t3a.micro"
	entityType = "VirtualMachine"
)

// Tests data block logic by mocking valid turbo api response
func TestCloudDataSource(t *testing.T) {
	mockServer := createLocalServer(t, loadTestFile(t, searchRespFileLoc), loadTestFile(t, actionRespFileLoc))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                        string
		testEntity                  string
		testEntityType              string
		expectedEntityName          string
		expectedEntityType          string
		expectedCurrentInstanceType string
		expectedNewInstanceType     string
		expectedDefaultSize         string
	}{
		{
			name:                        "Valid VM Recommendation",
			testEntity:                  vmName,
			testEntityType:              entityType,
			expectedEntityName:          vmName,
			expectedEntityType:          entityType,
			expectedCurrentInstanceType: vmCurrSize,
			expectedNewInstanceType:     vmNewSize,
			expectedDefaultSize:         vmNewSize,
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
							default_size = "` + tc.expectedDefaultSize + `"
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_type", tc.expectedEntityType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "current_instance_type", tc.expectedCurrentInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "new_instance_type", tc.expectedNewInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "default_size", tc.expectedDefaultSize),
						),
					},
				},
			})
		})
	}
}

// Tests default_size field in data block by mocking empty turbo api response
func TestDefaultSizeInCloudDataSource(t *testing.T) {
	mockServer := createLocalServer(t, loadTestFile(t, searchRespFileLoc), loadTestFile(t, actionEmptyRespFileLoc))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                        string
		testEntity                  string
		testEntityType              string
		expectedEntityName          string
		expectedEntityType          string
		expectedCurrentInstanceType string
		expectedNewInstanceType     string
		expectedDefaultSize         string
	}{
		{
			name:                        "Empty VM Recommendation",
			testEntity:                  vmName,
			testEntityType:              entityType,
			expectedEntityName:          vmName,
			expectedEntityType:          entityType,
			expectedCurrentInstanceType: vmCurrSize,
			expectedNewInstanceType:     vmNewSize,
			expectedDefaultSize:         vmNewSize,
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
							default_size = "` + tc.expectedDefaultSize + `"
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_type", tc.expectedEntityType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "current_instance_type", tc.expectedCurrentInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "new_instance_type", tc.expectedNewInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "default_size", tc.expectedDefaultSize),
						),
					},
				},
			})
		})
	}
}

// Tests when no default_size is specified
func TestCloudDataSourceWithoutDefaultSize(t *testing.T) {
	mockServer := createLocalServer(t, loadTestFile(t, searchRespFileLoc), loadTestFile(t, actionRespFileLoc))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                        string
		testEntity                  string
		testEntityType              string
		expectedEntityName          string
		expectedEntityType          string
		expectedCurrentInstanceType string
		expectedNewInstanceType     string
	}{
		{
			name:                        "Valid VM Recommendation",
			testEntity:                  vmName,
			testEntityType:              entityType,
			expectedEntityName:          vmName,
			expectedEntityType:          entityType,
			expectedCurrentInstanceType: vmCurrSize,
			expectedNewInstanceType:     vmNewSize,
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
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_type", tc.expectedEntityType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "current_instance_type", tc.expectedCurrentInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "new_instance_type", tc.expectedNewInstanceType),
						),
					},
				},
			})
		})
	}
}
