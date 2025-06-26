// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"strings"
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

	ebsVolEntityName = "testVM"
	ebsVolEntityType = "VirtualVolume"
	ebsVolCurrType   = "gp3"
	ebsVolNewType    = "standard"

	validEbsActTierResp        = "ebs_action_tier_valid_resp.json"
	validEbsActAmountResp      = "ebs_action_amount_valid_resp.json"
	validEbsActIopsResp        = "ebs_action_iops_valid_resp.json"
	validEbsActIopsAndThruResp = "ebs_action_iopsthru_valid_resp.json"

	ebsVolDataSourceRef = "data.turbonomic_" + EBS_VOL_DATASOURCE_NAME + ".test"
)

// Tests valid volume data source creation when turbo sends tier info in details
func TestVolumeWithTierRespDataSource(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
		loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                      string
		testEntity                string
		testDefaultType           string
		expectedEntityName        string
		expectedEntityType        string
		expectedCurrentVolumeType string
		expectedNewVolumeType     string
		expectedDefaultType       string
	}{
		{
			name:                      "Valid Volume Recommendation For Tier",
			testEntity:                ebsVolEntityName,
			testDefaultType:           ebsVolNewType,
			expectedEntityName:        ebsVolEntityName,
			expectedEntityType:        ebsVolEntityType,
			expectedCurrentVolumeType: ebsVolCurrType,
			expectedNewVolumeType:     ebsVolNewType,
			expectedDefaultType:       ebsVolNewType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(ebsVolConfig, tc.testEntity, tc.testDefaultType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_type", tc.expectedEntityType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_type", tc.expectedCurrentVolumeType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_type", tc.expectedNewVolumeType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_type", tc.expectedDefaultType),
						),
					},
				},
			})
		})
	}
}

// Tests default_size field in data block by mocking empty turbo search response
func TestVolumeWithEmptySearch(t *testing.T) {
	mockServer := createLocalServer(t,
		"[]",
		"",
		"",
		"")
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                      string
		testEntity                string
		expectedEntityName        string
		expectedEntityType        string
		expectedCurrentVolumeType string
		expectedNewVolumeType     string
		expectedDefaultType       string
	}{
		{
			name:                  "Checking Default on Empty Search",
			testEntity:            ebsVolEntityName,
			expectedEntityName:    ebsVolEntityName,
			expectedEntityType:    ebsVolEntityType,
			expectedNewVolumeType: ebsVolNewType,
			expectedDefaultType:   ebsVolNewType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(ebsVolConfig, tc.testEntity, tc.expectedDefaultType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckNoResourceAttr(ebsVolDataSourceRef, "entity_type"),
							resource.TestCheckNoResourceAttr(ebsVolDataSourceRef, "current_type"),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_type", tc.expectedNewVolumeType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_type", tc.expectedDefaultType),
						),
					},
				},
			})
		})
	}
}

// Tests default_size field in data block by mocking empty turbo action response
func TestVolumeWithEmptyAction(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
		loadTestFile(t, emptyActionRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                      string
		testEntity                string
		expectedEntityName        string
		expectedEntityType        string
		expectedCurrentVolumeType string
		expectedNewVolumeType     string
		expectedDefaultType       string
	}{
		{
			name:                      "Checking Default on Empty Action",
			testEntity:                ebsVolEntityName,
			expectedEntityName:        ebsVolEntityName,
			expectedEntityType:        ebsVolEntityType,
			expectedCurrentVolumeType: ebsVolCurrType,
			expectedNewVolumeType:     ebsVolNewType,
			expectedDefaultType:       ebsVolNewType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(ebsVolConfig, tc.testEntity, tc.expectedDefaultType),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_type", tc.expectedEntityType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_type", tc.expectedNewVolumeType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_type", tc.expectedDefaultType),
						),
					},
				},
			})
		})
	}
}
