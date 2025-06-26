// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const (
	providerConfig = `provider "turbonomic" {
								hostname = "%s"
								username = "testuser"
								password = "password"
								skipverify = true
								}
								`
)

// test if the function returns valid tag value
func TestGetTagFunction_ValidTagValue(t *testing.T) {

	testConfig := `
				output "turbonomic_tag" {
				value = provider::turbonomic::get_tag().turbonomic_optimized_by
				}
				`

	t.Run("check tag value is valid", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow((tfversion.Version1_8_0)),
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + testConfig,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckOutput("turbonomic_tag", OptimizedByTagValue),
					),
				},
			},
		})
	})
}

// test if the function returns valid tag key
func TestGetTagFunction_ValidTagKey(t *testing.T) {

	testConfig := `
				output "turbonomic_tag" {
				value = keys(provider::turbonomic::get_tag())[0]
				}
				`

	t.Run("check tag key is valid", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow((tfversion.Version1_8_0)),
			},
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + testConfig,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckOutput("turbonomic_tag", OptimizedByTagName),
					),
				},
			},
		})
	})
}
