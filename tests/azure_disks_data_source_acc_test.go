// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	azureDisksConfig = `
        data turbonomic_azurerm_managed_disk test {
            entity_name            	     = "%s"
            default_storage_account_type = "%s"
        }
	`
	azureDisksDataSourceRef = "data.turbonomic_azurerm_managed_disk.test"
)

// Tests `turbonomic_azurerm_managed_disk` data block
func TestAccAzureDisksDataSource(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		testEntity                 string
		testEntityType             string
		entityName                 string
		entityType                 string
		currentStorageAccountsType string
		newStorageAccountsType     string
		defaultStorageAccountsType string
	}{
		{
			name:                       "Valid Azure Disk Recommendation",
			testEntity:                 azureDiskName,
			testEntityType:             "VirtualVolume",
			entityName:                 azureDiskName,
			entityType:                 "VirtualVolume",
			currentStorageAccountsType: azureDiskCurrentType,
			newStorageAccountsType:     azureDiskNewType,
			defaultStorageAccountsType: azureDiskDefaultType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(azureDisksConfig, tc.testEntity, tc.defaultStorageAccountsType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(azureDisksDataSourceRef, "entity_name", tc.entityName),
							resource.TestCheckResourceAttr(azureDisksDataSourceRef, "entity_type", tc.entityType),
							resource.TestCheckResourceAttr(azureDisksDataSourceRef, "current_storage_account_type", tc.currentStorageAccountsType),
							resource.TestCheckResourceAttr(azureDisksDataSourceRef, "new_storage_account_type", tc.newStorageAccountsType),
							resource.TestCheckResourceAttr(azureDisksDataSourceRef, "default_storage_account_type", tc.defaultStorageAccountsType),
						),
					},
				},
			})
		})
	}
}
