// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	ebsVolConfig = `
	data "turbonomic_aws_ebs_volume" "test" {
		entity_name  = "%s"
		default_type = "%s"
	}
	`
	ebsVolDataSourceRef = "data.turbonomic_aws_ebs_volume.test"
)

// Tests `turbonomic_aws_ebs_volume` data block
func TestAccVolumeDataSource(t *testing.T) {
	for _, tc := range []struct {
		name              string
		testEntity        string
		testEntityType    string
		entityName        string
		entityType        string
		currentVolumeType string
		newVolumeType     string
		defaultType       string
	}{
		{
			name:              "Valid Volume Recommendation",
			testEntity:        ebsVolName,
			testEntityType:    "VirtualVolume",
			entityName:        ebsVolName,
			entityType:        "VirtualVolume",
			currentVolumeType: ebsVolCurrType,
			newVolumeType:     ebsVolNewType,
			defaultType:       ebsVolNewType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(ebsVolConfig, tc.testEntity, tc.defaultType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_name", tc.entityName),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_type", tc.entityType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_type", tc.currentVolumeType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_type", tc.newVolumeType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_type", tc.defaultType),
						),
					},
				},
			})
		})
	}
}
