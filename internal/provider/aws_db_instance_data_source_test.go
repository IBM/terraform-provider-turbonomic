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
	rdsSearchRespSuccess  = "rds_search_success_response.json"
	rdsActionRespSuccess  = "rds_action_response.json"
	rdsSearchRespEmpty    = "empty_array_response.json"
	rdsActionRespEmpty    = "empty_array_response.json"
	rdsName               = "testRDS"
	rdsCurrSizeCompute    = "db.t4g.medium"
	rdsNewSizeCompute     = "db.t4g.small"
	rdsCurrSizeStorage    = "gp3"
	rdsNewSizestorage     = "standard"
	rdsDefaultSizeCompute = "db.t4g.micro"
	rdsDefaultSizeStorage = "gp2"
)

// Tests RDS data block logic by mocking valid turbo api response
func TestRDSDataSource(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, rdsTestDataBaseDir, rdsSearchRespSuccess),
		loadTestFile(t, rdsTestDataBaseDir, rdsActionRespSuccess),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
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
	}{
		{
			name:                       "Valid RDS Recommendation",
			testEntity:                 rdsName,
			expectedEntityName:         rdsName,
			expectedCurrentComputeTier: rdsCurrSizeCompute,
			expectedNewComputeTier:     rdsNewSizeCompute,
			expectedDefaultComputeTier: rdsDefaultSizeCompute,
			expectedCurrentStorageTier: rdsCurrSizeStorage,
			expectedNewStorageTier:     rdsNewSizestorage,
			expectedDefaultStorageTier: rdsDefaultSizeStorage,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_aws_db_instance" "test" {
							entity_name  = "` + tc.testEntity + `"
							default_instance_class  = "` + tc.expectedDefaultComputeTier + `"
							default_storage_type = "` + tc.expectedDefaultStorageTier + `"
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "entity_type", "DatabaseServer"),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "current_instance_class", tc.expectedCurrentComputeTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "current_storage_type", tc.expectedCurrentStorageTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "new_instance_class", tc.expectedNewComputeTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "new_storage_type", tc.expectedNewStorageTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "default_instance_class", tc.expectedDefaultComputeTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "default_storage_type", tc.expectedDefaultStorageTier),
						),
					},
				},
			})
		})
	}
}

// Tests RDS data block for an entity that does not exist (initial creation in Terraform)
func TestRDSDataSourcNewInstance(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, rdsSearchRespEmpty),
		loadTestFile(t, rdsActionRespEmpty),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
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
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_aws_db_instance" "test" {
							entity_name  = "` + tc.testEntity + `"
							default_instance_class  = "` + tc.expectedDefaultComputeTier + `"
							default_storage_type = "` + tc.expectedDefaultStorageTier + `"
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckNoResourceAttr("data.turbonomic_aws_db_instance.test", "entity_type"),
							resource.TestCheckNoResourceAttr("data.turbonomic_aws_db_instance.test", "current_instance_class"),
							resource.TestCheckNoResourceAttr("data.turbonomic_aws_db_instance.test", "current_storage_type"),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "new_instance_class", tc.expectedNewComputeTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "new_storage_type", tc.expectedNewStorageTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "default_instance_class", tc.expectedDefaultComputeTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "default_storage_type", tc.expectedDefaultStorageTier),
						),
					},
				},
			})
		})
	}
}

// Tests RDS data block for an entity that has no actions
func TestRDSDataSourcNoActions(t *testing.T) {
	mockServer := createLocalServer(t,
		loadTestFile(t, rdsTestDataBaseDir, rdsSearchRespSuccess),
		loadTestFile(t, rdsActionRespEmpty),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
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
	}{
		{
			name:                       "RDS has no actions",
			testEntity:                 rdsName,
			expectedEntityName:         rdsName,
			expectedCurrentComputeTier: rdsCurrSizeCompute,
			expectedNewComputeTier:     rdsCurrSizeCompute,
			expectedDefaultComputeTier: rdsDefaultSizeCompute,
			expectedCurrentStorageTier: rdsCurrSizeStorage,
			expectedNewStorageTier:     rdsCurrSizeStorage,
			expectedDefaultStorageTier: rdsDefaultSizeStorage,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_aws_db_instance" "test" {
							entity_name  = "` + tc.testEntity + `"
							default_instance_class  = "` + tc.expectedDefaultComputeTier + `"
							default_storage_type = "` + tc.expectedDefaultStorageTier + `"
						}`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "entity_type", "DatabaseServer"),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "current_instance_class", tc.expectedCurrentComputeTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "current_storage_type", tc.expectedCurrentStorageTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "new_instance_class", tc.expectedNewComputeTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "new_storage_type", tc.expectedNewStorageTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "default_instance_class", tc.expectedDefaultComputeTier),
							resource.TestCheckResourceAttr("data.turbonomic_aws_db_instance.test", "default_storage_type", tc.expectedDefaultStorageTier),
						),
					},
				},
			})
		})
	}
}
