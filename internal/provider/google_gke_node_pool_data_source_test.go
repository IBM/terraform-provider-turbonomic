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
	gkeNodePoolTestDataDir   = "google_gke_node_pool_data_source"
	gkeNodePoolSearchFile    = "search_success.json"
	gkeNodePoolDataSourceRef = "data.turbonomic_google_gke_node_pool.test"

	gkeNodePoolCluster   = "my-gke-cluster"
	gkeNodePoolName      = "default-pool"
	gkeNodePoolFirstUUID = "333000000000001"
)

const gkeNodePoolConfig = `
data "turbonomic_google_gke_node_pool" "test" {
	cluster_name  = %q
	node_pool_name = %q
}
`

const gkeNodePoolConfigWithDefault = `
data "turbonomic_google_gke_node_pool" "test" {
	cluster_name      = %q
	node_pool_name    = %q
	default_node_count = %d
}
`

// TestGoogleGKENodePoolNoAction verifies the no-action path:
// 2 VMs found, no PROVISION/SUSPEND → new_node_count falls back to current (2).
func TestGoogleGKENodePoolNoAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, gkeNodePoolTestDataDir, gkeNodePoolSearchFile),
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
	dsConfig := fmt.Sprintf(gkeNodePoolConfig, gkeNodePoolCluster, gkeNodePoolName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(gkeNodePoolDataSourceRef, "entity_uuid", gkeNodePoolFirstUUID),
					resource.TestCheckResourceAttr(gkeNodePoolDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(gkeNodePoolDataSourceRef, "action_state", ""),
					resource.TestCheckResourceAttr(gkeNodePoolDataSourceRef, "action_mode", ""),
					resource.TestCheckResourceAttr(gkeNodePoolDataSourceRef, "current_node_count", "2"),
					// no action → falls back to current
					resource.TestCheckResourceAttr(gkeNodePoolDataSourceRef, "new_node_count", "2"),
				),
			},
		},
	})
}

// TestGoogleGKENodePoolEntityNotFound verifies the fallback path when no matching VMs are found.
func TestGoogleGKENodePoolEntityNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(gkeNodePoolConfigWithDefault, gkeNodePoolCluster, "nonexistent-pool", 4)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(gkeNodePoolDataSourceRef, "new_node_count", "4"),
				),
			},
		},
	})
}
