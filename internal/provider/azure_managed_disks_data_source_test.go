// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	azureDiskSearchRespSuccess  = "azure_disk_search_response.json"
	azureDiskActionRespSuccess  = "azure_disk_action_response_tier_action.json"
	azureDiskActionUnknownType  = "azure_disk_action_response_unknown_type.json"
	azureDiskSearchUnknownType  = "azure_disk_search_response_unknown_type.json"
	azureDiskActionStorageAmt   = "azure_disk_action_response_storage_amount.json"
	azureDiskSearchRespEmpty    = "empty_array_response.json"
	azureDiskActionRespEmpty    = "empty_array_response.json"
	azureDiskName               = "testAzureManagedDisk"
	azureDiskCurrSizeStorage    = "Premium_LRS"
	azureDiskNewSizeStorage     = "StandardSSD_LRS"
	azureDiskDefaultSizeStorage = "Standard_LRS"
	azureDiskDefaultUnsupported = "Managed Standard Fastest Ever"
)

// Test Azure Managed disk data block logic by mocking valid turbo api response
func TestAzureManagedDiskDataSource(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionRespSuccess),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedCurrentStorageType string
		expectedNewStorageType     string
		expectedDefaultStorageType string
	}{
		{
			name:                       "Valid Volume Recommendation",
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
							    default_storage_account_type  = "` + tc.expectedDefaultStorageType + `"
						    }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "entity_type", "VirtualVolume"),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "current_storage_account_type", tc.expectedCurrentStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "new_storage_account_type", tc.expectedNewStorageType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_managed_disk.test", "default_storage_account_type", tc.expectedDefaultStorageType),
						),
					},
				},
			})
		})
	}
}

// Test Azure Managed disk data block for an entity that does not exist (initial creation in Terraform)
func TestAzureManagedDiskDataSourcNewInstance(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, azureDiskSearchRespEmpty),
		loadTestFile(t, azureDiskActionRespEmpty),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedCurrentStorageType string
		expectedNewStorageType     string
		expectedDefaultStorageType string
	}{
		{
			name:                       "Non existing Volume",
			testEntity:                 azureDiskName,
			expectedEntityName:         azureDiskName,
			expectedCurrentStorageType: azureDiskCurrSizeStorage,
			expectedNewStorageType:     azureDiskDefaultSizeStorage,
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
							    default_storage_account_type  = "` + tc.expectedDefaultStorageType + `"
						    }`,
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
	mockServer := createLocalServer(t,
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
		loadTestFile(t, azureDiskActionRespEmpty),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedCurrentStorageType string
		expectedNewStorageType     string
		expectedDefaultStorageType string
	}{
		{
			name:                       "Valid Volume with no actions",
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
							    default_storage_account_type  = "` + tc.expectedDefaultStorageType + `"
						    }`,
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

// Test Azure Managed disk where new type from Turbonomic is not supported
func TestAzureManagedDiskUnknownStorageTypeAction(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionUnknownType),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedCurrentStorageType string
		expectedNewStorageType     string
		expectedDefaultStorageType string
	}{
		{
			name:                       "Invalid new Volume type provided",
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
							    default_storage_account_type  = "` + tc.expectedDefaultStorageType + `"
						    }`,
						ExpectError: regexp.MustCompile(`Unknown storage type for new value`),
					},
				},
			})
		})
	}
}

// Test Azure Managed disk where current type from Turbonomic is not supported
func TestAzureManagedDiskUnknownStorageTypeSearch(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchUnknownType),
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionUnknownType),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedCurrentStorageType string
		expectedNewStorageType     string
		expectedDefaultStorageType string
	}{
		{
			name:                       "Invalid current Volume type provided",
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
							    default_storage_account_type  = "` + tc.expectedDefaultStorageType + `"
						    }`,
						ExpectError: regexp.MustCompile(`Unknown storage type for current value`),
					},
				},
			})
		})
	}
}

// Test Azure Managed disk where default type provided is not supported
func TestAzureManagedDiskUnknownDefaultType(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchUnknownType),
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionUnknownType),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

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
						ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
					},
				},
			})
		})
	}
}

// Test Azure Managed disk where the action is not a tier change
func TestAzureManagedDiskUnsupportedAction(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskSearchRespSuccess),
		loadTestFile(t, azureManagedDiskTestDataBaseDir, azureDiskActionStorageAmt),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedCurrentStorageType string
		expectedNewStorageType     string
		expectedDefaultStorageType string
	}{
		{
			name:                       "Test where action is not a tier change",
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
							    default_storage_account_type  = "` + tc.expectedDefaultStorageType + `"
						    }`,
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
