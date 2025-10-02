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
	rdsConfig = `
	data "turbonomic_aws_db_instance" "test" {
		entity_name              = "%s"
		default_instance_class   = "%s"
		default_storage_type     = "%s"
		default_iops             = %d
		default_allocated_storage = %d
	}
	`

	rdsSearchRespSuccess  = "rds_search_success_response.json"
	rdsActionRespSuccess  = "rds_action_response.json"
	rdsSearchRespEmpty    = "empty_array_response.json"
	rdsActionRespEmpty    = "empty_array_response.json"
	rdsStatsRespSuccess   = "rds_stats_valid_resp.json"
	rdsStatsRespEmpty     = "empty_array_response.json"
	rdsName               = "testRDS"
	rdsCurrSizeCompute    = "db.t4g.medium"
	rdsNewSizeCompute     = "db.t4g.small"
	rdsCurrSizeStorage    = "gp3"
	rdsNewSizestorage     = "standard"
	rdsDefaultSizeCompute = "db.t4g.micro"
	rdsDefaultSizeStorage = "gp2"
	rdsDefaultIops        = 1000
	rdsDefaultStorage     = 20

	rdsDataSourceRef = "data.turbonomic_aws_db_instance.test"
)

// Tests RDS data block logic by mocking valid turbo api response
func TestRDSDataSource(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, rdsTestDataBaseDir, rdsSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, rdsTestDataBaseDir, rdsActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, rdsTestDataBaseDir, rdsStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedEntityType         string
		expectedCurrentComputeTier string
		expectedNewComputeTier     string
		expectedDefaultComputeTier string
		expectedCurrentStorageTier string
		expectedNewStorageTier     string
		expectedDefaultStorageTier string
		expectedDefaultIops        int64
		expectedDefaultStorage     int64
	}{
		{
			name:                       "Valid RDS Recommendation",
			testEntity:                 rdsName,
			expectedEntityName:         rdsName,
			expectedEntityType:         "DatabaseServer",
			expectedCurrentComputeTier: rdsCurrSizeCompute,
			expectedNewComputeTier:     rdsNewSizeCompute,
			expectedDefaultComputeTier: rdsDefaultSizeCompute,
			expectedCurrentStorageTier: rdsCurrSizeStorage,
			expectedNewStorageTier:     rdsNewSizestorage,
			expectedDefaultStorageTier: rdsDefaultSizeStorage,
			expectedDefaultIops:        rdsDefaultIops,
			expectedDefaultStorage:     rdsDefaultStorage,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(rdsConfig, tc.testEntity, tc.expectedDefaultComputeTier, tc.expectedDefaultStorageTier, tc.expectedDefaultIops, tc.expectedDefaultStorage),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(rdsDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "entity_type", tc.expectedEntityType),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "current_instance_class", tc.expectedCurrentComputeTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "current_storage_type", tc.expectedCurrentStorageTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "new_instance_class", tc.expectedNewComputeTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "new_storage_type", tc.expectedNewStorageTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_instance_class", tc.expectedDefaultComputeTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_storage_type", tc.expectedDefaultStorageTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_iops", fmt.Sprintf("%d", tc.expectedDefaultIops)),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_allocated_storage", fmt.Sprintf("%d", tc.expectedDefaultStorage)),
						),
					},
				},
			})
		})
	}
}

// Tests RDS data block for an entity that does not exist (initial creation in Terraform)
func TestRDSDataSourcNewInstance(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, rdsSearchRespEmpty),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, rdsActionRespEmpty),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, rdsStatsRespEmpty),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedCurrentComputeTier string
		expectedNewComputeTier     string
		expectedDefaultComputeTier string
		expectedCurrentStorageTier string
		expectedNewStorageTier     string
		expectedDefaultStorageTier string
		expectedDefaultIops        int64
		expectedDefaultStorage     int64
	}{
		{
			name:                       "Non existing RDS instance",
			testEntity:                 rdsName,
			expectedEntityName:         rdsName,
			expectedCurrentComputeTier: rdsDefaultSizeCompute,
			expectedNewComputeTier:     rdsDefaultSizeCompute,
			expectedDefaultComputeTier: rdsDefaultSizeCompute,
			expectedCurrentStorageTier: rdsDefaultSizeStorage,
			expectedNewStorageTier:     rdsDefaultSizeStorage,
			expectedDefaultStorageTier: rdsDefaultSizeStorage,
			expectedDefaultIops:        rdsDefaultIops,
			expectedDefaultStorage:     rdsDefaultStorage,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(rdsConfig, tc.testEntity, tc.expectedDefaultComputeTier, tc.expectedDefaultStorageTier, tc.expectedDefaultIops, tc.expectedDefaultStorage),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(rdsDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckNoResourceAttr(rdsDataSourceRef, "entity_type"),
							resource.TestCheckNoResourceAttr(rdsDataSourceRef, "current_instance_class"),
							resource.TestCheckNoResourceAttr(rdsDataSourceRef, "current_storage_type"),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "new_instance_class", tc.expectedNewComputeTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "new_storage_type", tc.expectedNewStorageTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_instance_class", tc.expectedDefaultComputeTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_storage_type", tc.expectedDefaultStorageTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_iops", fmt.Sprintf("%d", tc.expectedDefaultIops)),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_allocated_storage", fmt.Sprintf("%d", tc.expectedDefaultStorage)),
						),
					},
				},
			})
		})
	}
}

// Tests RDS data block for an entity that has no actions
func TestRDSDataSourcNoActions(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, rdsTestDataBaseDir, rdsSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, rdsActionRespEmpty),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/stats/{id}",
			ResponseBody: loadTestFile(t, rdsTestDataBaseDir, rdsStatsRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                       string
		testEntity                 string
		expectedEntityName         string
		expectedEntityType         string
		expectedCurrentComputeTier string
		expectedNewComputeTier     string
		expectedDefaultComputeTier string
		expectedCurrentStorageTier string
		expectedNewStorageTier     string
		expectedDefaultStorageTier string
		expectedDefaultIops        int64
		expectedDefaultStorage     int64
	}{
		{
			name:                       "RDS has no actions",
			testEntity:                 rdsName,
			expectedEntityName:         rdsName,
			expectedEntityType:         "DatabaseServer",
			expectedCurrentComputeTier: rdsCurrSizeCompute,
			expectedNewComputeTier:     rdsCurrSizeCompute,
			expectedDefaultComputeTier: rdsDefaultSizeCompute,
			expectedCurrentStorageTier: rdsCurrSizeStorage,
			expectedNewStorageTier:     rdsCurrSizeStorage,
			expectedDefaultStorageTier: rdsDefaultSizeStorage,
			expectedDefaultIops:        rdsDefaultIops,
			expectedDefaultStorage:     rdsDefaultStorage,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig + fmt.Sprintf(rdsConfig, tc.testEntity, tc.expectedDefaultComputeTier, tc.expectedDefaultStorageTier, tc.expectedDefaultIops, tc.expectedDefaultStorage),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(rdsDataSourceRef, "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "entity_type", tc.expectedEntityType),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "current_instance_class", tc.expectedCurrentComputeTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "current_storage_type", tc.expectedCurrentStorageTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "new_instance_class", tc.expectedNewComputeTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "new_storage_type", tc.expectedNewStorageTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_instance_class", tc.expectedDefaultComputeTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_storage_type", tc.expectedDefaultStorageTier),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_iops", fmt.Sprintf("%d", tc.expectedDefaultIops)),
							resource.TestCheckResourceAttr(rdsDataSourceRef, "default_allocated_storage", fmt.Sprintf("%d", tc.expectedDefaultStorage)),
						),
					},
				},
			})
		})
	}
}

func TestRDSDataSourceDefaultLengthValidationInstanceClass(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, rdsTestDataBaseDir, rdsSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, rdsTestDataBaseDir, rdsActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))
	t.Run("Default instance class atleast 1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig +
						`data "turbonomic_aws_db_instance" "test" {
							entity_name  = "` + rdsName + `"
							default_instance_class  = ""
							default_storage_type = "` + rdsDefaultSizeStorage + `"
						}`,
					ExpectError: regexp.MustCompile(`Attribute default_instance_class string length must be at least 1`),
				},
			},
		})
	})
}

func TestRDSDataSourceDefaultLengthValidationStorageType(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, rdsTestDataBaseDir, rdsSearchRespSuccess),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, rdsTestDataBaseDir, rdsActionRespSuccess),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))
	t.Run("Default storage type atleast 1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig +
						`data "turbonomic_aws_db_instance" "test" {
							entity_name  = "` + rdsName + `"
							default_instance_class  = "` + rdsDefaultSizeCompute + `"
							default_storage_type = ""
						}`,
					ExpectError: regexp.MustCompile(`Attribute default_storage_type string length must be at least 1`),
				},
			},
		})
	})
}
