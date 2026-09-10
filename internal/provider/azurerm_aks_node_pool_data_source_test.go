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
	aksNodePoolTestDataDir       = "azurerm_aks_node_pool_data_source"
	aksNodePoolSearchFile        = "search_success.json"
	aksNodePoolSuspendActionFile = "action_suspend.json"
	aksNodePoolDataSourceRef     = "data.turbonomic_azurerm_aks_node_pool.test"

	aksNodePoolCluster   = "my-aks-cluster"
	aksNodePoolName      = "agentpool"
	aksNodePoolFirstUUID = "222000000000001"
)

const aksNodePoolConfig = `
data "turbonomic_azurerm_aks_node_pool" "test" {
	cluster_name = %q
	name         = %q
}
`

const aksNodePoolConfigWithDefault = `
data "turbonomic_azurerm_aks_node_pool" "test" {
	cluster_name       = %q
	name               = %q
	default_node_count = %d
}
`

// TestAzurermAKSNodePoolSuspendAction verifies the SUSPEND action path:
// 3 nodes found, 1 SUSPEND action per node → new_node_count = 3 - 3 = 0.
func TestAzurermAKSNodePoolSuspendAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, aksNodePoolTestDataDir, aksNodePoolSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, aksNodePoolTestDataDir, aksNodePoolSuspendActionFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(aksNodePoolConfig, aksNodePoolCluster, aksNodePoolName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "entity_uuid", aksNodePoolFirstUUID),
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "action_type", "SUSPEND"),
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "action_state", "READY"),
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "action_mode", "MANUAL"),
					// 3 VMs found
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "current_node_count", "3"),
					// 1 SUSPEND per VM × 3 VMs = -3 delta → 3-3=0
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "new_node_count", "0"),
				),
			},
		},
	})
}

// TestAzurermAKSNodePoolNoAction verifies the fallback path (no actions).
func TestAzurermAKSNodePoolNoAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, aksNodePoolTestDataDir, aksNodePoolSearchFile),
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
	dsConfig := fmt.Sprintf(aksNodePoolConfig, aksNodePoolCluster, aksNodePoolName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "entity_uuid", aksNodePoolFirstUUID),
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "current_node_count", "3"),
					// no action → new_node_count falls back to current_node_count
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "new_node_count", "3"),
				),
			},
		},
	})
}

// TestAzurermAKSNodePoolEntityNotFound verifies the fallback path when no matching VMs are found.
func TestAzurermAKSNodePoolEntityNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(aksNodePoolConfigWithDefault, aksNodePoolCluster, "nonexistent-pool", 3)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aksNodePoolDataSourceRef, "new_node_count", "3"),
				),
			},
		},
	})
}
