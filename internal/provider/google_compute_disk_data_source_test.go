// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"net/http"
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

	// Config templates for vendor ID tests
	gcpVirtualVolumeConfigWithVendorId = `
	data "turbonomic_google_compute_disk" "test" {
		vendor_id    = "%s"
		default_type = "%s"
		default_size                   = %d
		default_provisioned_iops        = %d
		default_provisioned_throughput  = %d
	}
	`

	gcpVirtualVolumeConfigWithVendorIdAndName = `
	data "turbonomic_google_compute_disk" "test" {
		vendor_id    = "%s"
		default_type = "%s"
		entity_name  = "%s"
		default_size                   = %d
		default_provisioned_iops        = %d
		default_provisioned_throughput  = %d
	}
	`

	gcpVirtualVolumeConfigNoIdentifiers = `
	data "turbonomic_google_compute_disk" "test" {
		default_type = "%s"
	}
	`

	// Data source reference
	gcpVirtualVolumeDataSourceRef = "data.turbonomic_google_compute_disk.test"

	// Test vendor ID
	testVendorId   = "test-vendor-id"
	testEntityName = "test-volume"
)

// Tests Google Compute Disk data block logic by mocking valid turbo api response
func TestGoogleComputeDiskDataSource(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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
	mockServer := mockTurboServer(t, []MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: "[]",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: "[]",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/login",
			ResponseCode: http.StatusOK,
			ResponseBody: `{"status":"ok"}`,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "[]",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "[]",
			ResponseCode: http.StatusOK,
		},
	})

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
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: "[]",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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

func TestGoogleComputeDiskDataSourceDefaultThroughput(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default disk mbps read write atleast 0", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(googleComputeDiskConfig, entityName, defaultype, defaultSize, defaultSize, -1),
					ExpectError: regexp.MustCompile(`Attribute default_provisioned_throughput value must be at least 0`),
				},
			},
		})
	})
}

func TestGoogleComputeDiskDataSourceDefaultIops(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default disk mbps read write atleast 0", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(googleComputeDiskConfig, entityName, defaultype, defaultSize, -1, defaultProvisionedThroughput),
					ExpectError: regexp.MustCompile(`Attribute default_provisioned_iops value must be at least 0`),
				},
			},
		})
	})
}

func TestGoogleComputeDiskDataSourceDefaultSize(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default disk mbps read write atleast 0", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(googleComputeDiskConfig, entityName, defaultype, -1, defaultSize, defaultProvisionedThroughput),
					ExpectError: regexp.MustCompile(`Attribute default_size value must be at least 0`),
				},
			},
		})
	})
}

// Test for a valid entity using vendor_id
func TestGoogleComputeDiskDataSourceWithVendorId(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                                 string
		testVendorId                         string
		expectedVendorId                     string
		expectedDefaultType                  string
		expectedCurrentType                  string
		expectedNewType                      string
		expectedDefaultProvisionedIops       int
		expectedCurrentProvisionedIops       int
		expectedNewProvisionedIops           int
		expectedDefaultProvisionedThroughput int
		expectedCurrentProvisionedThroughput int
		expectedNewProvisionedThroughput     int
		expectedDefaultSize                  int
		expectedCurrentSize                  int
		expectedNewSize                      int
	}{
		{
			name:                                 "Valid VirtualVolume recommendation with vendor_id",
			testVendorId:                         testVendorId,
			expectedVendorId:                     testVendorId,
			expectedDefaultType:                  defaultype,
			expectedCurrentType:                  currentType,
			expectedNewType:                      newType,
			expectedDefaultProvisionedIops:       defaultProvisionedIops,
			expectedCurrentProvisionedIops:       currentProvisionedIops,
			expectedNewProvisionedIops:           newProvisionedIops,
			expectedDefaultProvisionedThroughput: defaultProvisionedThroughput,
			expectedCurrentProvisionedThroughput: currentProvisionedThroughput,
			expectedNewProvisionedThroughput:     newProvisionedThroughput,
			expectedDefaultSize:                  defaultSize,
			expectedCurrentSize:                  currentSize,
			expectedNewSize:                      newSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(gcpVirtualVolumeConfigWithVendorId, tc.testVendorId, tc.expectedDefaultType, tc.expectedDefaultSize, tc.expectedDefaultProvisionedIops, tc.expectedDefaultProvisionedThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "vendor_id", tc.expectedVendorId),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "entity_type", volumeEntityType),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_type", tc.expectedDefaultType),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "current_type", tc.expectedCurrentType),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_type", tc.expectedNewType),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_provisioned_iops", fmt.Sprintf("%d", tc.expectedDefaultProvisionedIops)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "current_provisioned_iops", fmt.Sprintf("%d", tc.expectedCurrentProvisionedIops)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_provisioned_iops", fmt.Sprintf("%d", tc.expectedNewProvisionedIops)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_provisioned_throughput", fmt.Sprintf("%d", tc.expectedDefaultProvisionedThroughput)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "current_provisioned_throughput", fmt.Sprintf("%d", tc.expectedCurrentProvisionedThroughput)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_provisioned_throughput", fmt.Sprintf("%d", tc.expectedNewProvisionedThroughput)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_size", fmt.Sprintf("%d", tc.expectedDefaultSize)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "current_size", fmt.Sprintf("%d", tc.expectedCurrentSize)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_size", fmt.Sprintf("%d", tc.expectedNewSize)),
						),
					},
				},
			})
		})
	}
}

// Test for a valid entity using vendor_id and entity_name
func TestGoogleComputeDiskDataSourceWithVendorIdAndName(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                                 string
		testVendorId                         string
		testEntity                           string
		expectedVendorId                     string
		expectedEntityName                   string
		expectedDefaultType                  string
		expectedCurrentType                  string
		expectedNewType                      string
		expectedDefaultProvisionedIops       int
		expectedCurrentProvisionedIops       int
		expectedNewProvisionedIops           int
		expectedDefaultProvisionedThroughput int
		expectedCurrentProvisionedThroughput int
		expectedNewProvisionedThroughput     int
		expectedDefaultSize                  int
		expectedCurrentSize                  int
		expectedNewSize                      int
	}{
		{
			name:                                 "Valid VirtualVolume recommendation with vendor_id and entity_name",
			testVendorId:                         testVendorId,
			testEntity:                           entityName,
			expectedVendorId:                     testVendorId,
			expectedEntityName:                   entityName,
			expectedDefaultType:                  defaultype,
			expectedCurrentType:                  currentType,
			expectedNewType:                      newType,
			expectedDefaultProvisionedIops:       defaultProvisionedIops,
			expectedCurrentProvisionedIops:       currentProvisionedIops,
			expectedNewProvisionedIops:           newProvisionedIops,
			expectedDefaultProvisionedThroughput: defaultProvisionedThroughput,
			expectedCurrentProvisionedThroughput: currentProvisionedThroughput,
			expectedNewProvisionedThroughput:     newProvisionedThroughput,
			expectedDefaultSize:                  defaultSize,
			expectedCurrentSize:                  currentSize,
			expectedNewSize:                      newSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(gcpVirtualVolumeConfigWithVendorIdAndName, tc.testVendorId, tc.expectedDefaultType, tc.testEntity, tc.expectedDefaultSize, tc.expectedDefaultProvisionedIops, tc.expectedDefaultProvisionedThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "vendor_id", tc.expectedVendorId),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "entity_type", volumeEntityType),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_type", tc.expectedDefaultType),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "current_type", tc.expectedCurrentType),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_type", tc.expectedNewType),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_provisioned_iops", fmt.Sprintf("%d", tc.expectedDefaultProvisionedIops)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "current_provisioned_iops", fmt.Sprintf("%d", tc.expectedCurrentProvisionedIops)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_provisioned_iops", fmt.Sprintf("%d", tc.expectedNewProvisionedIops)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_provisioned_throughput", fmt.Sprintf("%d", tc.expectedDefaultProvisionedThroughput)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "current_provisioned_throughput", fmt.Sprintf("%d", tc.expectedCurrentProvisionedThroughput)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_provisioned_throughput", fmt.Sprintf("%d", tc.expectedNewProvisionedThroughput)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_size", fmt.Sprintf("%d", tc.expectedDefaultSize)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "current_size", fmt.Sprintf("%d", tc.expectedCurrentSize)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_size", fmt.Sprintf("%d", tc.expectedNewSize)),
						),
					},
				},
			})
		})
	}
}

// Test for when neither entity_name nor vendor_id is provided
func TestGoogleComputeDiskDataSourceWithNoIdentifiers(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entitySearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, googleComputeDiskDataBaseDir, entityStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	t.Run("Error when neither entity_name nor vendor_id is provided", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(gcpVirtualVolumeConfigNoIdentifiers, defaultype),
					ExpectError: regexp.MustCompile(`At least one of these attributes must be configured`),
				},
			},
		})
	})

}

// Test for an invalid vendor_id
func TestGoogleComputeDiskDataSourceWithInvalidVendorId(t *testing.T) {
	mockServer := mockTurboServer(t, []MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: "[]",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: "",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/login",
			ResponseCode: http.StatusOK,
			ResponseBody: `{"status":"ok"}`,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "",
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: "",
			ResponseCode: http.StatusOK,
		},
	})
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                                 string
		testVendorId                         string
		expectedVendorId                     string
		expectedDefaultType                  string
		expectedCurrentType                  string
		expectedNewType                      string
		expectedDefaultProvisionedIops       int
		expectedCurrentProvisionedIops       int
		expectedNewProvisionedIops           int
		expectedDefaultProvisionedThroughput int
		expectedCurrentProvisionedThroughput int
		expectedNewProvisionedThroughput     int
		expectedDefaultSize                  int
		expectedCurrentSize                  int
		expectedNewSize                      int
	}{
		{
			name:                                 "Invalid vendor_id",
			testVendorId:                         testVendorId,
			expectedVendorId:                     testVendorId,
			expectedDefaultType:                  defaultype,
			expectedCurrentType:                  currentType,
			expectedNewType:                      defaultype,
			expectedDefaultProvisionedIops:       defaultProvisionedIops,
			expectedCurrentProvisionedIops:       currentProvisionedIops,
			expectedNewProvisionedIops:           defaultProvisionedIops,
			expectedDefaultProvisionedThroughput: defaultProvisionedThroughput,
			expectedCurrentProvisionedThroughput: currentProvisionedThroughput,
			expectedNewProvisionedThroughput:     defaultProvisionedThroughput,
			expectedDefaultSize:                  defaultSize,
			expectedCurrentSize:                  currentSize,
			expectedNewSize:                      defaultSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(gcpVirtualVolumeConfigWithVendorId, tc.testVendorId, tc.expectedDefaultType, tc.expectedDefaultSize, tc.expectedDefaultProvisionedIops, tc.expectedDefaultProvisionedThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "vendor_id", tc.expectedVendorId),
							resource.TestCheckNoResourceAttr(gcpVirtualVolumeDataSourceRef, "entity_type"),
							resource.TestCheckNoResourceAttr(gcpVirtualVolumeDataSourceRef, "current_type"),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_type", tc.expectedNewType),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_type", tc.expectedDefaultType),
							resource.TestCheckNoResourceAttr(gcpVirtualVolumeDataSourceRef, "current_provisioned_iops"),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_provisioned_iops", fmt.Sprintf("%d", tc.expectedDefaultProvisionedIops)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_provisioned_iops", fmt.Sprintf("%d", tc.expectedNewProvisionedIops)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_provisioned_throughput", fmt.Sprintf("%d", tc.expectedDefaultProvisionedThroughput)),
							resource.TestCheckNoResourceAttr(gcpVirtualVolumeDataSourceRef, "current_provisioned_throughput"),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_provisioned_throughput", fmt.Sprintf("%d", tc.expectedNewProvisionedThroughput)),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "default_size", fmt.Sprintf("%d", tc.expectedDefaultSize)),
							resource.TestCheckNoResourceAttr(gcpVirtualVolumeDataSourceRef, "current_size"),
							resource.TestCheckResourceAttr(gcpVirtualVolumeDataSourceRef, "new_size", fmt.Sprintf("%d", tc.expectedNewSize)),
						),
					},
				},
			})
		})
	}
}
