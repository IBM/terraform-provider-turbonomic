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
	"strings"
	"testing"
)

func createLocalServer(t *testing.T, searchResp, actionResp string) *httptest.Server {
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
		case r.Method == http.MethodPost && r.URL.Path == "/oauth2/token":
			body, _ := io.ReadAll(r.Body)
			defer r.Body.Close()
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"result":"POST success", "received": "%s"}`, string(body))
		default:
			http.NotFound(w, r)
		}
	}))
}

func loadTestFile(t *testing.T, filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}
	return string(data)
}

func init() {
	os.Setenv("TF_ACC", "1")
}
