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
	aksClusterTestDataDir              = "azurerm_aks_cluster_data_source"
	aksClusterSearchFile               = "search_success.json"
	aksClusterParkingRunningFile       = "parking_running.json"
	aksClusterParkingWithScheduleFile  = "parking_running_with_schedule.json"
	aksClusterDataSourceRef            = "data.turbonomic_azurerm_aks_cluster.test"

	aksClusterName = "Kubernetes-aks-parking-private"
	aksClusterUUID = "76257897421745"
)

const aksClusterConfig = `
data "turbonomic_azurerm_aks_cluster" "test" {
	name = %q
}
`

const aksClusterConfigWithDefault = `
data "turbonomic_azurerm_aks_cluster" "test" {
	name          = %q
	default_state = %q
}
`

// TestAzurermAKSClusterRunning verifies the RUNNING state path (no schedule — new_state == current_state).
func TestAzurermAKSClusterRunning(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, aksClusterTestDataDir, aksClusterSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/parking/entities/{id}",
			ResponseBody: loadTestFile(t, aksClusterTestDataDir, aksClusterParkingRunningFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(aksClusterConfig, aksClusterName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "entity_uuid", aksClusterUUID),
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "current_state", "RUNNING"),
					// no schedule → new_state == current_state
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "new_state", "RUNNING"),
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "cloud_provider", "AZURE"),
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "cloud_service_name", "AZURE_KUBERNETES_SERVICE"),
				),
			},
		},
	})
}

// TestAzurermAKSClusterRunningWithSchedule verifies that an active smart parking schedule
// causes new_state to be "STOPPED" when current_state is "RUNNING".
func TestAzurermAKSClusterRunningWithSchedule(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, aksClusterTestDataDir, aksClusterSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/parking/entities/{id}",
			ResponseBody: loadTestFile(t, aksClusterTestDataDir, aksClusterParkingWithScheduleFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(aksClusterConfig, aksClusterName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "entity_uuid", aksClusterUUID),
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "current_state", "RUNNING"),
					// schedule active → new_state == "STOPPED"
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "new_state", "STOPPED"),
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "cloud_provider", "AZURE"),
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "cloud_service_name", "AZURE_KUBERNETES_SERVICE"),
				),
			},
		},
	})
}

// TestAzurermAKSClusterNotFound verifies the fallback path when the cluster is not found.
func TestAzurermAKSClusterNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(aksClusterConfigWithDefault, "nonexistent-cluster", "RUNNING")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// not found → new_state falls back to default_state
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "new_state", "RUNNING"),
					resource.TestCheckResourceAttr(aksClusterDataSourceRef, "current_state", ""),
				),
			},
		},
	})
}
