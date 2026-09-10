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

// Package v2 provides a wrapper around the auto-generated Turbonomic API client
// with authentication, session management, and backward compatibility.
package v2

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/IBM/turbonomic-go-client/api/generated"
	"github.com/IBM/turbonomic-go-client/logging"
)

// ClientParameters contains the configuration for creating a Turbonomic client
type ClientParameters struct {
	Hostname   string
	Username   string
	Password   string
	OAuthCreds OAuthCreds
	Skipverify bool
	ApiInfo    ApiInfo
}

// OAuthCreds contains OAuth 2.0 credentials
type OAuthCreds struct {
	ClientId     string
	ClientSecret string
	Role         string
}

// ApiInfo contains API metadata
type ApiInfo struct {
	ApiOrigin string // e.g., "terraform-provider"
	Version   string // e.g., "1.0.0"
}

// Client wraps the generated API client with authentication and session management
type Client struct {
	// Generated API client
	api *generated.Client

	// HTTP client with cookie jar for session management
	httpClient *http.Client

	// Configuration
	baseURL string
	apiInfo ApiInfo
	logger  logging.LoggerCustom
	ctx     context.Context
}

// NewClient creates a new Turbonomic API client with authentication
func NewClient(params *ClientParameters) (*Client, error) {
	if params == nil {
		return nil, fmt.Errorf("client parameters cannot be nil")
	}

	logConfig := logging.SetLogConfig(nil)

	// Create HTTP client with cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	httpClient := &http.Client{
		Jar:     jar,
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: params.Skipverify,
			},
		},
	}

	// Build base URL
	baseURL := fmt.Sprintf("https://%s/api/v3", params.Hostname)

	// Authenticate and get session
	if err := authenticate(httpClient, baseURL, params, logConfig); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	apiClient, err := generated.NewClient(baseURL, generated.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	client := &Client{
		api:        apiClient,
		httpClient: httpClient,
		baseURL:    baseURL,
		apiInfo:    params.ApiInfo,
		logger:     logConfig.Logger,
		ctx:        logConfig.Ctx,
	}

	return client, nil
}

// NewClientWithHTTPClient creates a new Turbonomic API client using an existing authenticated HTTP client
// This is useful when you want to reuse an existing session from another client
func NewClientWithHTTPClient(httpClient *http.Client, hostname string, apiInfo ApiInfo) (*Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("HTTP client cannot be nil")
	}

	logConfig := logging.SetLogConfig(nil)

	// Build base URL
	baseURL := fmt.Sprintf("https://%s/api/v3", hostname)

	// Create the generated API client with the provided HTTP client
	apiClient, err := generated.NewClient(baseURL, generated.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	client := &Client{
		api:        apiClient,
		httpClient: httpClient,
		baseURL:    baseURL,
		apiInfo:    apiInfo,
		logger:     logConfig.Logger,
		ctx:        logConfig.Ctx,
	}

	return client, nil
}

// GetAPIClient returns the underlying generated API client for direct access
// Use this when you need to call API methods not yet wrapped by this package
func (c *Client) GetAPIClient() *generated.Client {
	return c.api
}

// GetHTTPClient returns the underlying HTTP client
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// GetBaseURL returns the base URL of the API
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetContext returns the client context
func (c *Client) GetContext() context.Context {
	return c.ctx
}
