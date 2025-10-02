// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

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
	azureDiskConfig = `
	data "turbonomic_azurerm_managed_disk" "test" {
		entity_name       				  = "%s"
		default_storage_account_type      = "%s"
		default_disk_size_gb      		  = %d
		default_disk_iops_read_write      = %d
		default_disk_mbps_read_write      = %d
	}
	`

	azureDiskSearchRespSuccess    = "azure_disk_search_response.json"
	azureDiskActionRespSuccess    = "azure_disk_action_response_tier_action.json"
	azureDiskActionUnknownType    = "azure_disk_action_response_unknown_type.json"
	azureDiskSearchUnknownType    = "azure_disk_search_response_unknown_type.json"
	azureDiskActionStorageAmt     = "azure_disk_action_response_storage_amount.json"
	azureDiskSearchRespEmpty      = "empty_array_response.json"
	azureDiskActionRespEmpty      = "empty_array_response.json"
	azureDiskStatsResp            = "azure_disk_stats_valid_response.json"
	azureDiskName                 = "testAzureManagedDisk"
	azureDiskCurrSizeStorage      = "Premium_LRS"
	azureDiskNewSizeStorage       = "StandardSSD_LRS"
	azureDiskDefaultSizeStorage   = "Standard_LRS"
	azureDiskDefaultUnsupported   = "Managed Standard Fastest Ever"
	azureDiskCurrSizeGb           = 30
	azureDiskNewSizeGb            = 30
	azureDiskDefaultSizeGb        = 30
	azureDiskCurrIopsReadWrite    = 500
	azureDiskNewIopsReadWrite     = 500
	azureDiskDefaultIopsReadWrite = 500
	azureDiskCurrMbpsReadWrite    = 100
	azureDiskNewMbpsReadWrite     = 60
	azureDiskDefaultMbpsReadWrite = 98

	// Config templates for vendor ID tests
	azureVirtualVolumeConfigWithVendorId = `
	data "turbonomic_azurerm_managed_disk" "test" {
		vendor_id    = "%s"
		default_storage_account_type = "%s"
		default_disk_size_gb      		  = %d
		default_disk_iops_read_write      = %d
		default_disk_mbps_read_write      = %d
	}
	`

	azureVirtualVolumeConfigWithVendorIdAndName = `
	data "turbonomic_azurerm_managed_disk" "test" {
		vendor_id    = "%s"
		default_storage_account_type = "%s"
		entity_name  = "%s"
		default_disk_size_gb      		  = %d
		default_disk_iops_read_write      = %d
		default_disk_mbps_read_write      = %d
	}
	`

	azureVirtualVolumeConfigNoIdentifiers = `
	data "turbonomic_azurerm_managed_disk" "test" {
		default_storage_account_type = "%s"
	}
	`
	volumeEntityType = "VirtualVolume"
	// Data source reference
	azureVirtualVolumeDataSourceRef = "data.turbonomic_azurerm_managed_disk.test"
)

// Test Azure Managed disk data block logic by mocking valid turbo api response
func TestAzureManagedDiskDataSource(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                         string
		testEntity                   string
		expectedEntityName           string
		expectedCurrentStorageType   string
		expectedNewStorageType       string
		expectedDefaultStorageType   string
		expectedCurrentSizeGb        int64
		expectedNewSizeGb            int64
		expectedDefautSizeGb         int64
		expectedCurrentIopsReadWrite int64
		expectedNewIopsReadWrite     int64
		expectedDefaultIopsReadWrite int64
		expectedCurrentMbpsReadWrite int64
		expectedNewMbpsReadWrite     int64
		expectedDefaultMbpsReadWrite int64
	}{
		{
			name:                         "Valid Volume Recommendation",
			testEntity:                   azureDiskName,
			expectedEntityName:           azureDiskName,
			expectedCurrentStorageType:   azureDiskCurrSizeStorage,
			expectedNewStorageType:       azureDiskNewSizeStorage,
			expectedDefaultStorageType:   azureDiskDefaultSizeStorage,
			expectedCurrentSizeGb:        azureDiskCurrSizeGb,
			expectedNewSizeGb:            azureDiskNewSizeGb,
			expectedDefautSizeGb:         azureDiskDefaultSizeGb,
			expectedCurrentIopsReadWrite: azureDiskCurrIopsReadWrite,
			expectedNewIopsReadWrite:     azureDiskNewIopsReadWrite,
			expectedDefaultIopsReadWrite: azureDiskDefaultIopsReadWrite,
			expectedCurrentMbpsReadWrite: azureDiskCurrMbpsReadWrite,
			expectedNewMbpsReadWrite:     azureDiskNewMbpsReadWrite,
			expectedDefaultMbpsReadWrite: azureDiskDefaultMbpsReadWrite,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(azureDiskConfig, tc.testEntity, tc.expectedDefaultStorageType, tc.expectedDefautSizeGb, tc.expectedDefaultIopsReadWrite, tc.expectedDefaultMbpsReadWrite),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_type", "VirtualVolume"),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "current_storage_account_type", tc.expectedCurrentStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "new_storage_account_type", tc.expectedNewStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "default_storage_account_type", tc.expectedDefaultStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "current_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedCurrentIopsReadWrite)),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "new_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedNewIopsReadWrite)),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "default_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedDefaultIopsReadWrite)),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "current_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedCurrentMbpsReadWrite)),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "new_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedNewMbpsReadWrite)),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "default_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedDefaultMbpsReadWrite)),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "current_disk_size_gb", fmt.Sprintf("%d", tc.expectedCurrentSizeGb)),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "new_disk_size_gb", fmt.Sprintf("%d", tc.expectedNewSizeGb)),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "default_disk_size_gb", fmt.Sprintf("%d", tc.expectedDefautSizeGb)),
						),
					},
				},
			})
		})
	}
}

// Test Azure Managed disk data block for an entity that does not exist (initial creation in Terraform)
func TestAzureManagedDiskDataSourcNewInstance(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureDiskSearchRespEmpty),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureDiskActionRespEmpty),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                         string
		testEntity                   string
		expectedEntityName           string
		expectedCurrentStorageType   string
		expectedNewStorageType       string
		expectedDefaultStorageType   string
		expectedDefautSizeGb         int64
		expectedDefaultIopsReadWrite int64
		expectedDefaultMbpsReadWrite int64
	}{
		{
			name:                         "Non existing Volume",
			testEntity:                   azureDiskName,
			expectedEntityName:           azureDiskName,
			expectedCurrentStorageType:   azureDiskCurrSizeStorage,
			expectedNewStorageType:       azureDiskDefaultSizeStorage,
			expectedDefaultStorageType:   azureDiskDefaultSizeStorage,
			expectedDefautSizeGb:         azureDiskDefaultSizeGb,
			expectedDefaultIopsReadWrite: azureDiskDefaultIopsReadWrite,
			expectedDefaultMbpsReadWrite: azureDiskDefaultMbpsReadWrite,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(azureDiskConfig, tc.testEntity, tc.expectedDefaultStorageType, tc.expectedDefautSizeGb, tc.expectedDefaultIopsReadWrite, tc.expectedDefaultMbpsReadWrite),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckNoResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_type"),
							resource.TestCheckNoResourceAttr("data.turbonomic_azurerm_managed_disk.test", "current_storage_account_type"),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "new_storage_account_type", tc.expectedDefaultStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "default_storage_account_type", tc.expectedDefaultStorageType),
						),
					},
				},
			})
		})
	}
}

// Test Azure Managed disk data block for an entity that has no actions
func TestAzureManagedDiskDataSourcNoActions(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureDiskActionRespEmpty),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                         string
		testEntity                   string
		expectedEntityName           string
		expectedCurrentStorageType   string
		expectedNewStorageType       string
		expectedDefaultStorageType   string
		expectedDefautSizeGb         int64
		expectedDefaultIopsReadWrite int64
		expectedDefaultMbpsReadWrite int64
	}{
		{
			name:                         "Valid Volume with no actions",
			testEntity:                   azureDiskName,
			expectedEntityName:           azureDiskName,
			expectedCurrentStorageType:   azureDiskCurrSizeStorage,
			expectedNewStorageType:       azureDiskNewSizeStorage,
			expectedDefaultStorageType:   azureDiskDefaultSizeStorage,
			expectedDefautSizeGb:         azureDiskDefaultSizeGb,
			expectedDefaultIopsReadWrite: azureDiskDefaultIopsReadWrite,
			expectedDefaultMbpsReadWrite: azureDiskDefaultMbpsReadWrite,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(azureDiskConfig, tc.testEntity, tc.expectedDefaultStorageType, tc.expectedDefautSizeGb, tc.expectedDefaultIopsReadWrite, tc.expectedDefaultMbpsReadWrite),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_type", "VirtualVolume"),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "current_storage_account_type", tc.expectedCurrentStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "new_storage_account_type", tc.expectedCurrentStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "default_storage_account_type", tc.expectedDefaultStorageType),
						),
					},
				},
			})
		})
	}
}

// // Test Azure Managed disk where new type from Turbonomic is not supported
// func TestAzureManagedDiskUnknownStorageTypeAction(t *testing.T) {
// 	mockServer := mockTurboServer(t, append([]MockRoute{
// 		{
// 			Method:       http.MethodPost,
// 			Path:         "/api/v3/search",
// 			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
// 			ResponseCode: http.StatusOK,
// 		},
// 		{
// 			Method:       http.MethodPost,
// 			Path:         "/api/v3/entities/{id}/actions",
// 			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionUnknownType),
// 			ResponseCode: http.StatusOK,
// 		},
// 		{
// 			Method:       http.MethodPost,
// 			Path:         "/api/v3/stats/{id}",
// 			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
// 			ResponseCode: http.StatusOK,
// 		},
// 	}, LoginAndTagRoutes(t)...))
// 	defer mockServer.Close()

// 	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

// 	for _, tc := range []struct {
// 		name                         string
// 		testEntity                   string
// 		expectedEntityName           string
// 		expectedCurrentStorageType   string
// 		expectedNewStorageType       string
// 		expectedDefaultStorageType   string
// 		expectedDefautSizeGb         int64
// 		expectedDefaultIopsReadWrite int64
// 		expectedDefaultMbpsReadWrite int64
// 	}{
// 		{
// 			name:                         "Invalid new Volume type provided",
// 			testEntity:                   azureDiskName,
// 			expectedEntityName:           azureDiskName,
// 			expectedCurrentStorageType:   azureDiskCurrSizeStorage,
// 			expectedNewStorageType:       azureDiskNewSizeStorage,
// 			expectedDefaultStorageType:   azureDiskDefaultSizeStorage,
// 			expectedDefautSizeGb:         azureDiskDefaultSizeGb,
// 			expectedDefaultIopsReadWrite: azureDiskDefaultIopsReadWrite,
// 			expectedDefaultMbpsReadWrite: azureDiskDefaultMbpsReadWrite,
// 		},
// 	} {
// 		t.Run(tc.name, func(t *testing.T) {
// 			resource.Test(t, resource.TestCase{
// 				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 				Steps: []resource.TestStep{
// 					{
// 						Config:      providerConfig + fmt.Sprintf(azureDiskConfig, tc.testEntity, tc.expectedDefaultStorageType, tc.expectedDefautSizeGb, tc.expectedDefaultIopsReadWrite, tc.expectedDefaultMbpsReadWrite),
// 						ExpectError: regexp.MustCompile(`Unknown storage type for new value`),
// 					},
// 				},
// 			})
// 		})
// 	}
// }

// // Test Azure Managed disk where current type from Turbonomic is not supported
// func TestAzureManagedDiskUnknownStorageTypeSearch(t *testing.T) {
// 	mockServer := mockTurboServer(t, append([]MockRoute{
// 		{
// 			Method:       http.MethodPost,
// 			Path:         "/api/v3/search",
// 			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchUnknownType),
// 			ResponseCode: http.StatusOK,
// 		},
// 		{
// 			Method:       http.MethodPost,
// 			Path:         "/api/v3/entities/{id}/actions",
// 			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionUnknownType),
// 			ResponseCode: http.StatusOK,
// 		},
// 	}, LoginAndTagRoutes(t)...))
// 	defer mockServer.Close()

// 	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

// 	for _, tc := range []struct {
// 		name                         string
// 		testEntity                   string
// 		expectedEntityName           string
// 		expectedCurrentStorageType   string
// 		expectedNewStorageType       string
// 		expectedDefaultStorageType   string
// 		expectedDefautSizeGb         int64
// 		expectedDefaultIopsReadWrite int64
// 		expectedDefaultMbpsReadWrite int64
// 	}{
// 		{
// 			name:                         "Invalid current Volume type provided",
// 			testEntity:                   azureDiskName,
// 			expectedEntityName:           azureDiskName,
// 			expectedCurrentStorageType:   azureDiskCurrSizeStorage,
// 			expectedNewStorageType:       azureDiskNewSizeStorage,
// 			expectedDefaultStorageType:   azureDiskDefaultSizeStorage,
// 			expectedDefautSizeGb:         azureDiskDefaultSizeGb,
// 			expectedDefaultIopsReadWrite: azureDiskDefaultIopsReadWrite,
// 			expectedDefaultMbpsReadWrite: azureDiskDefaultMbpsReadWrite,
// 		},
// 	} {
// 		t.Run(tc.name, func(t *testing.T) {
// 			resource.Test(t, resource.TestCase{
// 				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 				Steps: []resource.TestStep{
// 					{
// 						Config:      providerConfig + fmt.Sprintf(azureDiskConfig, tc.testEntity, tc.expectedDefaultStorageType, tc.expectedDefautSizeGb, tc.expectedDefaultIopsReadWrite, tc.expectedDefaultMbpsReadWrite),
// 						ExpectError: regexp.MustCompile(`Unknown storage type for current value`),
// 					},
// 				},
// 			})
// 		})
// 	}
// }

// Test Azure Managed disk where default type provided is not supported
func TestAzureManagedDiskUnknownDefaultType(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchUnknownType),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionUnknownType),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedCurrentStorageType string
		expectedNewStorageType     string
		expectedDefaultStorageType string
	}{
		{
			name:                       "Invalid default Volume type provided",
			testEntity:                 azureDiskName,
			expectedEntityName:         azureDiskName,
			expectedCurrentStorageType: azureDiskCurrSizeStorage,
			expectedNewStorageType:     azureDiskNewSizeStorage,
			expectedDefaultStorageType: azureDiskDefaultSizeStorage,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_azurerm_managed_disk" "test" {
							    entity_name  = "` + tc.testEntity + `"
							    default_storage_account_type  = "` + azureDiskDefaultUnsupported + `"
						    }`,
						ExpectError: regexp.MustCompile(`Attribute default_storage_account_type value must be one of:`),
					},
				},
			})
		})
	}
}

// Test Azure Managed disk where the action is not a tier change
func TestAzureManagedDiskUnsupportedAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionStorageAmt),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                         string
		testEntity                   string
		expectedEntityName           string
		expectedCurrentStorageType   string
		expectedNewStorageType       string
		expectedDefaultStorageType   string
		expectedDefautSizeGb         int64
		expectedDefaultIopsReadWrite int64
		expectedDefaultMbpsReadWrite int64
	}{
		{
			name:                         "Test where action is not a tier change",
			testEntity:                   azureDiskName,
			expectedEntityName:           azureDiskName,
			expectedCurrentStorageType:   azureDiskCurrSizeStorage,
			expectedNewStorageType:       azureDiskNewSizeStorage,
			expectedDefaultStorageType:   azureDiskDefaultSizeStorage,
			expectedDefautSizeGb:         azureDiskDefaultSizeGb,
			expectedDefaultIopsReadWrite: azureDiskDefaultIopsReadWrite,
			expectedDefaultMbpsReadWrite: azureDiskDefaultMbpsReadWrite,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(azureDiskConfig, tc.testEntity, tc.expectedDefaultStorageType, tc.expectedDefautSizeGb, tc.expectedDefaultIopsReadWrite, tc.expectedDefaultMbpsReadWrite),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_type", "VirtualVolume"),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "current_storage_account_type", tc.expectedCurrentStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "new_storage_account_type", tc.expectedCurrentStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "default_storage_account_type", tc.expectedDefaultStorageType),
						),
					},
				},
			})
		})
	}
}

func TestAzureManagedDiskDefaultDiskSize(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionStorageAmt),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default disk size gb atleast 0", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(azureDiskConfig, azureDiskName, azureDiskDefaultSizeStorage, -1, azureDiskDefaultIopsReadWrite, azureDiskDefaultMbpsReadWrite),
					ExpectError: regexp.MustCompile(`Attribute default_disk_size_gb value must be at least 0`),
				},
			},
		})
	})
}

func TestAzureManagedDiskDefaultIops(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionStorageAmt),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default disk iops read write atleast 0", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(azureDiskConfig, azureDiskName, azureDiskDefaultSizeStorage, azureDiskDefaultSizeGb, -1, azureDiskDefaultMbpsReadWrite),
					ExpectError: regexp.MustCompile(`Attribute default_disk_iops_read_write value must be at least 0`),
				},
			},
		})
	})
}

func TestAzureManagedDiskDefaultReadWrite(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionStorageAmt),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default disk mbps read write atleast 0", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(azureDiskConfig, azureDiskName, azureDiskDefaultSizeStorage, azureDiskDefaultSizeGb, azureDiskDefaultIopsReadWrite, -1),
					ExpectError: regexp.MustCompile(`Attribute default_disk_mbps_read_write value must be at least 0`),
				},
			},
		})
	})
}

// Test for a valid entity using vendor_id
func TestAzurermManagedDiskDataSourceWithVendorId(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                              string
		testVendorId                      string
		expectedVendorId                  string
		expectedDefaultStorageAccountType string
		expectedCurrentStorageAccountType string
		expectedNewStorageAccountType     string
		expectedDefaultDiskIopsReadWrite  int
		expectedCurrentDiskIopsReadWrite  int
		expectedNewDiskIopsReadWrite      int
		expectedDefaultDiskMbpsReadWrite  int
		expectedCurrentDiskMbpsReadWrite  int
		expectedNewDiskMbpsReadWrite      int
		expectedDefaultDiskSizeGb         int
		expectedCurrentDiskSizeGb         int
		expectedNewDiskSizeGb             int
	}{
		{
			name:                              "Valid VirtualVolume recommendation with vendor_id",
			testVendorId:                      testVendorId,
			expectedVendorId:                  testVendorId,
			expectedDefaultStorageAccountType: azureDiskDefaultSizeStorage,
			expectedCurrentStorageAccountType: azureDiskCurrSizeStorage,
			expectedNewStorageAccountType:     azureDiskNewSizeStorage,
			expectedDefaultDiskIopsReadWrite:  azureDiskDefaultIopsReadWrite,
			expectedCurrentDiskIopsReadWrite:  azureDiskCurrIopsReadWrite,
			expectedNewDiskIopsReadWrite:      azureDiskDefaultIopsReadWrite,
			expectedDefaultDiskMbpsReadWrite:  azureDiskDefaultMbpsReadWrite,
			expectedCurrentDiskMbpsReadWrite:  azureDiskCurrMbpsReadWrite,
			expectedNewDiskMbpsReadWrite:      azureDiskNewMbpsReadWrite,
			expectedDefaultDiskSizeGb:         azureDiskDefaultSizeGb,
			expectedCurrentDiskSizeGb:         azureDiskCurrSizeGb,
			expectedNewDiskSizeGb:             azureDiskNewSizeGb,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(azureVirtualVolumeConfigWithVendorId, tc.testVendorId, tc.expectedDefaultStorageAccountType, tc.expectedDefaultDiskSizeGb, tc.expectedDefaultDiskIopsReadWrite, tc.expectedDefaultDiskMbpsReadWrite),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "vendor_id", tc.expectedVendorId),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "entity_type", volumeEntityType),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_storage_account_type", tc.expectedDefaultStorageAccountType),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "current_storage_account_type", tc.expectedCurrentStorageAccountType),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_storage_account_type", tc.expectedNewStorageAccountType),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedDefaultDiskIopsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "current_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedCurrentDiskIopsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedNewDiskIopsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedDefaultDiskMbpsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "current_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedCurrentDiskMbpsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedNewDiskMbpsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_disk_size_gb", fmt.Sprintf("%d", tc.expectedDefaultDiskSizeGb)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "current_disk_size_gb", fmt.Sprintf("%d", tc.expectedCurrentDiskSizeGb)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_disk_size_gb", fmt.Sprintf("%d", tc.expectedNewDiskSizeGb)),
						),
					},
				},
			})
		})
	}
}

// Test for a valid entity using vendor_id and entity_name
func TestAzurermManagedDiskDataSourceWithVendorIdAndName(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                              string
		testVendorId                      string
		testEntity                        string
		expectedVendorId                  string
		expectedEntityName                string
		expectedDefaultStorageAccountType string
		expectedCurrentStorageAccountType string
		expectedNewStorageAccountType     string
		expectedDefaultDiskIopsReadWrite  int
		expectedCurrentDiskIopsReadWrite  int
		expectedNewDiskIopsReadWrite      int
		expectedDefaultDiskMbpsReadWrite  int
		expectedCurrentDiskMbpsReadWrite  int
		expectedNewDiskMbpsReadWrite      int
		expectedDefaultDiskSizeGb         int
		expectedCurrentDiskSizeGb         int
		expectedNewDiskSizeGb             int
	}{
		{
			name:                              "Valid VirtualVolume recommendation with vendor_id and entity_name",
			testVendorId:                      testVendorId,
			expectedVendorId:                  testVendorId,
			testEntity:                        "test-volume",
			expectedEntityName:                "test-volume",
			expectedDefaultStorageAccountType: azureDiskDefaultSizeStorage,
			expectedCurrentStorageAccountType: azureDiskCurrSizeStorage,
			expectedNewStorageAccountType:     azureDiskNewSizeStorage,
			expectedDefaultDiskIopsReadWrite:  azureDiskDefaultIopsReadWrite,
			expectedCurrentDiskIopsReadWrite:  azureDiskCurrIopsReadWrite,
			expectedNewDiskIopsReadWrite:      azureDiskDefaultIopsReadWrite,
			expectedDefaultDiskMbpsReadWrite:  azureDiskDefaultMbpsReadWrite,
			expectedCurrentDiskMbpsReadWrite:  azureDiskCurrMbpsReadWrite,
			expectedNewDiskMbpsReadWrite:      azureDiskNewMbpsReadWrite,
			expectedDefaultDiskSizeGb:         azureDiskDefaultSizeGb,
			expectedCurrentDiskSizeGb:         azureDiskCurrSizeGb,
			expectedNewDiskSizeGb:             azureDiskNewSizeGb,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(azureVirtualVolumeConfigWithVendorIdAndName, tc.testVendorId, tc.expectedDefaultStorageAccountType, tc.testEntity, tc.expectedDefaultDiskSizeGb, tc.expectedDefaultDiskIopsReadWrite, tc.expectedDefaultDiskMbpsReadWrite),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "vendor_id", tc.expectedVendorId),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "entity_type", volumeEntityType),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_storage_account_type", tc.expectedDefaultStorageAccountType),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "current_storage_account_type", tc.expectedCurrentStorageAccountType),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_storage_account_type", tc.expectedNewStorageAccountType),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedDefaultDiskIopsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "current_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedCurrentDiskIopsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedNewDiskIopsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedDefaultDiskMbpsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "current_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedCurrentDiskMbpsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedNewDiskMbpsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_disk_size_gb", fmt.Sprintf("%d", tc.expectedDefaultDiskSizeGb)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "current_disk_size_gb", fmt.Sprintf("%d", tc.expectedCurrentDiskSizeGb)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_disk_size_gb", fmt.Sprintf("%d", tc.expectedNewDiskSizeGb)),
						),
					},
				},
			})
		})
	}
}

// Test for when neither entity_name nor vendor_id is provided
func TestAzurermManagedDiskDataSourceWithNoIdentifiers(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionStorageAmt),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	t.Run("Error when neither entity_name nor vendor_id is provided", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(azureVirtualVolumeConfigNoIdentifiers, azureDiskDefaultSizeStorage),
					ExpectError: regexp.MustCompile(`At least one of these attributes must be configured`),
				},
			},
		})
	})

}

// Test for an invalid vendor_id
func TestAzurermManagedDiskDataSourceWithInvalidVendorId(t *testing.T) {
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
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                              string
		testVendorId                      string
		expectedVendorId                  string
		expectedDefaultStorageAccountType string
		expectedCurrentStorageAccountType string
		expectedNewStorageAccountType     string
		expectedDefaultDiskIopsReadWrite  int
		expectedCurrentDiskIopsReadWrite  int
		expectedNewDiskIopsReadWrite      int
		expectedDefaultDiskMbpsReadWrite  int
		expectedCurrentDiskMbpsReadWrite  int
		expectedNewDiskMbpsReadWrite      int
		expectedDefaultDiskSizeGb         int
		expectedCurrentDiskSizeGb         int
		expectedNewDiskSizeGb             int
	}{
		{
			name:                              "Invalid vendor_id",
			testVendorId:                      testVendorId,
			expectedVendorId:                  testVendorId,
			expectedDefaultStorageAccountType: azureDiskDefaultSizeStorage,
			expectedCurrentStorageAccountType: azureDiskCurrSizeStorage,
			expectedNewStorageAccountType:     azureDiskDefaultSizeStorage,
			expectedDefaultDiskIopsReadWrite:  azureDiskDefaultIopsReadWrite,
			expectedCurrentDiskIopsReadWrite:  azureDiskCurrIopsReadWrite,
			expectedNewDiskIopsReadWrite:      azureDiskDefaultIopsReadWrite,
			expectedDefaultDiskMbpsReadWrite:  azureDiskDefaultMbpsReadWrite,
			expectedCurrentDiskMbpsReadWrite:  azureDiskCurrMbpsReadWrite,
			expectedNewDiskMbpsReadWrite:      azureDiskDefaultMbpsReadWrite,
			expectedDefaultDiskSizeGb:         azureDiskDefaultSizeGb,
			expectedCurrentDiskSizeGb:         azureDiskCurrSizeGb,
			expectedNewDiskSizeGb:             azureDiskDefaultSizeGb,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(azureVirtualVolumeConfigWithVendorId, tc.testVendorId, tc.expectedDefaultStorageAccountType, tc.expectedDefaultDiskSizeGb, tc.expectedDefaultDiskIopsReadWrite, tc.expectedDefaultDiskMbpsReadWrite),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "vendor_id", tc.expectedVendorId),
							resource.TestCheckNoResourceAttr(azureVirtualVolumeDataSourceRef, "entity_type"),
							resource.TestCheckNoResourceAttr(azureVirtualVolumeDataSourceRef, "current_storage_account_type"),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_storage_account_type", tc.expectedNewStorageAccountType),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_storage_account_type", tc.expectedDefaultStorageAccountType),
							resource.TestCheckNoResourceAttr(azureVirtualVolumeDataSourceRef, "current_disk_iops_read_write"),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedNewDiskIopsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_disk_iops_read_write", fmt.Sprintf("%d", tc.expectedDefaultDiskIopsReadWrite)),
							resource.TestCheckNoResourceAttr(azureVirtualVolumeDataSourceRef, "current_disk_mbps_read_write"),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedNewDiskMbpsReadWrite)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_disk_mbps_read_write", fmt.Sprintf("%d", tc.expectedDefaultDiskMbpsReadWrite)),
							resource.TestCheckNoResourceAttr(azureVirtualVolumeDataSourceRef, "current_disk_size_gb"),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "new_disk_size_gb", fmt.Sprintf("%d", tc.expectedNewDiskSizeGb)),
							resource.TestCheckResourceAttr(azureVirtualVolumeDataSourceRef, "default_disk_size_gb", fmt.Sprintf("%d", tc.expectedDefaultDiskSizeGb)),
						),
					},
				},
			})
		})
	}
}
