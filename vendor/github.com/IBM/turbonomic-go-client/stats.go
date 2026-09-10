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
	"time"
)

// StatsRequest represents the parameters for retrieving statistics from Turbonomic's API
type StatsRequest struct {
	EntityUUID      string
	StartDate       string
	EndDate         string
	Statistics      []StatisticRequest
	CommonReqParams CommonReqParams
}

// StatisticRequest represents a single statistic to be requested
type StatisticRequest struct {
	Name              string   `json:"name"`
	RelatedEntityType string   `json:"relatedEntityType,omitempty"`
	Filters           []Filter `json:"filters,omitempty"`
}

// Filter represents a filter to be applied to the statistics
type Filter struct {
	Type        string      `json:"type"`
	Value       string      `json:"value"`
	DisplayName interface{} `json:"displayName"`
}

// StatsRequestBody represents the request body for the statistics API
type StatsRequestBody struct {
	StartDate  string             `json:"startDate,omitempty"`
	EndDate    string             `json:"endDate,omitempty"`
	Statistics []StatisticRequest `json:"statistics"`
}

// StatsResponse represents the response from the statistics API
type StatsResponse []EntityStats

// EntityStats represents statistics for a single entity
type EntityStats struct {
	DisplayName string      `json:"displayName"`
	Date        time.Time   `json:"date"`
	Statistics  []Statistic `json:"statistics"`
	Epoch       string      `json:"epoch"`
}

// Statistic represents a single statistic in the response
type Statistic struct {
	Name             string                 `json:"name"`
	Capacity         StatValues             `json:"capacity"`
	Reserved         StatValues             `json:"reserved"`
	Filters          []Filter               `json:"filters"`
	RelatedEntity    *RelatedEntity         `json:"relatedEntity,omitempty"`
	Units            string                 `json:"units"`
	Values           StatValues             `json:"values"`
	Value            float64                `json:"value"`
	CommoditySource  map[string]interface{} `json:"commoditySource"`
	HistUtilizations []HistUtilization      `json:"histUtilizations,omitempty"`
}

// RelatedEntity represents a related entity in a statistic
type RelatedEntity struct {
	UUID string `json:"uuid"`
}

// StatValues represents statistical values
type StatValues struct {
	Max      float64 `json:"max"`
	Min      float64 `json:"min"`
	Avg      float64 `json:"avg"`
	Total    float64 `json:"total"`
	TotalMax float64 `json:"totalMax,omitempty"`
	TotalMin float64 `json:"totalMin,omitempty"`
}

// HistUtilization represents historical utilization data
type HistUtilization struct {
	Type     string  `json:"type"`
	Usage    float64 `json:"usage"`
	Capacity float64 `json:"capacity"`
}

// GetStats retrieves statistics from Turbonomic's API based on request parameters
func (c *Client) GetStats(statsReq StatsRequest) (StatsResponse, error) {
	requestBody := StatsRequestBody{
		StartDate:  statsReq.StartDate,
		EndDate:    statsReq.EndDate,
		Statistics: statsReq.Statistics,
	}

	dtoBuf := new(bytes.Buffer)
	if err := json.NewEncoder(dtoBuf).Encode(requestBody); err != nil {
		return nil, err
	}

	urlPath := "/stats/" + statsReq.EntityUUID
	reqDTO := RequestOptions{
		Method: "POST",
		Path:   urlPath,
		ReqDTO: dtoBuf,
		CommonReqParams: CommonReqParams{
			Headers:         statsReq.CommonReqParams.Headers,
			QueryParameters: statsReq.CommonReqParams.QueryParameters,
		},
	}

	restResp, err := c.request(reqDTO)
	if err != nil {
		return nil, err
	}

	var statsResponse StatsResponse
	if err := json.Unmarshal(restResp, &statsResponse); err != nil {
		return nil, err
	}

	return statsResponse, nil
}
