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

// ParkingEntityResult holds the response from GET /api/v3/parking/entities/{uuid}.
// The parking API exposes cluster stop/start state managed by Turbonomic Smart Parking;
// it is separate from the general /entities/{uuid}/actions endpoint.
type ParkingEntityResult struct {
	UUID                       string                 `json:"uuid"`
	DisplayName                string                 `json:"displayName"`
	Provider                   string                 `json:"provider"`
	State                      string                 `json:"state"`
	EntityType                 string                 `json:"entityType"`
	CloudServiceName           string                 `json:"cloudServiceName"`
	Cost                       float64                `json:"cost"`
	OnDemandRateWhenRunning    float64                `json:"onDemandRateWhenRunning"`
	ProjectedSavings           float64                `json:"projectedSavings"`
	SmartParkingRecommendation map[string]interface{} `json:"smartParkingRecommendation"`
}

// GetParkingEntity retrieves the parking state for a cluster entity by UUID.
// It calls GET /api/v3/parking/entities/{uuid} and returns the decoded result.
//
// Valid ParkingState values: RUNNING, STOPPED, SUSPENDED, MAINTENANCE, FAILOVER,
// UNKNOWN, STARTING, STOPPING, UNINITIALIZED.
//
// SmartParkingRecommendation is an empty map ({}) when no schedule recommendation
// is active, or contains uuid/displayName of the applicable parking schedule.
func (c *Client) GetParkingEntity(uuid string) (*ParkingEntityResult, error) {
	restResp, err := c.request(RequestOptions{
		Method: "GET",
		Path:   "/parking/entities/" + uuid,
		ReqDTO: new(bytes.Buffer),
	})
	if err != nil {
		return nil, err
	}
	c.Logger.Debug(c.Ctx, string(restResp))

	var result ParkingEntityResult
	if err := json.Unmarshal(restResp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
