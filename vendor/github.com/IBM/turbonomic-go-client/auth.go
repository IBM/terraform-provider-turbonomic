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

package turboclient

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/IBM/turbonomic-go-client/logging"
)

// Parameters for authenticating to the Turbonomic API
type AuthRequest struct {
	basePath   string
	hostname   string
	username   string
	password   string
	oAuthCreds OAuthCreds
	httpClient *http.Client
	apiInfo    ApiInfo
}

type oAuthResp struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type authMethod string

const (
	usernamePassword  authMethod = "username/password"
	clientSecretBasic authMethod = "client_secret_basic"
	clientSecretPost  authMethod = "client_secret_post"
)

// Creates authorized Turbonomic API Client
func clientAuth(authreq *AuthRequest, logConfig logging.LoggerConfig) (*Client, error) {

	if authreq == nil {
		return nil, errors.New("please provide valid credentials")
	}

	urlPath, payload, err := setAuthParams(*authreq)
	if err != nil {
		return nil, err
	}

	var clientMethod authMethod
	if strings.HasSuffix(urlPath, "/login") {
		clientMethod = usernamePassword
	} else {
		clientMethod = clientSecretBasic
	}

	req, err := buildAuthRequest(authreq, urlPath, payload)
	if err != nil {
		return nil, err
	}

	resp, err := authreq.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logConfig.Logger.Error(logConfig.Ctx, "failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode == http.StatusUnauthorized {
		logConfig.Logger.Debug(logConfig.Ctx, "authentication failed for client_secret_basic method, trying client_secret_post")
		if closeErr := resp.Body.Close(); closeErr != nil {
			logConfig.Logger.Error(logConfig.Ctx, "failed to close response body", "error", closeErr)
		}
		req, err = buildOAuthPostRequest(authreq, urlPath)
		if err != nil {
			return nil, err
		}
		resp, err = authreq.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		clientMethod = clientSecretPost
		defer func() {
			if closeErr := resp.Body.Close(); closeErr != nil {
				logConfig.Logger.Error(logConfig.Ctx, "failed to close response body", "error", closeErr)
			}
		}()
	}

	if resp.StatusCode >= http.StatusBadRequest {
		logConfig.Logger.Error(logConfig.Ctx, "failed to establish a connection with the Turbonomic instance:", "Status", resp.Status)
		return nil, errors.New(resp.Status)
	}

	logConfig.Logger.Debug(logConfig.Ctx, fmt.Sprintf("successfully logged into Turbonomic using %s authentication method", clientMethod))

	newClient, err := buildClientFromResponse(authreq, resp)
	if err != nil {
		logConfig.Logger.Error(logConfig.Ctx, err.Error())
		return nil, err
	}

	newClient.Logger = logConfig.Logger
	newClient.Ctx = logConfig.Ctx

	return newClient, nil
}

func setAuthParams(authreq AuthRequest) (urlPath string, payload *strings.Reader, err error) {
	if (authreq.username) != "" && (authreq.password != "") {
		urlPath = "https://" + authreq.hostname + authreq.basePath + "/login"
		payload = strings.NewReader("username=" + authreq.username + "&password=" + authreq.password)
	} else if (authreq.oAuthCreds.Role.String() != "") &&
		(authreq.oAuthCreds.ClientId != "") &&
		(authreq.oAuthCreds.ClientSecret != "") {

		urlPath = "https://" + authreq.hostname + "/oauth2/token"
		payload = strings.NewReader("grant_type=" + "client_credentials" +
			"&scope=role:" + authreq.oAuthCreds.Role.String())
	} else {
		return "", nil, errors.New("please provide valid credentials; username/password or oauth2")
	}
	return urlPath, payload, nil

}

func buildAuthRequest(authreq *AuthRequest, urlPath string, payload *strings.Reader) (*http.Request, error) {
	req, err := http.NewRequest("POST", urlPath, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if origin := authreq.apiInfo.ApiOrigin; origin != "" {
		req.Header.Add("User-Agent", origin+"/"+authreq.apiInfo.Version)
	}

	if creds := authreq.oAuthCreds; creds.ClientId != "" && creds.ClientSecret != "" {
		auth := creds.ClientId + ":" + creds.ClientSecret
		req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}

	return req, nil
}

func buildOAuthPostRequest(authreq *AuthRequest, urlPath string) (*http.Request, error) {
	payload := strings.NewReader("grant_type=client_credentials" +
		"&scope=role:" + authreq.oAuthCreds.Role.String() +
		"&client_id=" + authreq.oAuthCreds.ClientId +
		"&client_secret=" + authreq.oAuthCreds.ClientSecret)

	req, err := http.NewRequest("POST", urlPath, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func buildClientFromResponse(authreq *AuthRequest, resp *http.Response) (*Client, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result oAuthResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	client := &Client{
		BaseURL:    "https://" + authreq.hostname + authreq.basePath,
		HTTPClient: authreq.httpClient,
		Headers:    make(map[string]string),
	}

	if result.AccessToken != "" {
		client.Headers["Authorization"] = "Bearer " + result.AccessToken
	}
	if origin := authreq.apiInfo.ApiOrigin; origin != "" {
		client.Headers["User-Agent"] = origin + "/" + authreq.apiInfo.Version
	}

	return client, nil
}
