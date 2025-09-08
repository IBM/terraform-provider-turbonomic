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
		entity_name       = "%s"
		default_type      = "%s"
		default_size      = %d
		default_iops      = %d
		default_throughput = %d
	}
	`

	ebsVolEntityName     = "testVM"
	ebsVolEntityType     = "VirtualVolume"
	ebsVolCurrType       = "gp3"
	ebsVolNewType        = "standard"
	ebsVolCurrSize       = 4
	ebsVolNewSize        = 4
	ebsVolCurrIops       = 3000
	ebsVolNewIops        = 160
	ebsVolCurrThroughput = 125
	ebsVolNewThroughput  = 128

	validEbsActTierResp        = "ebs_action_tier_valid_resp.json"
	validEbsStatsResp          = "ebs_stats_valid_resp.json"
	validEbsSearchRespTestData = "search_success.json"

	ebsVolDataSourceRef = "data.turbonomic_aws_ebs_volume.test"
)

// Tests valid volume data source creation when turbo sends tier info in details
func TestVolumeWithTierRespDataSource(t *testing.T) {
	mockServer := createLocalCloudVolumeServer(t,
		loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
		loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
		loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
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
		expectedDefaultSize       int64
		expectedDefaultIops       int64
		expectedDefaultThroughput int64
	}{
		{
			name:                      "Valid Volume Recommendation For Tier",
			testEntity:                ebsVolEntityName,
			expectedEntityName:        ebsVolEntityName,
			expectedEntityType:        ebsVolEntityType,
			expectedCurrentVolumeType: ebsVolCurrType,
			expectedNewVolumeType:     ebsVolNewType,
			expectedDefaultType:       ebsVolNewType,
			expectedDefaultSize:       ebsVolNewSize,
			expectedDefaultIops:       ebsVolNewIops,
			expectedDefaultThroughput: ebsVolNewThroughput,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(ebsVolConfig, tc.testEntity, tc.expectedDefaultType, tc.expectedDefaultSize, tc.expectedDefaultIops, tc.expectedDefaultThroughput),
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

// Tests valid volume data source creation when turbo sends tier,iops,throuput and size info in action and stat details
func TestVolumeWithValidActionRespDataSource(t *testing.T) {
	mockServer := createLocalCloudVolumeServer(t,
		loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
		loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
		loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
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
		expectedCurrentSize       int64
		expectedNewSize           int64
		expectedDefaultSize       int64
		expectedCurrentIops       int64
		expectedNewIops           int64
		expectedDefaultIops       int64
		expectedCurrentThroughput int64
		expectedNewThroughput     int64
		expectedDefaultThroughput int64
	}{
		{
			name:                      "Valid Volume Recommendation For Tier,IOPs,Throughput and Size",
			testEntity:                ebsVolEntityName,
			expectedEntityName:        ebsVolEntityName,
			expectedEntityType:        ebsVolEntityType,
			expectedCurrentVolumeType: ebsVolCurrType,
			expectedNewVolumeType:     ebsVolNewType,
			expectedDefaultType:       ebsVolNewType,
			expectedCurrentSize:       ebsVolCurrSize,
			expectedNewSize:           ebsVolNewSize,
			expectedDefaultSize:       ebsVolNewSize,
			expectedCurrentIops:       ebsVolCurrIops,
			expectedNewIops:           ebsVolNewIops,
			expectedDefaultIops:       ebsVolNewIops,
			expectedCurrentThroughput: ebsVolCurrThroughput,
			expectedNewThroughput:     ebsVolNewThroughput,
			expectedDefaultThroughput: ebsVolNewThroughput,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(ebsVolConfig, tc.testEntity, tc.expectedDefaultType, tc.expectedDefaultSize, tc.expectedDefaultIops, tc.expectedDefaultThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_type", tc.expectedEntityType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_type", tc.expectedCurrentVolumeType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_type", tc.expectedNewVolumeType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_type", tc.expectedDefaultType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_size", fmt.Sprintf("%d", tc.expectedCurrentSize)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_size", fmt.Sprintf("%d", tc.expectedNewSize)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_size", fmt.Sprintf("%d", tc.expectedDefaultSize)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_iops", fmt.Sprintf("%d", tc.expectedCurrentIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_iops", fmt.Sprintf("%d", tc.expectedNewIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_iops", fmt.Sprintf("%d", tc.expectedDefaultIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_throughput", fmt.Sprintf("%d", tc.expectedCurrentThroughput)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_throughput", fmt.Sprintf("%d", tc.expectedNewThroughput)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_throughput", fmt.Sprintf("%d", tc.expectedDefaultThroughput)),
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
						Config: providerConfig + fmt.Sprintf(ebsVolConfig, tc.testEntity, tc.expectedDefaultType, ebsVolNewSize, ebsVolNewIops, ebsVolNewThroughput),
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
	mockServer := createLocalCloudVolumeServer(t,
		loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
		loadTestFile(t, emptyActionRespTestData),
		loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
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
			expectedNewVolumeType:     ebsVolCurrType,
			expectedDefaultType:       ebsVolNewType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(ebsVolConfig, tc.testEntity, tc.expectedDefaultType, ebsVolNewSize, ebsVolNewIops, ebsVolNewThroughput),
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
