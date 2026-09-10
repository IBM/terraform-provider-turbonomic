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
	k8sVolumeTestDataDir = "kubernetes_volume_data_source"

	k8sVolumeSearchFile      = "search_success.json"
	k8sVolumeScaleActionFile = "action_scale.json"

	k8sVolumeDataSourceRef = "data.turbonomic_kubernetes_volume.test"

	k8sVolumeCluster = "Kubernetes-VC14-OCP-GPU"
	k8sVolumeName    = "pvc-0513e11e-7798-47ae-abf5-e91fd8722beb"
	k8sVolumeUUID    = "76581453684439"
)

const k8sVolumeConfig = `
data "turbonomic_kubernetes_volume" "test" {
	cluster = %q
	name    = %q
}
`

const k8sVolumeConfigWithDefault = `
data "turbonomic_kubernetes_volume" "test" {
	cluster          = %q
	name             = %q
	default_size_gib = %d
}
`

// TestKubernetesVolumeScaleAction verifies the SCALE action path with MiB→GiB conversion.
func TestKubernetesVolumeScaleAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sVolumeTestDataDir, k8sVolumeSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, k8sVolumeTestDataDir, k8sVolumeScaleActionFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sVolumeConfig, k8sVolumeCluster, k8sVolumeName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Entity
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "entity_uuid", k8sVolumeUUID),
					// Action metadata
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "action_type", "SCALE"),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "action_state", "READY"),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "action_mode", "RECOMMEND"),
					// 10240 MiB → 10 GiB, 20480 MiB → 20 GiB
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "current_size_gib", "10"),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "new_size_gib", "20"),
				),
			},
		},
	})
}

// TestKubernetesVolumeNoAction verifies that no action returns all-empty action fields
// and falls back to current_size_gib / default_size_gib when no SCALE action exists.
func TestKubernetesVolumeNoAction(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, k8sVolumeTestDataDir, k8sVolumeSearchFile),
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
	dsConfig := fmt.Sprintf(k8sVolumeConfigWithDefault, k8sVolumeCluster, k8sVolumeName, 50)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "entity_uuid", k8sVolumeUUID),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "action_state", ""),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "action_mode", ""),
					// No action → current and new both fall back to default_size_gib = 50
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "current_size_gib", "50"),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "new_size_gib", "50"),
				),
			},
		},
	})
}

// TestKubernetesVolumeEntityNotFound verifies the warning path when the entity is not found.
func TestKubernetesVolumeEntityNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(k8sVolumeConfig, k8sVolumeCluster, "pvc-nonexistent")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "entity_uuid", ""),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "action_type", ""),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "current_size_gib", "0"),
					resource.TestCheckResourceAttr(k8sVolumeDataSourceRef, "new_size_gib", "0"),
				),
			},
		},
	})
}
