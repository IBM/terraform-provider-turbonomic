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

	// Config templates for vendor ID tests
	awsVirtualVolumeConfigWithVendorId = `
	data "turbonomic_aws_ebs_volume" "test" {
		vendor_id    = "%s"
		default_type = "%s"
		default_size      = %d
		default_iops      = %d
		default_throughput = %d
	}
	`

	awsVirtualVolumeConfigWithVendorIdAndName = `
	data "turbonomic_aws_ebs_volume" "test" {
		vendor_id    = "%s"
		default_type = "%s"
		entity_name  = "%s"
		default_size      = %d
		default_iops      = %d
		default_throughput = %d
	}
	`

	awsVirtualVolumeConfigNoIdentifiers = `
	data "turbonomic_aws_ebs_volume" "test" {
		default_type = "%s"
	}
	`
)

// Tests valid volume data source creation when turbo sends tier info in details
func TestVolumeWithTierRespDataSource(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
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

func TestVolumeDataSourceDefaultSize(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default size atleast 0", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(ebsVolConfig, ebsVolEntityName, ebsVolNewType, -1, ebsVolNewIops, ebsVolNewThroughput),
					ExpectError: regexp.MustCompile(`Attribute default_size value must be at least 0`),
				},
			},
		})
	})
}

func TestVolumeDataSourceDefaultThroughput(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default throughput atleast 0", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(ebsVolConfig, ebsVolEntityName, ebsVolNewType, ebsVolNewSize, ebsVolNewIops, -1),
					ExpectError: regexp.MustCompile(`Attribute default_throughput value must be at least 0`),
				},
			},
		})
	})
}

func TestVolumeDataSourceDefaultIops(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	t.Run("Default iops atleast 0", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(ebsVolConfig, ebsVolEntityName, ebsVolNewType, ebsVolNewSize, -1, ebsVolNewThroughput),
					ExpectError: regexp.MustCompile(`Attribute default_iops value must be at least 0`),
				},
			},
		})
	})
}

// Test for a valid entity using vendor_id
func TestAwsEbsVolumeDataSourceWithVendorId(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                      string
		testVendorId              string
		expectedVendorId          string
		expectedDefaultType       string
		expectedCurrentType       string
		expectedNewType           string
		expectedDefaultIops       int
		expectedCurrentIops       int
		expectedNewIops           int
		expectedDefaultThroughput int
		expectedCurrentThroughput int
		expectedNewThroughput     int
		expectedDefaultSize       int
		expectedCurrentSize       int
		expectedNewSize           int
	}{
		{
			name:                      "Valid VirtualVolume recommendation with vendor_id",
			testVendorId:              "vol-123456789",
			expectedVendorId:          "vol-123456789",
			expectedDefaultType:       ebsVolNewType,
			expectedCurrentType:       ebsVolCurrType,
			expectedNewType:           ebsVolNewType,
			expectedDefaultIops:       ebsVolNewIops,
			expectedCurrentIops:       ebsVolCurrIops,
			expectedNewIops:           ebsVolNewIops,
			expectedDefaultThroughput: ebsVolNewThroughput,
			expectedCurrentThroughput: ebsVolCurrThroughput,
			expectedNewThroughput:     ebsVolNewThroughput,
			expectedDefaultSize:       ebsVolNewSize,
			expectedCurrentSize:       ebsVolCurrSize,
			expectedNewSize:           ebsVolNewSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(awsVirtualVolumeConfigWithVendorId, tc.testVendorId, tc.expectedDefaultType, tc.expectedDefaultSize, tc.expectedDefaultIops, tc.expectedDefaultThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "vendor_id", tc.expectedVendorId),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_type", ebsVolEntityType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_type", tc.expectedDefaultType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_type", tc.expectedCurrentType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_type", tc.expectedNewType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_iops", fmt.Sprintf("%d", tc.expectedDefaultIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_iops", fmt.Sprintf("%d", tc.expectedCurrentIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_iops", fmt.Sprintf("%d", tc.expectedNewIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_throughput", fmt.Sprintf("%d", tc.expectedDefaultThroughput)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_throughput", fmt.Sprintf("%d", tc.expectedCurrentThroughput)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_throughput", fmt.Sprintf("%d", tc.expectedNewThroughput)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_size", fmt.Sprintf("%d", tc.expectedDefaultSize)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_size", fmt.Sprintf("%d", tc.expectedCurrentSize)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_size", fmt.Sprintf("%d", tc.expectedNewSize)),
						),
					},
				},
			})
		})
	}
}

// Test for a valid entity using vendor_id and entity_name
func TestAwsEbsVolumeDataSourceWithVendorIdAndName(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))

	for _, tc := range []struct {
		name                      string
		testVendorId              string
		testEntity                string
		expectedVendorId          string
		expectedEntityName        string
		expectedDefaultType       string
		expectedCurrentType       string
		expectedNewType           string
		expectedDefaultIops       int
		expectedCurrentIops       int
		expectedNewIops           int
		expectedDefaultThroughput int
		expectedCurrentThroughput int
		expectedNewThroughput     int
		expectedDefaultSize       int
		expectedCurrentSize       int
		expectedNewSize           int
	}{
		{
			name:                      "Valid VirtualVolume recommendation with vendor_id and entity_name",
			testVendorId:              "vol-123456789",
			testEntity:                "test-volume",
			expectedVendorId:          "vol-123456789",
			expectedEntityName:        "test-volume",
			expectedDefaultType:       ebsVolNewType,
			expectedCurrentType:       ebsVolCurrType,
			expectedNewType:           ebsVolNewType,
			expectedDefaultIops:       ebsVolNewIops,
			expectedCurrentIops:       ebsVolCurrIops,
			expectedNewIops:           ebsVolNewIops,
			expectedDefaultThroughput: ebsVolNewThroughput,
			expectedCurrentThroughput: ebsVolCurrThroughput,
			expectedNewThroughput:     ebsVolNewThroughput,
			expectedDefaultSize:       ebsVolNewSize,
			expectedCurrentSize:       ebsVolCurrSize,
			expectedNewSize:           ebsVolNewSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(awsVirtualVolumeConfigWithVendorIdAndName, tc.testVendorId, tc.expectedDefaultType, tc.testEntity, tc.expectedDefaultSize, tc.expectedDefaultIops, tc.expectedDefaultThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "vendor_id", tc.expectedVendorId),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "entity_type", ebsVolEntityType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_type", tc.expectedDefaultType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_type", tc.expectedCurrentType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_type", tc.expectedNewType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_iops", fmt.Sprintf("%d", tc.expectedDefaultIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_iops", fmt.Sprintf("%d", tc.expectedCurrentIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_iops", fmt.Sprintf("%d", tc.expectedNewIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_throughput", fmt.Sprintf("%d", tc.expectedDefaultThroughput)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_throughput", fmt.Sprintf("%d", tc.expectedCurrentThroughput)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_throughput", fmt.Sprintf("%d", tc.expectedNewThroughput)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_size", fmt.Sprintf("%d", tc.expectedDefaultSize)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "current_size", fmt.Sprintf("%d", tc.expectedCurrentSize)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_size", fmt.Sprintf("%d", tc.expectedNewSize)),
						),
					},
				},
			})
		})
	}
}

// Test for when neither entity_name nor vendor_id is provided
func TestAwsEbsVolumeDataSourceWithNoIdentifiers(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsSearchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsActTierResp),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, ebsTestDataBaseDir, validEbsStatsResp),
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
					Config:      providerConfig + fmt.Sprintf(awsVirtualVolumeConfigNoIdentifiers, ebsVolNewType),
					ExpectError: regexp.MustCompile(`At least one of these attributes must be configured`),
				},
			},
		})
	})

}

// Test for an invalid vendor_id
func TestAwsEbsVolumeDataSourceWithInvalidVendorId(t *testing.T) {
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
		name                      string
		testVendorId              string
		expectedVendorId          string
		expectedDefaultType       string
		expectedCurrentType       string
		expectedNewType           string
		expectedDefaultIops       int
		expectedCurrentIops       int
		expectedNewIops           int
		expectedDefaultThroughput int
		expectedCurrentThroughput int
		expectedNewThroughput     int
		expectedDefaultSize       int
		expectedCurrentSize       int
		expectedNewSize           int
	}{
		{
			name:                      "Invalid vendor_id",
			testVendorId:              "vol-123456789",
			expectedVendorId:          "vol-123456789",
			expectedDefaultType:       ebsVolNewType,
			expectedCurrentType:       ebsVolCurrType,
			expectedNewType:           ebsVolNewType,
			expectedDefaultIops:       ebsVolNewIops,
			expectedCurrentIops:       ebsVolCurrIops,
			expectedNewIops:           ebsVolNewIops,
			expectedDefaultThroughput: ebsVolNewThroughput,
			expectedCurrentThroughput: ebsVolCurrThroughput,
			expectedNewThroughput:     ebsVolNewThroughput,
			expectedDefaultSize:       ebsVolNewSize,
			expectedCurrentSize:       ebsVolCurrSize,
			expectedNewSize:           ebsVolNewSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(awsVirtualVolumeConfigWithVendorId, tc.testVendorId, tc.expectedDefaultType, tc.expectedDefaultSize, tc.expectedDefaultIops, tc.expectedDefaultThroughput),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "vendor_id", tc.expectedVendorId),
							resource.TestCheckNoResourceAttr(ebsVolDataSourceRef, "entity_type"),
							resource.TestCheckNoResourceAttr(ebsVolDataSourceRef, "current_type"),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_type", tc.expectedNewType),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_type", tc.expectedDefaultType),
							resource.TestCheckNoResourceAttr(ebsVolDataSourceRef, "current_iops"),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_iops", fmt.Sprintf("%d", tc.expectedDefaultIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_iops", fmt.Sprintf("%d", tc.expectedNewIops)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_throughput", fmt.Sprintf("%d", tc.expectedDefaultThroughput)),
							resource.TestCheckNoResourceAttr(ebsVolDataSourceRef, "current_throughput"),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_throughput", fmt.Sprintf("%d", tc.expectedNewThroughput)),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "default_size", fmt.Sprintf("%d", tc.expectedDefaultSize)),
							resource.TestCheckNoResourceAttr(ebsVolDataSourceRef, "current_size"),
							resource.TestCheckResourceAttr(ebsVolDataSourceRef, "new_size", fmt.Sprintf("%d", tc.expectedNewSize)),
						),
					},
				},
			})
		})
	}
}
