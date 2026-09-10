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

// Parameters for retriving actions from Turbonomic's API
type ActionsRequest struct {
	Uuid            string
	ActionState     []string
	ActionType      []string
	DetailLevel     string
	Headers         map[string]string
	QueryParameters map[string]string
}

// Criteria passed for retriving actions
type ActionsCriteria struct {
	ActionStateList []string `json:"actionStateList"`
	ActionTypeList  []string `json:"actionTypeList"`
	DetailLevel     string   `json:"detailLevel,omitempty"`
}

// Results of GetActions Turbonomc API call
type ActionResults []struct {
	UUID                   string  `json:"uuid"`
	DisplayName            string  `json:"displayName"`
	ActionImpactID         int64   `json:"actionImpactID"`
	MarketID               int     `json:"marketID"`
	CreateTime             string  `json:"createTime"`
	ActionType             string  `json:"actionType"`
	ActionState            string  `json:"actionState"`
	ActionMode             string  `json:"actionMode"`
	ActionStateDescription string  `json:"actionStateDescription"`
	Details                string  `json:"details"`
	Importance             float32 `json:"importance"`
	Target                 struct {
		UUID            string `json:"uuid"`
		DisplayName     string `json:"displayName"`
		ClassName       string `json:"className"`
		EnvironmentType string `json:"environmentType"`
		DiscoveredBy    struct {
			UUID              string `json:"uuid"`
			DisplayName       string `json:"displayName"`
			Category          string `json:"category"`
			IsProbeRegistered bool   `json:"isProbeRegistered"`
			Type              string `json:"type"`
			Readonly          bool   `json:"readonly"`
		} `json:"discoveredBy"`
		VendorIds map[string]string   `json:"vendorIds"`
		State     string              `json:"state"`
		Aspects   json.RawMessage     `json:"aspects"`
		Tags      map[string][]string `json:"tags"`
	} `json:"target"`
	CurrentEntity struct {
		UUID            string `json:"uuid"`
		DisplayName     string `json:"displayName"`
		ClassName       string `json:"className"`
		EnvironmentType string `json:"environmentType"`
		DiscoveredBy    struct {
			UUID              string `json:"uuid"`
			DisplayName       string `json:"displayName"`
			Category          string `json:"category"`
			IsProbeRegistered bool   `json:"isProbeRegistered"`
			Type              string `json:"type"`
			Readonly          bool   `json:"readonly"`
		} `json:"discoveredBy"`
		VendorIds map[string]string `json:"vendorIds"`
		State     string            `json:"state"`
	} `json:"currentEntity"`
	NewEntity struct {
		UUID            string `json:"uuid"`
		DisplayName     string `json:"displayName"`
		ClassName       string `json:"className"`
		EnvironmentType string `json:"environmentType"`
		DiscoveredBy    struct {
			UUID              string `json:"uuid"`
			DisplayName       string `json:"displayName"`
			Category          string `json:"category"`
			IsProbeRegistered bool   `json:"isProbeRegistered"`
			Type              string `json:"type"`
			Readonly          bool   `json:"readonly"`
		} `json:"discoveredBy"`
		VendorIds map[string]string `json:"vendorIds"`
		State     string            `json:"state"`
	} `json:"newEntity"`
	CurrentValue    string `json:"currentValue"`
	NewValue        string `json:"newValue"`
	ValueUnits      string `json:"valueUnits"`
	ResizeAttribute string `json:"resizeAttribute"`
	Template        struct {
		UUID        string `json:"uuid"`
		DisplayName string `json:"displayName"`
		ClassName   string `json:"className"`
		Discovered  bool   `json:"discovered"`
		EnableMatch bool   `json:"enableMatch"`
	} `json:"template"`
	Risk struct {
		SubCategory       string   `json:"subCategory"`
		Description       string   `json:"description"`
		Severity          string   `json:"severity"`
		Importance        float32  `json:"importance"`
		ReasonCommodities []string `json:"reasonCommodities"`
	} `json:"risk"`
	Stats []struct {
		Name    string `json:"name"`
		Filters []struct {
			Type        string `json:"type"`
			Value       string `json:"value"`
			DisplayName string `json:"displayName"`
		} `json:"filters"`
		Units string  `json:"units"`
		Value float64 `json:"value"`
	} `json:"stats"`
	CurrentLocation struct {
		UUID            string `json:"uuid"`
		DisplayName     string `json:"displayName"`
		ClassName       string `json:"className"`
		EnvironmentType string `json:"environmentType"`
		DiscoveredBy    struct {
			UUID              string `json:"uuid"`
			DisplayName       string `json:"displayName"`
			Category          string `json:"category"`
			IsProbeRegistered bool   `json:"isProbeRegistered"`
			Type              string `json:"type"`
			Readonly          bool   `json:"readonly"`
		} `json:"discoveredBy"`
		VendorIds map[string]string `json:"vendorIds"`
	} `json:"currentLocation"`
	NewLocation struct {
		UUID            string `json:"uuid"`
		DisplayName     string `json:"displayName"`
		ClassName       string `json:"className"`
		EnvironmentType string `json:"environmentType"`
		DiscoveredBy    struct {
			UUID              string `json:"uuid"`
			DisplayName       string `json:"displayName"`
			Category          string `json:"category"`
			IsProbeRegistered bool   `json:"isProbeRegistered"`
			Type              string `json:"type"`
			Readonly          bool   `json:"readonly"`
		} `json:"discoveredBy"`
		VendorIds map[string]string `json:"vendorIds"`
	} `json:"newLocation"`
	ActionSchedule struct {
		UUID                               string `json:"uuid"`
		DisplayName                        string `json:"displayName"`
		NextOccurrence                     string `json:"nextOccurrence"`
		NextOccurrenceTimestamp            int64  `json:"nextOccurrenceTimestamp"`
		TimeZoneId                         string `json:"timeZoneId"`
		Mode                               string `json:"mode"`
		AcceptedByUserForMaintenanceWindow bool   `json:"acceptedByUserForMaintenanceWindow"`
		RemaingTimeActiveInMs              int64  `json:"remaingTimeActiveInMs"`
	}
	CompoundActions []struct {
		DisplayName string `json:"displayName"`
		ActionType  string `json:"actionType"`
		ActionState string `json:"actionState"`
		ActionMode  string `json:"actionMode"`
		Details     string `json:"details"`
		Target      struct {
			UUID            string `json:"uuid"`
			DisplayName     string `json:"displayName"`
			ClassName       string `json:"className"`
			EnvironmentType string `json:"environmentType"`
			DiscoveredBy    struct {
				UUID              string `json:"uuid"`
				DisplayName       string `json:"displayName"`
				Category          string `json:"category"`
				IsProbeRegistered bool   `json:"isProbeRegistered"`
				Type              string `json:"type"`
				Readonly          bool   `json:"readonly"`
			} `json:"discoveredBy"`
			VendorIds map[string]string   `json:"vendorIds"`
			State     string              `json:"state"`
			Tags      map[string][]string `json:"tags"`
		} `json:"target"`
		CurrentEntity struct {
			UUID            string `json:"uuid"`
			DisplayName     string `json:"displayName"`
			ClassName       string `json:"className"`
			EnvironmentType string `json:"environmentType"`
			DiscoveredBy    struct {
				UUID              string `json:"uuid"`
				DisplayName       string `json:"displayName"`
				Category          string `json:"category"`
				IsProbeRegistered bool   `json:"isProbeRegistered"`
				Type              string `json:"type"`
				Readonly          bool   `json:"readonly"`
			} `json:"discoveredBy"`
			VendorIds map[string]string `json:"vendorIds"`
			State     string            `json:"state"`
		} `json:"currentEntity"`
		NewEntity struct {
			UUID            string `json:"uuid"`
			DisplayName     string `json:"displayName"`
			ClassName       string `json:"className"`
			EnvironmentType string `json:"environmentType"`
			DiscoveredBy    struct {
				UUID              string `json:"uuid"`
				DisplayName       string `json:"displayName"`
				Category          string `json:"category"`
				IsProbeRegistered bool   `json:"isProbeRegistered"`
				Type              string `json:"type"`
				Readonly          bool   `json:"readonly"`
			} `json:"discoveredBy"`
			VendorIds map[string]string `json:"vendorIds"`
			State     string            `json:"state"`
		} `json:"newEntity"`
		CurrentValue    string `json:"currentValue"`
		NewValue        string `json:"newValue"`
		ValueUnits      string `json:"valueUnits"`
		ResizeAttribute string `json:"resizeAttribute"`
		Risk            struct {
			SubCategory       string   `json:"subCategory"`
			Description       string   `json:"description"`
			Severity          string   `json:"severity"`
			Importance        float32  `json:"importance"`
			ReasonCommodities []string `json:"reasonCommodities"`
		} `json:"risk"`
	} `json:"compoundActions"`
	Source   string `json:"source"`
	ActionID int64  `json:"actionID"`
}

// Retrives actions from Turbonomic's API based on request parameters
func (c *Client) GetActionsByUUID(actionReq ActionsRequest) (ActionResults, error) {

	actionCriteria := ActionsCriteria{
		ActionStateList: actionReq.ActionState,
		ActionTypeList:  actionReq.ActionType,
		DetailLevel:     actionReq.DetailLevel,
	}

	dtoBuf := new(bytes.Buffer)
	if err := json.NewEncoder(dtoBuf).Encode(actionCriteria); err != nil {
		return nil, err
	}
	urlPath := "/entities/" + actionReq.Uuid + "/actions"
	reqDTO := RequestOptions{
		Method: "POST",
		Path:   urlPath,
		ReqDTO: dtoBuf,
		CommonReqParams: CommonReqParams{
			Headers:         actionReq.Headers,
			QueryParameters: actionReq.QueryParameters}}

	restResp, err := c.request(reqDTO)
	if err != nil {
		return nil, err
	}

	var actionResults ActionResults

	if err := json.Unmarshal(restResp, &actionResults); err != nil {
		return nil, err
	}

	return actionResults, err
}
