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
	googleComputeDiskConfig = `
	data "turbonomic_google_compute_disk" "test" {
		entity_name       			  = "%s"
		default_type                  = "%s"
		default_size                   = %d
		default_provisioned_iops        = %d
		default_provisioned_throughput  = %d
	}
	`

	entitySearchRespSuccess      = "google_compute_disk_search_success_response.json"
	entityActionRespSuccess      = "google_compute_disk_action_response.json"
	entityStatsRespSuccess       = "google_compute_disk_stats_response.json"
	entityName                   = "terraform-demo-instance-1"
	currentType                  = "pd-standard"
	newType                      = "pd-balanced"
	defaultype                   = "pd-standard"
	currentProvisionedIops       = 80000
	newProvisionedIops           = 86010
	defaultProvisionedIops       = 80000
	currentProvisionedThroughput = 4000
	newProvisionedThroughput     = 2400
	defaultProvisionedThroughput = 4000
	currentSize                  = 1000
	newSize                      = 2667
	defaultSize                  = 1000
)

// Tests Google Compute Disk data block logic by mocking valid turbo api response
func TestGoogleComputeDiskDataSource(t *testing.T) {
	mockServer := createLocalCloudVolumeServer(t,
		loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
		loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
		loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))

	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                                 string
		testEntity                           string
		expectedEntityName                   string
		expectedCurrentType                  string
		expectedNewType                      string
		expectedDefaultType                  string
		expectedCurrentProvisionedIops       int64
		expectedNewProvisionedIops           int64
		expectedDefaultProvisionedIops       int64
		expectedCurrentProvisionedThroughput int64
		expectedNewProvisionedThroughput     int64
		expectedDefaultProvisionedThroughput int64
		expectedCurrentSize                  int64
		expectedNewSize                      int64
		expectedDefaultSize                  int64
	}{
		{
			name:                                 "Valid VirtualVolume Recommendation",
			testEntity:                           entityName,
			expectedEntityName:                   entityName,
			expectedCurrentType:                  currentType,
			expectedNewType:                      newType,
			expectedDefaultType:                  defaultype,
			expectedCurrentProvisionedIops:       currentProvisionedIops,
			expectedNewProvisionedIops:           newProvisionedIops,
			expectedDefaultProvisionedIops:       defaultProvisionedIops,
			expectedCurrentProvisionedThroughput: currentProvisionedThroughput,
			expectedNewProvisionedThroughput:     newProvisionedThroughput,
			expectedDefaultProvisionedThroughput: defaultProvisionedThroughput,
			expectedCurrentSize:                  currentSize,
			expectedNewSize:                      newSize,
			expectedDefaultSize:                  defaultSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(googleComputeDiskConfig, tc.testEntity, tc.expectedDefaultType, tc.expectedDefaultSize, tc.expectedDefaultProvisionedIops, tc.expectedDefaultProvisionedThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "entity_type", "VirtualVolume"),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "current_type", tc.expectedCurrentType),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "new_type", tc.expectedNewType),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "default_type", tc.expectedDefaultType),

							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "current_provisioned_iops", fmt.Sprintf("%d", tc.expectedCurrentProvisionedIops)),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "new_provisioned_iops", fmt.Sprintf("%d", tc.expectedNewProvisionedIops)),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "default_provisioned_iops", fmt.Sprintf("%d", tc.expectedDefaultProvisionedIops)),

							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "current_provisioned_throughput", fmt.Sprintf("%d", tc.expectedCurrentProvisionedThroughput)),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "new_provisioned_throughput", fmt.Sprintf("%d", tc.expectedNewProvisionedThroughput)),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "default_provisioned_throughput", fmt.Sprintf("%d", tc.expectedDefaultProvisionedThroughput)),

							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "current_size", fmt.Sprintf("%d", tc.expectedCurrentSize)),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "new_size", fmt.Sprintf("%d", tc.expectedNewSize)),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "default_size", fmt.Sprintf("%d", tc.expectedDefaultSize)),
						),
					},
				},
			})
		})
	}
}

// Tests Google Compute Disk data block for an entity that does not exist
func TestGoogleComputeDiskDataSourceNewInstance(t *testing.T) {
	mockServer := createLocalServer(t, "[]", "[]", "[]", "[]")
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                                 string
		testEntity                           string
		expectedEntityName                   string
		expectedNewType                      string
		expectedDefaultType                  string
		expectedDefaultProvisionedIops       int64
		expectedDefaultProvisionedThroughput int64
		expectedDefaultSize                  int64
	}{
		{
			name:                                 "VirtualVolume does not exist",
			testEntity:                           entityName,
			expectedEntityName:                   entityName,
			expectedNewType:                      defaultype,
			expectedDefaultType:                  defaultype,
			expectedDefaultProvisionedIops:       defaultProvisionedIops,
			expectedDefaultProvisionedThroughput: defaultProvisionedThroughput,
			expectedDefaultSize:                  defaultSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(googleComputeDiskConfig, tc.testEntity, tc.expectedDefaultType, tc.expectedDefaultSize, tc.expectedDefaultProvisionedIops, tc.expectedDefaultProvisionedThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckNoResourceAttr("data.turbonomic_google_compute_disk.test", "entity_type"),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "new_type", tc.expectedNewType),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "default_type", tc.expectedDefaultType),
						),
					},
				},
			})
		})
	}
}

// Tests Google Compute Disk data block for an entity that has no actions
func TestGoogleComputeDiskDataSourceNoActions(t *testing.T) {
	mockServer := createLocalCloudVolumeServer(t,
		loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
		"[]",
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                                 string
		testEntity                           string
		expectedEntityName                   string
		expectedCurrentType                  string
		expectedNewType                      string
		expectedDefaultType                  string
		expectedDefaultProvisionedIops       int64
		expectedDefaultProvisionedThroughput int64
		expectedDefaultSize                  int64
	}{
		{
			name:                                 "Valid VirtualVolume has not actions",
			testEntity:                           entityName,
			expectedEntityName:                   entityName,
			expectedCurrentType:                  currentType,
			expectedNewType:                      currentType,
			expectedDefaultType:                  defaultype,
			expectedDefaultProvisionedIops:       defaultProvisionedIops,
			expectedDefaultProvisionedThroughput: defaultProvisionedThroughput,
			expectedDefaultSize:                  defaultSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(googleComputeDiskConfig, tc.testEntity, tc.expectedDefaultType, tc.expectedDefaultSize, tc.expectedDefaultProvisionedIops, tc.expectedDefaultProvisionedThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "entity_type", "VirtualVolume"),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "current_type", tc.expectedCurrentType),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "new_type", tc.expectedNewType),
							resource.TestCheckResourceAttr("data.turbonomic_google_compute_disk.test", "default_type", tc.expectedDefaultType),
						),
					},
				},
			})
		})
	}
}

// Tests Google Compute Disk data block with invalid default type
func TestGoogleComputeDiskDataSourceInvalidDefaultType(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
		loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	t.Run("Valid VirtualVolume with invalid default type provided", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig +
						`data "turbonomic_google_compute_disk" "test" {
							entity_name  = "testEntity"
							default_type  = "invalid-type"
						}`,
					ExpectError: regexp.MustCompile(`Attribute default_type value must be one of:`),
				},
			},
		})
	})
}
