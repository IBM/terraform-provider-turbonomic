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
	"encoding/json"
)

// Parameters for retriving an entity from Turbonomic's API
type EntityRequest struct {
	Uuid             string
	CommonReqOptions CommonReqParams
}

// Parameters for tagging an entity using Turbonomic's API
type TagEntityRequest struct {
	Uuid             string
	Tags             []Tag
	CommonReqOptions CommonReqParams
}

// Result for retriving an entity tag from Turbonomic's API
type Tag struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

// Results from GetEntity request
type EntityResults struct {
	UUID            string `json:"uuid,omitempty"`
	DisplayName     string `json:"displayName,omitempty"`
	ClassName       string `json:"className,omitempty"`
	EnvironmentType string `json:"environmentType,omitempty"`
	DiscoveredBy    struct {
		UUID        string `json:"uuid,omitempty"`
		DisplayName string `json:"displayName,omitempty"`
		Category    string `json:"category,omitempty"`
		Type        string `json:"type,omitempty"`
		Readonly    bool   `json:"readonly,omitempty"`
	} `json:"discoveredBy,omitempty"`
	VendorIds struct {
		Turbonomicamp string `json:"turbonomicamp,omitempty"`
	} `json:"vendorIds,omitempty"`
	State             string  `json:"state,omitempty"`
	Severity          string  `json:"severity,omitempty"`
	CostPrice         float64 `json:"costPrice,omitempty"`
	SeverityBreakdown struct {
		UNKNOWN  int `json:"UNKNOWN,omitempty"`
		NORMAL   int `json:"NORMAL,omitempty"`
		MINOR    int `json:"MINOR,omitempty"`
		MAJOR    int `json:"MAJOR,omitempty"`
		CRITICAL int `json:"CRITICAL,omitempty"`
	} `json:"severityBreakdown,omitempty"`
	Providers []struct {
		UUID        string `json:"uuid,omitempty"`
		DisplayName string `json:"displayName,omitempty"`
		ClassName   string `json:"className,omitempty"`
	} `json:"providers,omitempty"`
	Template struct {
		UUID        string  `json:"uuid,omitempty"`
		DisplayName string  `json:"displayName,omitempty"`
		Price       float64 `json:"price,omitempty"`
		Discovered  bool    `json:"discovered,omitempty"`
		EnableMatch bool    `json:"enableMatch,omitempty"`
	} `json:"template,omitempty"`
	Tags      struct{} `json:"tags,omitempty"`
	Staleness string   `json:"staleness,omitempty"`
}

// Retrives entity based on its provided uuid
func (c *Client) GetEntity(reqOpts EntityRequest) (*EntityResults, error) {

	restResp, err := c.request(RequestOptions{Method: "GET", Path: "/entities/" + reqOpts.Uuid, ReqDTO: new(bytes.Buffer),
		CommonReqParams: CommonReqParams{
			Headers:         reqOpts.CommonReqOptions.Headers,
			QueryParameters: reqOpts.CommonReqOptions.QueryParameters}})

	if err != nil {
		return nil, err
	}
	c.Logger.Debug(c.Ctx, string(restResp))

	var entityResults EntityResults
	if err := json.Unmarshal(restResp, &entityResults); err != nil {
		return nil, err
	}

	return &entityResults, err
}

// Tags entity by provided entity uuid
func (c *Client) TagEntity(reqOpts TagEntityRequest) ([]Tag, error) {

	dtoBuf := new(bytes.Buffer)
	if err := json.NewEncoder(dtoBuf).Encode(reqOpts.Tags); err != nil {
		return nil, err
	}

	restResp, err := c.request(RequestOptions{Method: "POST", Path: "/entities/" + reqOpts.Uuid + "/tags", ReqDTO: dtoBuf,
		CommonReqParams: CommonReqParams{
			Headers:         reqOpts.CommonReqOptions.Headers,
			QueryParameters: reqOpts.CommonReqOptions.QueryParameters}})

	if err != nil {
		return nil, err
	}
	c.Logger.Debug(c.Ctx, string(restResp))

	var tagsResult []Tag
	if err := json.Unmarshal(restResp, &tagsResult); err != nil {
		return nil, err
	}

	return tagsResult, err
}

// Retrives entity tags by provided entity uuid
func (c *Client) GetEntityTags(reqOpts EntityRequest) ([]Tag, error) {

	restResp, err := c.request(RequestOptions{Method: "GET", Path: "/entities/" + reqOpts.Uuid + "/tags", ReqDTO: new(bytes.Buffer),
		CommonReqParams: CommonReqParams{
			Headers:         reqOpts.CommonReqOptions.Headers,
			QueryParameters: reqOpts.CommonReqOptions.QueryParameters}})

	if err != nil {
		return nil, err
	}
	c.Logger.Debug(c.Ctx, string(restResp))

	var tagsResult []Tag
	if err := json.Unmarshal(restResp, &tagsResult); err != nil {
		return nil, err
	}

	return tagsResult, err
}
