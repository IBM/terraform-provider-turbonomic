// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/IBM/turbonomic-go-client/logging"
)

type oAuthResp struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// authenticate performs authentication with the Turbonomic API
func authenticate(httpClient *http.Client, baseURL string, params *ClientParameters, logConfig logging.LoggerConfig) error {
	if params.Username != "" && params.Password != "" {
		return authenticateWithPassword(httpClient, baseURL, params, logConfig)
	} else if params.OAuthCreds.ClientId != "" && params.OAuthCreds.ClientSecret != "" {
		return authenticateWithOAuth(httpClient, baseURL, params, logConfig)
	}

	return fmt.Errorf("no valid authentication credentials provided")
}

// authenticateWithPassword authenticates using username and password
func authenticateWithPassword(httpClient *http.Client, baseURL string, params *ClientParameters, logConfig logging.LoggerConfig) error {
	loginURL := strings.TrimSuffix(baseURL, "/api/v3") + "/api/v3/login"

	payload := map[string]string{
		"username": params.Username,
		"password": params.Password,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login payload: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if params.ApiInfo.ApiOrigin != "" {
		req.Header.Set("X-API-Origin", params.ApiInfo.ApiOrigin)
	}
	if params.ApiInfo.Version != "" {
		req.Header.Set("X-API-Version", params.ApiInfo.Version)
	}

	logConfig.Logger.Debug(logConfig.Ctx, "attempting username/password authentication", "url", loginURL)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	logConfig.Logger.Info(logConfig.Ctx, "successfully authenticated with username/password")
	return nil
}

// authenticateWithOAuth authenticates using OAuth 2.0 client credentials
func authenticateWithOAuth(httpClient *http.Client, baseURL string, params *ClientParameters, logConfig logging.LoggerConfig) error {
	tokenURL := strings.TrimSuffix(baseURL, "/api/v3") + "/api/v3/oauth/token"

	// Try client_secret_basic first
	err := authenticateOAuthBasic(httpClient, tokenURL, params, logConfig)
	if err == nil {
		return nil
	}

	logConfig.Logger.Debug(logConfig.Ctx, "client_secret_basic failed, trying client_secret_post", "error", err)

	// Fall back to client_secret_post
	return authenticateOAuthPost(httpClient, tokenURL, params, logConfig)
}

// authenticateOAuthBasic uses client_secret_basic authentication
func authenticateOAuthBasic(httpClient *http.Client, tokenURL string, params *ClientParameters, logConfig logging.LoggerConfig) error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", params.OAuthCreds.Role)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create OAuth request: %w", err)
	}

	// Set Basic Auth header
	auth := base64.StdEncoding.EncodeToString([]byte(params.OAuthCreds.ClientId + ":" + params.OAuthCreds.ClientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	if params.ApiInfo.ApiOrigin != "" {
		req.Header.Set("X-API-Origin", params.ApiInfo.ApiOrigin)
	}
	if params.ApiInfo.Version != "" {
		req.Header.Set("X-API-Version", params.ApiInfo.Version)
	}

	logConfig.Logger.Debug(logConfig.Ctx, "attempting OAuth client_secret_basic authentication")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OAuth request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OAuth authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var oauthResp oAuthResp
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		return fmt.Errorf("failed to decode OAuth response: %w", err)
	}

	logConfig.Logger.Info(logConfig.Ctx, "successfully authenticated with OAuth client_secret_basic")
	return nil
}

// authenticateOAuthPost uses client_secret_post authentication
func authenticateOAuthPost(httpClient *http.Client, tokenURL string, params *ClientParameters, logConfig logging.LoggerConfig) error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", params.OAuthCreds.ClientId)
	data.Set("client_secret", params.OAuthCreds.ClientSecret)
	data.Set("scope", params.OAuthCreds.Role)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create OAuth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	if params.ApiInfo.ApiOrigin != "" {
		req.Header.Set("X-API-Origin", params.ApiInfo.ApiOrigin)
	}
	if params.ApiInfo.Version != "" {
		req.Header.Set("X-API-Version", params.ApiInfo.Version)
	}

	logConfig.Logger.Debug(logConfig.Ctx, "attempting OAuth client_secret_post authentication")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OAuth request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OAuth authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var oauthResp oAuthResp
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		return fmt.Errorf("failed to decode OAuth response: %w", err)
	}

	logConfig.Logger.Info(logConfig.Ctx, "successfully authenticated with OAuth client_secret_post")
	return nil
}
