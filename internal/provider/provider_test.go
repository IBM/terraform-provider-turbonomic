// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS-IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"turbonomic": providerserver.NewProtocol6WithError(New("test", "turbonomic")()),
}

const (
	resourceConfig = `data "turbonomic_cloud_entity_recommendation" "test" {
		entity_name  = "testEntity"
		entity_type  = "VirtualMachine"
		default_size = "testDefaultSize"
	}
	`
)

func TestProviderUsernamePassword(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	testConfig := `provider "turbonomic" {
								hostname = "%s"
								username = "testuser"
								password = "password"
								skipverify = true
								}
					`
	providerConfig := fmt.Sprintf(testConfig, strings.TrimPrefix(mockServer.URL, "https://"))

	t.Run("test1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + resourceConfig,
				},
			},
		})
	})
}

func TestProviderOAuth(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/oauth2/token",
			ResponseCode: http.StatusOK,
			ResponseBody: `{"status":"ok"}`,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	testConfig := `provider "turbonomic" {
								hostname = "%s"
								client_id = "12345"
								client_secret = "201918171615141312"
								role = "OBSERVER"
								skipverify = true
								}
					`
	providerConfig := fmt.Sprintf(testConfig, strings.TrimPrefix(mockServer.URL, "https://"))

	t.Run("test1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: providerConfig + resourceConfig,
				},
			},
		})
	})
}

func TestProviderNoHostname(t *testing.T) {

	testConfig := `provider "turbonomic" {
								username = "testuser"
								password = "password"
								role = "OBSERVER"
								skipverify = true
								}
					`

	t.Run("test1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testConfig + resourceConfig,
					ExpectError: regexp.MustCompile(`missing turbonomic api hostname`),
				},
			},
		})
	})
}

func TestProviderMultipleAuth(t *testing.T) {

	testConfig := `provider "turbonomic" {
								hostname = "%s"
								username = "testuser"
								client_id = "12345"
								skipverify = true
								}
					`

	t.Run("test1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testConfig + resourceConfig,
					ExpectError: regexp.MustCompile(`invalid attribute combination -> multiple authentication methods provided`),
				},
			},
		})
	})
}

func TestProviderMissingPassword(t *testing.T) {

	testConfig := `provider "turbonomic" {
								hostname = "%s"
								username = "testuser"
								skipverify = true
								}
					`

	t.Run("test1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testConfig + resourceConfig,
					ExpectError: regexp.MustCompile(`invalid attribute combination -> username/password`),
				},
			},
		})
	})
}

func TestProviderMissingClientSecret(t *testing.T) {

	testConfig := `provider "turbonomic" {
								hostname = "%s"
								client_id = "12345"
								skipverify = true
								}
					`

	t.Run("test1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testConfig + resourceConfig,
					ExpectError: regexp.MustCompile(`invalid attribute combination -> oAuth`),
				},
			},
		})
	})
}

func TestProviderUnknownRole(t *testing.T) {

	testConfig := `provider "turbonomic" {
						hostname = "%s"
						client_id = "12345"
						client_secret = "201918171615141312"
						role = "FakeRole"
						skipverify = true
						}
					`

	t.Run("test1", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testConfig + resourceConfig,
					ExpectError: regexp.MustCompile(`invalid attribute value match -> unknown role`),
				},
			},
		})
	})
}

func TestProviderTurboApiNotWorking(t *testing.T) {
	mockServer := mockTurboServer(t, append([]MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/search",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, searchRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/actions",
			ResponseBody: loadTestFile(t, cloudTestDataBaseDir, validVmActionRespTestData),
			ResponseCode: http.StatusOK,
		},
	}, LoginAndTagRoutes(t)...))
	defer mockServer.Close()

	testConfig := `provider "turbonomic" {
		hostname = "%s"
		username = "testuser"
		password = "password"
		skipverify = true
	}
	`
	providerConfig := fmt.Sprintf(testConfig, "invalid-hostname")
	dsName := "data.turbonomic_cloud_entity_recommendation.test"

	t.Run("tests no error when turbo api is not working", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + fmt.Sprintf(vmConfig, vmName, vmType, vmNewSize),
					ExpectError: nil,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(dsName, "entity_name", vmName),
						resource.TestCheckResourceAttr(dsName, "entity_type", vmType),
						resource.TestCheckNoResourceAttr(dsName, "current_instance_type"),
						resource.TestCheckResourceAttr(dsName, "new_instance_type", vmNewSize),
						resource.TestCheckResourceAttr(dsName, "default_size", vmNewSize),
					),
				},
			},
		})
	})
}
