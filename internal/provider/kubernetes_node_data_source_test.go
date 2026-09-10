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
	k8sNodeTestDataDir = "kubernetes_node_data_source"

	k8sNodeSearchFile       = "search_success.json"
	k8sNodeSuspendActionFile = "action_suspend.json"

	k8sNodeDataSourceRef = "data.turbonomic_kubernetes_node.test"

	k8sNodeCluster = "Kubernetes-Turbonomic"
	k8sNodeName    = "ip-10-0-1-42.ec2.internal"
	k8sNodeUUID    = "85100000000001"
)

const k8sNodeConfig = `
data "turbonomic_kubernetes_node" "test" {
	cluster = %q
	name    = %q
}
`

// TestKubernetesNodeSuspendAction verifies the SUSPEND action path.
func TestKubernetesNodeSuspendAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sNodeTestDataDir, k8sNodeSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, k8sNodeTestDataDir, k8sNodeSuspendActionFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sNodeConfig, k8sNodeCluster, k8sNodeName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "entity_uuid", k8sNodeUUID),
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "action_type", "SUSPEND"),
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "action_state", "READY"),
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "action_mode", "MANUAL"),
				),
			},
		},
	})
}

// TestKubernetesNodeNoAction verifies the fallback path when no actions are returned.
func TestKubernetesNodeNoAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sNodeTestDataDir, k8sNodeSearchFile),
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
	dsConfig := fmt.Sprintf(k8sNodeConfig, k8sNodeCluster, k8sNodeName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "entity_uuid", k8sNodeUUID),
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "action_state", ""),
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "action_mode", ""),
				),
			},
		},
	})
}

// TestKubernetesNodeEntityNotFound verifies the warning path when the entity is not found.
func TestKubernetesNodeEntityNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sNodeConfig, k8sNodeCluster, "nonexistent-node")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(k8sNodeDataSourceRef, "entity_uuid", ""),
				),
			},
		},
	})
}
