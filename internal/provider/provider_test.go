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
	mockServer := createLocalServer(t, loadTestFile(t, searchRespFileLoc), loadTestFile(t, actionRespFileLoc))
	defer mockServer.Close()

	testConfig := `provider "turbonomic" {
								hostname = "%s"
								username = "testuser"
								password = "password"
								skipverify = true
								}
					`
	providerConfig := fmt.Sprintf(testConfig, strings.TrimLeft(mockServer.URL, "htps:/"))

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
	mockServer := createLocalServer(t, loadTestFile(t, searchRespFileLoc), loadTestFile(t, actionRespFileLoc))
	defer mockServer.Close()

	testConfig := `provider "turbonomic" {
								hostname = "%s"
								client_id = "12345"
								client_secret = "201918171615141312"
								role = "OBSERVER"
								skipverify = true
								}
					`
	providerConfig := fmt.Sprintf(testConfig, strings.TrimLeft(mockServer.URL, "htps:/"))

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
					ExpectError: regexp.MustCompile(`Missing Turbonomic API Hostname`),
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
					ExpectError: regexp.MustCompile(`Invalid Attribute Combination -> Multiple Authentication Methods Provided`),
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
					ExpectError: regexp.MustCompile(`Invalid Attribute Combination -> Username/Password`),
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
					ExpectError: regexp.MustCompile(`Invalid Attribute Combination -> oAuth`),
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
					ExpectError: regexp.MustCompile(`Invalid Attribute Value Match -> Unknown Role`),
				},
			},
		})
	})
}
