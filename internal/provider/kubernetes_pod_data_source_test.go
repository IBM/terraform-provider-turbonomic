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
	k8sPodTestDataDir = "kubernetes_pod_data_source"

	k8sPodSearchFile    = "search_success.json"
	k8sPodMoveActionFile = "action_move.json"

	k8sPodDataSourceRef = "data.turbonomic_kubernetes_pod.test"

	k8sPodCluster   = "Kubernetes-Turbonomic"
	k8sPodNamespace = "production"
	k8sPodName      = "api-server-abc12"
	k8sPodUUID      = "84200000000001"
)

const k8sPodConfig = `
data "turbonomic_kubernetes_pod" "test" {
	cluster   = %q
	namespace = %q
	name      = %q
}
`

// TestKubernetesPodMoveAction verifies the MOVE action path.
func TestKubernetesPodMoveAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sPodTestDataDir, k8sPodSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, k8sPodTestDataDir, k8sPodMoveActionFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sPodConfig, k8sPodCluster, k8sPodNamespace, k8sPodName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "entity_uuid", k8sPodUUID),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "action_type", "MOVE"),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "action_state", "READY"),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "action_mode", "MANUAL"),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "current_node", "ip-10-0-1-42.ec2.internal"),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "current_node_uuid", "84300000000001"),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "new_node", "ip-10-0-2-55.ec2.internal"),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "new_node_uuid", "84300000000002"),
				),
			},
		},
	})
}

// TestKubernetesPodNoAction verifies the fallback path when no actions are returned.
func TestKubernetesPodNoAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sPodTestDataDir, k8sPodSearchFile),
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
	dsConfig := fmt.Sprintf(k8sPodConfig, k8sPodCluster, k8sPodNamespace, k8sPodName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "entity_uuid", k8sPodUUID),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "current_node", ""),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "new_node", ""),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "current_node_uuid", ""),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "new_node_uuid", ""),
				),
			},
		},
	})
}

// TestKubernetesPodEntityNotFound verifies the warning path when the entity is not found.
func TestKubernetesPodEntityNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sPodConfig, k8sPodCluster, k8sPodNamespace, "nonexistent-pod")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(k8sPodDataSourceRef, "entity_uuid", ""),
				),
			},
		},
	})
}
