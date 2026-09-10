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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// policyHTTPPost marshals body to JSON, POSTs to baseURL+path, and returns the response bytes.
// Accepts either HTTP 200 or HTTP 201 as success.
func policyHTTPPost(ctx context.Context, client *http.Client, baseURL, path string, body interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("could not create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s failed: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("POST %s returned status %d: %s", path, resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// policyHTTPGet GETs baseURL+path and returns (responseBytes, statusCode, error).
// The caller is responsible for checking statusCode (e.g. 404 → remove from state).
// Does NOT return an error on 404.
func policyHTTPGet(ctx context.Context, client *http.Client, baseURL, path string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("could not create GET request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("GET %s failed: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return nil, resp.StatusCode, fmt.Errorf("GET %s returned status %d: %s", path, resp.StatusCode, string(respBody))
	}
	return respBody, resp.StatusCode, nil
}

// policyHTTPPut marshals body to JSON, PUTs to baseURL+path, and returns the response bytes.
// Accepts HTTP 200 as success.
func policyHTTPPut(ctx context.Context, client *http.Client, baseURL, path string, body interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("could not create PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PUT %s failed: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PUT %s returned status %d: %s", path, resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// policyHTTPDelete DELETEs baseURL+path. Accepts HTTP 200, 204, or 404 as success.
// A 404 is treated as success because the resource is already gone.
func policyHTTPDelete(ctx context.Context, client *http.Client, baseURL, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("could not create DELETE request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s failed: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("DELETE %s returned status %d: %s", path, resp.StatusCode, string(body))
}

// rawValueToString converts a json.RawMessage (as returned by the settings policy API for setting
// values) to a plain Go string suitable for storing in Terraform state.
// JSON strings are unquoted: `"RECOMMEND"` → `RECOMMEND`.
// JSON numbers/bools are returned as their string representation: `90.0`, `true`.
// Nil or empty input returns "".
func rawValueToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try to unquote a JSON string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Fall back to the raw bytes (numbers, bools, etc.).
	return string(raw)
}

// stringToRawValue converts a plain Go string (from Terraform state) back to json.RawMessage for
// inclusion in an API request body.
// If the string is a valid non-string JSON token (number, bool, null, object, array) it is used
// as-is. Otherwise it is JSON-encoded as a quoted string.
func stringToRawValue(s string) json.RawMessage {
	if s == "" {
		return nil
	}
	// Check if it's already a valid non-string JSON literal.
	var probe interface{}
	if err := json.Unmarshal([]byte(s), &probe); err == nil {
		// Make sure it wasn't interpreted as a string (strings need quoting).
		if _, isString := probe.(string); !isString {
			return json.RawMessage(s)
		}
	}
	// Quote it as a JSON string.
	quoted, _ := json.Marshal(s)
	return json.RawMessage(quoted)
}
