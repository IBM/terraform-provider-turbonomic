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
	k8sNamespaceTestDataDir = "kubernetes_namespace_data_source"

	k8sNamespaceSearchFile      = "search_success.json"
	k8sNamespaceResizeActionFile = "action_resize.json"

	k8sNamespaceDataSourceRef = "data.turbonomic_kubernetes_namespace.test"

	k8sNamespaceCluster = "Kubernetes-Turbonomic"
	k8sNamespaceName    = "production"
	k8sNamespaceUUID    = "75878878702732"
)

const k8sNamespaceConfig = `
data "turbonomic_kubernetes_namespace" "test" {
	cluster = %q
	name    = %q
}
`

const k8sNamespaceConfigWithDefaults = `
data "turbonomic_kubernetes_namespace" "test" {
	cluster                    = %q
	name                       = %q
	default_cpu_limit_quota    = "500"
	default_cpu_request_quota  = "300"
	default_mem_limit_quota    = "1024"
	default_mem_request_quota  = "512"
}
`

// TestKubernetesNamespaceResizeAction verifies the RESIZE action path with multiple quota commodities.
func TestKubernetesNamespaceResizeAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sNamespaceTestDataDir, k8sNamespaceSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, k8sNamespaceTestDataDir, k8sNamespaceResizeActionFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sNamespaceConfig, k8sNamespaceCluster, k8sNamespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "entity_uuid", k8sNamespaceUUID),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "action_type", "RESIZE"),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "action_state", "READY"),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "action_mode", "MANUAL"),
					// CPU limit quota: current=1000 mCores, new=1500 mCores (from compoundActions)
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "current_cpu_limit_quota", "1000.0"),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_cpu_limit_quota", "1500.0"),
					// Memory limit quota: current=2097152 KB, new=4194304 KB (from compoundActions, valueUnits=KB)
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "current_mem_limit_quota", "2097152.0"),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_mem_limit_quota", "4194304.0"),
					// No CPU request or mem request quota in fixture → empty
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "current_cpu_request_quota", ""),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_cpu_request_quota", ""),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "current_mem_request_quota", ""),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_mem_request_quota", ""),
				),
			},
		},
	})
}

// TestKubernetesNamespaceNoAction verifies the fallback path when no actions are returned.
func TestKubernetesNamespaceNoAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sNamespaceTestDataDir, k8sNamespaceSearchFile),
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
	dsConfig := fmt.Sprintf(k8sNamespaceConfigWithDefaults, k8sNamespaceCluster, k8sNamespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "entity_uuid", k8sNamespaceUUID),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "action_type", ""),
					// new quota fields fall back to defaults
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_cpu_limit_quota", "500"),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_cpu_request_quota", "300"),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_mem_limit_quota", "1024"),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_mem_request_quota", "512"),
				),
			},
		},
	})
}

// TestKubernetesNamespaceEntityNotFound verifies the warning path when the entity is not found.
func TestKubernetesNamespaceEntityNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sNamespaceConfigWithDefaults, k8sNamespaceCluster, "nonexistent-namespace")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_cpu_limit_quota", "500"),
					resource.TestCheckResourceAttr(k8sNamespaceDataSourceRef, "new_mem_limit_quota", "1024"),
				),
			},
		},
	})
}
