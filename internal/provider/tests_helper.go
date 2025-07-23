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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	entityTagTestDataBaseDir        = "entity_tags"
	rdsTestDataBaseDir              = "rds_data_source"
	ebsTestDataBaseDir              = "ebs_data_source"
	azureManagedDiskTestDataBaseDir = "azure_mananged_disk_data_source"
	googleComputeDiskDataBaseDir    = "google_compute_disk_data_source"
)

type Response struct {
	Message    string
	HttpStatus int
}

func createLocalServer(t *testing.T, searchResp, actionResp, entityTagsResp, entityTagResp string) *httptest.Server {
	return createLocalServerWithResponse(t, searchResp, actionResp, entityTagsResp, Response{
		Message:    entityTagResp,
		HttpStatus: http.StatusOK})
}

// creates a mock turbo client which responds with provided search and action result
func createLocalServerWithResponse(t *testing.T, searchResp, actionResp, entityTagsResp string, entityTagResp Response) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/login":
			body, _ := io.ReadAll(r.Body)
			defer r.Body.Close()
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"result":"POST success", "received": "%s"}`, string(body))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/search":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, searchResp)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v3/entities/") &&
			strings.HasSuffix(r.URL.Path, "/actions"):
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, actionResp)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v3/entities/") && strings.HasSuffix(r.URL.Path, "/tags"):
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, entityTagsResp)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v3/entities/") && strings.HasSuffix(r.URL.Path, "/tags"):
			w.WriteHeader(entityTagResp.HttpStatus)
			fmt.Fprint(w, entityTagResp.Message)
		case r.Method == http.MethodPost && r.URL.Path == "/oauth2/token":
			body, _ := io.ReadAll(r.Body)
			defer r.Body.Close()
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"result":"POST success", "received": "%s"}`, string(body))
		default:
			// Endpoint not defined
			http.NotFound(w, r)
			t.FailNow()
		}
	}))
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

func init() {
	os.Setenv("TF_ACC", "1")
}
