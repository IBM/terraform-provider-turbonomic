// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	rdsConfig = `
        data turbonomic_aws_db_instance test {
            entity_name            = "%s"
            default_instance_class = "%s"
            default_storage_type   = "%s"
        }
	`
	rdsDataSourceRef = "data.turbonomic_aws_db_instance.test"
)

// Tests `azurerm_managed_disk` data block
func TestAccRDSDataSource(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		testEntity                  string
		testEntityType              string
		entityName                  string
		entityType                  string
		currentComputeInstanceClass string
		newComputeInstanceClass     string
		defaultComputeClass         string
		currentStorageInstanceType  string
		newStorageInstanceType      string
		defaultStorageType          string
	}{
		{
			name:                        "Valid RDS Recommendation",
			testEntity:                  rdsName,
			testEntityType:              "DatabaseServer",
			entityName:                  rdsName,
			entityType:                  "DatabaseServer",
			currentComputeInstanceClass: rdsCurrComputeClass,
			newComputeInstanceClass:     rdsNewComputeClass,
			defaultComputeClass:         rdsDefaultComputeClass,
			currentStorageInstanceType:  rdsCurrStorageType,
			newStorageInstanceType:      rdsNewStorageType,
			defaultStorageType:          rdsDefaultStorageType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(rdsConfig, tc.testEntity, tc.defaultComputeClass, tc.defaultStorageType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(rdsDataSourceRef, "entity_name", tc.entityName),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "entity_type", tc.entityType),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "current_instance_class", tc.currentComputeInstanceClass),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "new_instance_class", tc.newComputeInstanceClass),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_instance_class", tc.defaultComputeClass),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "current_storage_type", tc.currentStorageInstanceType),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "new_storage_type", tc.newStorageInstanceType),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_storage_type", tc.defaultStorageType),
						),
					},
				},
			})
		})
	}
}
