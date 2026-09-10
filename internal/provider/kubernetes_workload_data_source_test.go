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
	k8sWorkloadTestDataDir = "kubernetes_workload_data_source"

	k8sWorkloadSearchFile      = "search_success.json"
	k8sWorkloadResizeActionFile = "action_resize.json"
	k8sWorkloadScaleActionFile  = "action_scale.json"

	k8sWorkloadDataSourceRef = "data.turbonomic_kubernetes_workload.test"

	k8sWorkloadCluster   = "Kubernetes-Turbonomic"
	k8sWorkloadNamespace = "turbonomic"
	k8sWorkloadName      = "action-orchestrator"
	k8sWorkloadUUID      = "76084922964120"
)

// Terraform config helpers

const k8sWorkloadConfig = `
data "turbonomic_kubernetes_workload" "test" {
	cluster   = %q
	namespace = %q
	name      = %q
}
`

const k8sWorkloadConfigWithDefaults = `
data "turbonomic_kubernetes_workload" "test" {
	cluster          = %q
	namespace        = %q
	name             = %q
	default_replicas = %d
	default_containers = [
		{
			name                    = "action-orchestrator"
			default_cpu_request     = "150m"
			default_cpu_limit       = "500m"
			default_memory_request  = "256Mi"
			default_memory_limit    = "512Mi"
		}
	]
}
`

// TestKubernetesWorkloadResizeAction verifies the RESIZE action path.
func TestKubernetesWorkloadResizeAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sWorkloadTestDataDir, k8sWorkloadSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, k8sWorkloadTestDataDir, k8sWorkloadResizeActionFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sWorkloadConfig, k8sWorkloadCluster, k8sWorkloadNamespace, k8sWorkloadName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "entity_uuid", k8sWorkloadUUID),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "action_type", "RESIZE"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "action_state", "READY"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "action_mode", "MANUAL"),
					// current_replicas from action Target.Aspects
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "current_replicas", "1"),
					// containers — one container "action-orchestrator"
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.#", "1"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.name", "action-orchestrator"),
					// memory request: 393216 KB → 384 Mi; new: 720896 KB → 704 Mi
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.current_memory_request", "384Mi"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.new_memory_request", "704Mi"),
					// memory limit: 786432 KB → 768 Mi; new: 1048576 KB → 1024 Mi
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.current_memory_limit", "768Mi"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.new_memory_limit", "1024Mi"),
					// cpu request: 200 mCores → "200m"; new: 100 mCores → "100m"
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.current_cpu_request", "200m"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.new_cpu_request", "100m"),
				),
			},
		},
	})
}

// TestKubernetesWorkloadScaleAction verifies the SCALE action path (replica recommendation).
func TestKubernetesWorkloadScaleAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sWorkloadTestDataDir, k8sWorkloadSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, k8sWorkloadTestDataDir, k8sWorkloadScaleActionFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sWorkloadConfig, k8sWorkloadCluster, k8sWorkloadNamespace, k8sWorkloadName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "entity_uuid", k8sWorkloadUUID),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "action_type", "SCALE"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "action_state", "READY"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "action_mode", "MANUAL"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "current_replicas", "1"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "new_replicas", "3"),
					// No RESIZE action — containers list should be empty
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.#", "0"),
				),
			},
		},
	})
}

// TestKubernetesWorkloadNoAction verifies the fallback path when no actions are returned.
func TestKubernetesWorkloadNoAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sWorkloadTestDataDir, k8sWorkloadSearchFile),
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
	dsConfig := fmt.Sprintf(k8sWorkloadConfigWithDefaults, k8sWorkloadCluster, k8sWorkloadNamespace, k8sWorkloadName, 2)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "entity_uuid", k8sWorkloadUUID),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "action_state", ""),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "action_mode", ""),
					// new_replicas falls back to default_replicas
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "new_replicas", "2"),
					// containers: one entry from default_containers with default resource values
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.#", "1"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.name", "action-orchestrator"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.new_cpu_request", "150m"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.new_cpu_limit", "500m"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.new_memory_request", "256Mi"),
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "containers.0.new_memory_limit", "512Mi"),
				),
			},
		},
	})
}

// TestKubernetesWorkloadEntityNotFound verifies the warning path when the entity is not found.
func TestKubernetesWorkloadEntityNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sWorkloadConfigWithDefaults, k8sWorkloadCluster, k8sWorkloadNamespace, "nonexistent-workload", 5)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// entity not found → new_replicas falls back to default
					resource.TestCheckResourceAttr(k8sWorkloadDataSourceRef, "new_replicas", "5"),
				),
			},
		},
	})
}
