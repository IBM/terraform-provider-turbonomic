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
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/IBM/turbonomic-go-client/logging"
)

// Default base path of Turbonomic API
const (
	BasePathv3 = "/api/v3"
)

// Parameters for creating a Turbonomic client
type ClientParameters struct {
	Baseurl    string
	Hostname   string
	Username   string
	Password   string
	OAuthCreds OAuthCreds
	Skipverify bool
	ApiInfo    ApiInfo
}
type OAuthCreds struct {
	ClientId     string
	ClientSecret string
	Role         TurboRoles
}

type ApiInfo struct {
	ApiOrigin string
	Version   string
}
type T8cClient interface {
	GetActionsByUUID(actionReq ActionsRequest) (ActionResults, error)
	GetEntity(reqOpts EntityRequest) (*EntityResults, error)
	GetEntityTags(reqOpts EntityRequest) ([]Tag, error)
	TagEntity(reqOpts TagEntityRequest) ([]Tag, error)
	SearchEntities(searchCriteria SearchDTO, reqParams CommonReqParams) (SearchResults, error)
	SearchEntityByName(searchReq SearchRequest) (SearchResults, error)
	SearchEntityByVendorId(searchReq SearchRequestByVendorId) (SearchResults, error)
	GetStats(statsReq StatsRequest) (StatsResponse, error)
}

// Turbonomic Client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Headers    map[string]string
	Logger     logging.LoggerCustom
	Ctx        context.Context
}

type CommonReqParams struct {
	QueryParameters map[string]string
	Headers         map[string]string
}

// Parameters for making a request to Turbonomic's API
type RequestOptions struct {
	Method          string
	Path            string
	ReqDTO          *bytes.Buffer
	CommonReqParams CommonReqParams
}

type TurboRoles int

const (
	ADMINISTRATOR TurboRoles = iota + 1
	SITE_ADMIN
	AUTOMATOR
	DEPLOYER
	ADVISOR
	OBSERVER
	OPERATIONAL_OBSERVER
	SHARED_ADVISOR
	SHARED_OBSERVER
	REPORT_EDITOR
)

func (t TurboRoles) String() string {
	return [...]string{"UNSET",
		"ADMINISTRATOR",
		"SITE_ADMIN",
		"AUTOMATOR",
		"DEPLOYER",
		"ADVISOR",
		"OBSERVER",
		"OPERATIONAL_OBSERVER",
		"SHARED_ADVISOR",
		"SHARED_OBSERVER",
		"REPORT_EDITOR"}[t]
}

func GetRolefromString(role string) TurboRoles {
	switch strings.ToUpper(role) {
	case "ADMINISTRATOR":
		return ADMINISTRATOR
	case "SITE_ADMIN":
		return SITE_ADMIN
	case "AUTOMATOR":
		return AUTOMATOR
	case "DEPLOYER":
		return DEPLOYER
	case "ADVISOR":
		return ADVISOR
	case "OBSERVER":
		return OBSERVER
	case "OPERATIONAL_OBSERVER":
		return OPERATIONAL_OBSERVER
	case "SHARED_ADVISOR":
		return SHARED_ADVISOR
	case "SHARED_OBSERVER":
		return SHARED_OBSERVER
	case "REPORT_EDITOR":
		return REPORT_EDITOR
	default:
		panic("unrecognized Turbonomic role")
	}
}

// Creates a new instance of the Turbonomic Client
func NewClient(clientParams *ClientParameters, options ...logging.LoggingOption) (T8cClient, error) {

	logConfig := logging.SetLogConfig(options)

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	if clientParams.Skipverify {
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: clientParams.Skipverify}
	}

	var basepath string
	if clientParams.Baseurl == "" {
		basepath = BasePathv3
	} else {
		basepath = clientParams.Baseurl
	}
	jar, _ := cookiejar.New(nil)

	client := &AuthRequest{
		basePath:   basepath,
		hostname:   clientParams.Hostname,
		username:   clientParams.Username,
		password:   clientParams.Password,
		oAuthCreds: clientParams.OAuthCreds,
		httpClient: &http.Client{
			Jar:       jar,
			Timeout:   time.Minute,
			Transport: customTransport,
		},
		apiInfo: clientParams.ApiInfo,
	}

	return clientAuth(client, logConfig)

}

// Returns *url.URL with query parameters set
func setParams(baseUrl string, QueryParameters map[string]string) (*url.URL, error) {
	fullUrl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	query := fullUrl.Query()
	for k, v := range QueryParameters {
		query.Set(k, v)
	}

	fullUrl.RawQuery = query.Encode()
	return fullUrl, err
}

// Make request to Turbonomic API using http package
func (c *Client) request(reqOpt RequestOptions) ([]byte, error) {

	baseUrl := c.BaseURL + reqOpt.Path
	fullUrl, err := setParams(baseUrl, reqOpt.CommonReqParams.QueryParameters)
	if err != nil {
		return nil, err
	}

	restReq, err := http.NewRequest(reqOpt.Method, fullUrl.String(), reqOpt.ReqDTO)
	if err != nil {
		return nil, err
	}

	restReq.Header.Add("Content-Type", "application/json")

	for k, v := range c.Headers {
		restReq.Header.Set(k, v)
	}

	for k, v := range reqOpt.CommonReqParams.Headers {
		restReq.Header.Set(k, v)
	}

	restResp, err := c.HTTPClient.Do(restReq)
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(restResp.Body)
	defer func() {
		if closeErr := restResp.Body.Close(); closeErr != nil {
			c.Logger.Error(c.Ctx, "failed to close response body", "error", closeErr)
		}
	}()

	if restResp.StatusCode >= 400 {
		err = errors.New(string(respBody))
		return respBody, err
	}

	if err != nil {
		return nil, err
	}

	return respBody, err
}
