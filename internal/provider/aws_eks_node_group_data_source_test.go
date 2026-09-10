// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	eksNodeGroupTestDataDir         = "aws_eks_node_group_data_source"
	eksNodeGroupSearchFile          = "search_success.json"
	eksNodeGroupProvisionActionFile = "action_provision.json"
	eksNodeGroupDataSourceRef       = "data.turbonomic_aws_eks_node_group.test"

	eksNodeGroupCluster   = "my-eks-cluster"
	eksNodeGroupName      = "my-node-group"
	eksNodeGroupFirstUUID = "111000000000001"
)

const eksNodeGroupConfig = `
data "turbonomic_aws_eks_node_group" "test" {
	cluster_name    = %q
	node_group_name = %q
}
`

const eksNodeGroupConfigWithDefault = `
data "turbonomic_aws_eks_node_group" "test" {
	cluster_name        = %q
	node_group_name     = %q
	default_desired_size = %d
}
`

// TestAwsEKSNodeGroupProvisionAction verifies the PROVISION action path:
// 2 nodes found, 1 PROVISION action per node → new_node_count = 2 + 2 = 4.
func TestAwsEKSNodeGroupProvisionAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, eksNodeGroupTestDataDir, eksNodeGroupSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, eksNodeGroupTestDataDir, eksNodeGroupProvisionActionFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(eksNodeGroupConfig, eksNodeGroupCluster, eksNodeGroupName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "entity_uuid", eksNodeGroupFirstUUID),
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "action_type", "PROVISION"),
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "action_state", "READY"),
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "action_mode", "MANUAL"),
					// 2 VMs found
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "current_node_count", "2"),
					// 1 PROVISION per VM × 2 VMs = +2 delta → 2+2=4
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "new_node_count", "4"),
				),
			},
		},
	})
}

// TestAwsEKSNodeGroupNoAction verifies the fallback path (no PROVISION/SUSPEND actions).
func TestAwsEKSNodeGroupNoAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, eksNodeGroupTestDataDir, eksNodeGroupSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(eksNodeGroupConfig, eksNodeGroupCluster, eksNodeGroupName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "entity_uuid", eksNodeGroupFirstUUID),
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "action_state", ""),
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "action_mode", ""),
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "current_node_count", "2"),
					// no action → new_node_count falls back to current_node_count
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "new_node_count", "2"),
				),
			},
		},
	})
}

// TestAwsEKSNodeGroupEntityNotFound verifies the fallback path when no matching VMs are found.
func TestAwsEKSNodeGroupEntityNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(eksNodeGroupConfigWithDefault, eksNodeGroupCluster, "nonexistent-group", 5)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// entity not found → new_node_count falls back to default_desired_size
					resource.TestCheckResourceAttr(eksNodeGroupDataSourceRef, "new_node_count", "5"),
				),
			},
		},
	})
}
