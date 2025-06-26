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
	entitySearchRespSuccess = "google_compute_disk_search_success_response.json"
	entityActionRespSuccess = "google_compute_disk_action_response.json"
	entityName              = "terraform-demo-instance-1"
	currentType             = "pd-standard"
	newType                 = "pd-balanced"
	defaultype              = "pd-standard"
)

// Tests Google Compute Disk data block logic by mocking valid turbo api response
func TestGoogleComputeDiskDataSource(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
		loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))

	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                string
		testEntity          string
		expectedEntityName  string
		expectedCurrentType string
		expectedNewType     string
		expectedDefaultType string
	}{
		{
			name:                "Valid VirtualVolume Recommendation",
			testEntity:          entityName,
			expectedEntityName:  entityName,
			expectedCurrentType: currentType,
			expectedNewType:     newType,
			expectedDefaultType: defaultype,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_google_compute_disk" "test" {
							entity_name  = "` + tc.testEntity + `"
							default_type  = "` + tc.expectedDefaultType + `"
						}`,
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

// Tests Google Compute Disk data block for an entity that does not exist
func TestGoogleComputeDiskDataSourceNewInstance(t *testing.T) {
	mockServer := createLocalServer(t, "[]", "[]", "[]", "[]")
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                string
		testEntity          string
		expectedEntityName  string
		expectedNewType     string
		expectedDefaultType string
	}{
		{
			name:                "VirtualVolume does not exist",
			testEntity:          entityName,
			expectedEntityName:  entityName,
			expectedNewType:     defaultype,
			expectedDefaultType: defaultype,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_google_compute_disk" "test" {
							entity_name  = "` + tc.testEntity + `"
							default_type  = "` + tc.expectedDefaultType + `"
						}`,
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
	mockServer := createLocalServer(t,
		loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
		"[]",
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                string
		testEntity          string
		expectedEntityName  string
		expectedCurrentType string
		expectedNewType     string
		expectedDefaultType string
	}{
		{
			name:                "Valid VirtualVolume has not actions",
			testEntity:          entityName,
			expectedEntityName:  entityName,
			expectedCurrentType: currentType,
			expectedNewType:     currentType,
			expectedDefaultType: defaultype,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_google_compute_disk" "test" {
							entity_name  = "` + tc.testEntity + `"
							default_type  = "` + tc.expectedDefaultType + `"
						}`,
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
