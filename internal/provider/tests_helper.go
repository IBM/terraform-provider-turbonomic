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
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	config = `
	provider "turbonomic" {
		username = "administrator"
		password = "12345"
		hostname = "%s"
		skipverify = true
	}
	`

	testDataBaseDir                 = "testdata"
	cloudTestDataBaseDir            = "cloud_data_source"
	awsVMTestDataBaseDir            = "aws_instance_data_source"
	azureLinuxVMTestDataBaseDir     = "azurerm_linux_virtual_machine_data_source"
	azureWindowsVMTestDataBaseDir   = "azurerm_windows_virtual_machine_data_source"
	googleVMTestDataBaseDir         = "google_compute_instance_data_source"
	entityTagTestDataBaseDir        = "entity_tags"
	rdsTestDataBaseDir              = "rds_data_source"
	ebsTestDataBaseDir              = "ebs_data_source"
	azureManagedDiskTestDataBaseDir = "azure_mananged_disk_data_source"
	googleComputeDiskDataBaseDir    = "google_compute_disk_data_source"
	entityActionDir                 = "entity_action_data_source"
	azureMSSQLTestDataBaseDir       = "azurerm_mssql_database_data_source"

	vmEntityType = "VirtualMachine"

	entityTagsRespTestData = "entity_tags_success_response.json"
	entityTagRespTestData  = "entity_tag_success_response.json"
)

type Response struct {
	Message    string
	HttpStatus int
}

type MockRoute struct {
	Method       string
	Path         string
	ExpectedBody string
	ResponseBody string
	ResponseCode int
}

func mockTurboServer(t *testing.T, routes []MockRoute) *httptest.Server {
	// Set shorter timeouts to prevent hanging connections
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, route := range routes {
			if r.Method == route.Method && matchPath(route.Path, r.URL.Path) {
				if len(route.ExpectedBody) > 0 {
					body, _ := io.ReadAll(r.Body)
					defer r.Body.Close()
					if !bytes.Equal(bytes.TrimSpace(body), []byte(route.ExpectedBody)) {
						http.Error(w, fmt.Sprintf("unexpected body: expected: %q, got %q", route.ExpectedBody, string(body)), http.StatusBadRequest)
						return
					}
				}

				w.WriteHeader(route.ResponseCode)
				fmt.Fprint(w, route.ResponseBody)
				return
			}
		}
		http.NotFound(w, r)
	}))

	// Configure TLS with shorter timeouts
	server.Config.ReadHeaderTimeout = 1 * time.Second
	server.Config.WriteTimeout = 1 * time.Second
	server.Config.IdleTimeout = 1 * time.Second
	server.TLS = &tls.Config{
		InsecureSkipVerify: true,
	}
	server.StartTLS()

	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	return server
}

func matchPath(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := range patternParts {
		if strings.HasPrefix(patternParts[i], "{") && strings.HasSuffix(patternParts[i], "}") {
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}
	return true
}

func loadTestFile(t *testing.T, pathParts ...string) string {
	t.Helper()

	p := filepath.Join(append([]string{testDataBaseDir}, pathParts...)...)
	d, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("failed to read test file %q: %v", p, err)
	}

	return string(d)
}

func LoginAndTagRoutes(t *testing.T) []MockRoute {
	return []MockRoute{
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/login",
			ResponseCode: http.StatusOK,
			ResponseBody: `{"status":"ok"}`,
		},
		{
			Method:       http.MethodPost,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData),
			ResponseCode: http.StatusOK,
		},
		{
			Method:       http.MethodGet,
			Path:         "/api/v3/entities/{id}/tags",
			ResponseBody: loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
			ResponseCode: http.StatusOK,
		},
	}
}

func init() {
	os.Setenv("TF_ACC", "1")
}
