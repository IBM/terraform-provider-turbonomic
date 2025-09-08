// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Test turbonomic_azurerm_mssql_database
func TestAccAzureMSSQLDatabaseDataSource(t *testing.T) {
	for _, tc := range []struct {
		name           string
		testEntity     string
		entityName     string
		entityType     string
		currentSkuName string
		newSkuName     string
		defaultSkuName string
	}{
		{
			name:           "Valid Azure Mssql database recommendation",
			testEntity:     azureMSSQLDatabaseName,
			entityName:     azureMSSQLDatabaseName,
			entityType:     "Database",
			currentSkuName: azureMSSQLDatabaseCurrentSkuName,
			newSkuName:     azureMSSQLDatabaseNewSkuName,
			defaultSkuName: azureMSSQLDatabaseDefaultSkuName,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_azurerm_mssql_database" "test" {
							entity_name  = "` + tc.testEntity + `"
							default_sku_name = "` + tc.defaultSkuName + `"
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_mssql_database.test", "entity_name", tc.entityName),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_mssql_database.test", "entity_type", tc.entityType),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_mssql_database.test", "current_sku_name", tc.currentSkuName),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_mssql_database.test", "new_sku_name", tc.newSkuName),
							resource.TestCheckResourceAttr("data.turbonomic_azurerm_mssql_database.test", "default_sku_name", tc.defaultSkuName),
						),
					},
				},
			})
		})
	}
}
