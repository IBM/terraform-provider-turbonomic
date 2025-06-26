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
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	vmConfig = `
	data "turbonomic_cloud_entity_recommendation" "test" {
		entity_name  = "%s"
		entity_type  = "%s"
		default_size = "%s"
	}
	`

	vmName     = "testVM"
	vmType     = "VirtualMachine"
	vmCurrSize = "t2.micro"
	vmNewSize  = "t3a.micro"

	searchRespTestData        = "search_success_response.json"
	validVmActionRespTestData = "action_success_response.json"
	emptyActionRespTestData   = "empty_array_response.json"

	entityTagsRespTestData = "entity_tags_success_response.json"
	entityTagRespTestData  = "entity_tag_success_response.json"

	searchEmptyRespFileLoc = "empty_array_response.json"
	actionEmptyRespFileLoc = "empty_array_response.json"
)

// Tests data block logic by mocking valid turbo api response
func TestCloudDataSource(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
		loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

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
			testEntityType:              vmType,
			expectedEntityName:          vmName,
			expectedEntityType:          vmType,
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
						Config: providerConfig + fmt.Sprintf(vmConfig, tc.testEntity, tc.testEntityType, tc.expectedDefaultSize),
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

// Tests default_size field in data block by mocking empty turbo api search response
func TestDefaultSizeOnEmptySearch(t *testing.T) {
	mockServer := createLocalServer(t,
		"[]",
		"",
		"",
		"")
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

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
			name:                    "Empty VM Search",
			testEntity:              vmName,
			testEntityType:          vmType,
			expectedEntityName:      vmName,
			expectedEntityType:      vmType,
			expectedNewInstanceType: vmNewSize,
			expectedDefaultSize:     vmNewSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(vmConfig, tc.testEntity, tc.testEntityType, tc.expectedDefaultSize),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckNoResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_type"),
							resource.TestCheckNoResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "current_instance_type"),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "new_instance_type", tc.expectedNewInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "default_size", tc.expectedDefaultSize),
						),
					},
				},
			})
		})
	}
}

// Tests default_size field in data block by mocking empty turbo api action response
func TestDefaultSizeInCloudDataSource(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
		loadTestFile(t, emptyActionRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))

	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

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
			testEntityType:              vmType,
			expectedEntityName:          vmName,
			expectedEntityType:          vmType,
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
						Config: providerConfig + fmt.Sprintf(vmConfig, tc.testEntity, tc.testEntityType, tc.expectedDefaultSize),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_type", tc.expectedEntityType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "current_instance_type", tc.expectedCurrentInstanceType),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "new_instance_type", tc.expectedCurrentInstanceType),
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
	mockServer := createLocalServer(t,
		loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
		loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

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
			testEntityType:              vmType,
			expectedEntityName:          vmName,
			expectedEntityType:          vmType,
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

// Tests default_size field in data block by when search response is empty
func TestDefaultEmptySearchResp(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, searchEmptyRespFileLoc),
		loadTestFile(t, actionEmptyRespFileLoc),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
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
			name:                        "Empty VM Search",
			testEntity:                  vmName,
			testEntityType:              vmType,
			expectedEntityName:          vmName,
			expectedEntityType:          vmType,
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
							resource.TestCheckNoResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "entity_type"),
							resource.TestCheckNoResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "current_instance_type"),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "new_instance_type", tc.expectedDefaultSize),
							resource.TestCheckResourceAttr("data.turbonomic_cloud_entity_recommendation.test", "default_size", tc.expectedDefaultSize),
						),
					},
				},
			})
		})
	}
}

// Tests error while retrieving entity tags
func TestCloudDataSourceGetEntityTagsError(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
		loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
		"",
		"")
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	resourceConfig := `data "turbonomic_cloud_entity_recommendation" "test" {
		entity_name  = "testVM"
		entity_type  = "VirtualMachine"
		default_size = "t3a.micro"
	}`

	t.Run("Valid VM Recommendation", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + resourceConfig,
					ExpectError: regexp.MustCompile(`Unable to retrieve entity tags from Turbonomic`),
				},
			},
		})
	})
}

// Tests error while tagging an entity
func TestCloudDataSourceTagEntityError(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
		loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
		"[]",
		"")
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	resourceConfig := `data "turbonomic_cloud_entity_recommendation" "test" {
		entity_name  = "testVM"
		entity_type  = "VirtualMachine"
		default_size = "t3a.micro"
	}`

	t.Run("Valid VM Recommendation", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + resourceConfig,
					ExpectError: regexp.MustCompile(`Unable to tag an entity in Turbonomic`),
				},
			},
		})
	})
}

// Tests error while tagging already tagged entity with different "optimized by" tag value
func TestCloudDataSourceTagEntity(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
		loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
		`[{"key": "turbonomic_optimized_by","values": ["Different from Turbonomic Terraform Provider"]}]`,
		"INVALID_ARGUMENT: Trying to insert a tag with a key that already exists")
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	resourceConfig := `data "turbonomic_cloud_entity_recommendation" "test" {
		entity_name  = "testVM"
		entity_type  = "VirtualMachine"
		default_size = "t3a.micro"
	}`

	t.Run("Valid VM Recommendation", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + resourceConfig,
					ExpectError: regexp.MustCompile(`Unable to tag an entity in Turbonomic`),
				},
			},
		})
	})
}
