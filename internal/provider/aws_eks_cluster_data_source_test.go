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
	eksClusterTestDataDir             = "aws_eks_cluster_data_source"
	eksClusterSearchFile              = "search_success.json"
	eksClusterParkingStoppedFile      = "parking_stopped.json"
	eksClusterParkingWithScheduleFile = "parking_stopped_with_schedule.json"
	eksClusterDataSourceRef           = "data.turbonomic_aws_eks_cluster.test"

	eksClusterName = "Kubernetes-my-eks-cluster"
	eksClusterUUID = "88112345678901"
)

const eksClusterConfig = `
data "turbonomic_aws_eks_cluster" "test" {
	name = %q
}
`

const eksClusterConfigWithDefault = `
data "turbonomic_aws_eks_cluster" "test" {
	name          = %q
	default_state = %q
}
`

// TestAwsEKSClusterStopped verifies the STOPPED state path (no schedule — new_state == current_state).
func TestAwsEKSClusterStopped(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, eksClusterTestDataDir, eksClusterSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/parking/entities/{id}",
			ResponseBody: loadTestFile(t, eksClusterTestDataDir, eksClusterParkingStoppedFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(eksClusterConfig, eksClusterName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "entity_uuid", eksClusterUUID),
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "current_state", "STOPPED"),
					// no schedule → new_state == current_state
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "new_state", "STOPPED"),
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "cloud_service_name", "AMAZON_ELASTIC_KUBERNETES_SERVICE"),
				),
			},
		},
	})
}

// TestAwsEKSClusterStoppedWithSchedule verifies that an active smart parking schedule
// causes new_state to be "RUNNING" when current_state is "STOPPED".
func TestAwsEKSClusterStoppedWithSchedule(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, eksClusterTestDataDir, eksClusterSearchFile),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/parking/entities/{id}",
			ResponseBody: loadTestFile(t, eksClusterTestDataDir, eksClusterParkingWithScheduleFile),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(eksClusterConfig, eksClusterName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "entity_uuid", eksClusterUUID),
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "current_state", "STOPPED"),
					// schedule active → new_state == "RUNNING" (unpark)
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "new_state", "RUNNING"),
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "cloud_service_name", "AMAZON_ELASTIC_KUBERNETES_SERVICE"),
				),
			},
		},
	})
}

// TestAwsEKSClusterNotFound verifies the fallback path when the cluster is not found.
func TestAwsEKSClusterNotFound(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, emptyActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))

	providerConfig := fmt.Sprintf(config, strings.TrimPrefix(mockServer.URL, "https://"))
	dsConfig := fmt.Sprintf(eksClusterConfigWithDefault, "nonexistent-cluster", "STOPPED")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + dsConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					// not found → new_state falls back to default_state
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "new_state", "STOPPED"),
					resource.TestCheckResourceAttr(eksClusterDataSourceRef, "current_state", ""),
				),
			},
		},
	})
}
