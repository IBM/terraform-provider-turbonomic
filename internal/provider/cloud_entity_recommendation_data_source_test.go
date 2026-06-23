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
	"net/http"
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

	searchEmptyRespFileLoc = "empty_array_response.json"
	actionEmptyRespFileLoc = "empty_array_response.json"

	// Azure specific constants
	azureVMName          = "test-vm"
	azureVMSize          = "Standard_B1s"
	azureVMCurrSize      = "standard_b1s"    // lowercase because the data source converts to lowercase
	azureVMNewSize       = "standard_d2s_v3" // lowercase because the data source converts to lowercase
	azureTestDataBaseDir = "cloud_data_source/azure"
)

// Tests data block logic by mocking valid turbo api response
func TestCloudDataSource(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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
	mockServer := mockTurboServer(t, []MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: "[]",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: "",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/login",
			ResponseCode: http.StatusOK,
			ResponseBody: `{"status":"ok"}`,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "",
			ResponseCode: http.StatusOK,
		},
	})

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
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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
			expectedNewInstanceType:     vmCurrSize,
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

// Tests when no default_size is specified
func TestCloudDataSourceWithoutDefaultSize(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, searchEmptyRespFileLoc),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, actionEmptyRespFileLoc),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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

// Tests no error while retrieving entity tags
func TestCloudDataSourceGetEntityTagsError(t *testing.T) {
	mockServer := mockTurboServer(t, []MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/login",
			ResponseCode: http.StatusOK,
			ResponseBody: `{"status":"ok"}`,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "",
			ResponseCode: http.StatusOK,
		},
	})

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	resourceConfig := `data "turbonomic_cloud_entity_recommendation" "test" {
		entity_name  = "testVM"
		entity_type  = "VirtualMachine"
		default_size = "t3a.micro"
	}`

	t.Run("no error while retrieving entity tags", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + resourceConfig,
					ExpectError: regexp.MustCompile(`unable to retrieve entity tags from turbonomic`),
				},
			},
		})
	})
}

// Tests error while tagging an entity
func TestCloudDataSourceTagEntityError(t *testing.T) {
	mockServer := mockTurboServer(t, []MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/login",
			ResponseCode: http.StatusOK,
			ResponseBody: `{"status":"ok"}`,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "[]",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "",
			ResponseCode: http.StatusOK,
		},
	})

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
					ExpectError: regexp.MustCompile(`unable to tag an entity in turbonomic`),
				},
			},
		})
	})
}

// Tests no error while tagging already tagged entity with discovered "optimized by" tag value
func TestCloudDataSourceTagEntityAlreadyTaggedDiscovered(t *testing.T) {
	mockServer := mockTurboServer(t, []MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/login",
			ResponseCode: http.StatusOK,
			ResponseBody: `{"status":"ok"}`,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: `[{"key": "turbonomic_optimized_by","values": ["turbonomic-terraform-provider"]}]`,
			ResponseCode: http.StatusOK,
		},
	})

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

// Tests no error while tagging already tagged entity with not discovered "optimized by" tag value
func TestCloudDataSourceTagEntityAlreadyTaggedNotDiscovered(t *testing.T) {
	mockServer := mockTurboServer(t, []MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/login",
			ResponseCode: http.StatusOK,
			ResponseBody: `{"status":"ok"}`,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: `[]`,
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "Entity service RPC call failed to complete request: INVALID_ARGUMENT: Trying to insert a tag with a key that already exists: turbonomic_optimized_by",
			ResponseCode: http.StatusBadRequest,
		},
	})

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

// Tests Azure VM recommendation
func TestAzureCloudDataSource(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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
			name:                        "Azure VM Recommendation",
			testEntity:                  azureVMName,
			testEntityType:              vmType,
			expectedEntityName:          azureVMName,
			expectedEntityType:          vmType,
			expectedCurrentInstanceType: azureVMCurrSize,
			expectedNewInstanceType:     azureVMNewSize,
			expectedDefaultSize:         azureVMSize,
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

func TestCloudEntityDataSourceDefaultSize(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default size length should be atleast 1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(vmConfig, vmName, vmType, ""),
					ExpectError: regexp.MustCompile(`Attribute default_size string length must be at least 1`),
				},
			},
		})
	})
}
