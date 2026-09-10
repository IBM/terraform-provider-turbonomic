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
	"fmt"
)

var entityNameMap = map[string]string{
	"ApplicationComponentSpec":      "appComponentSpecsByName",
	"ApplicationComponent":          "appCompsByName",
	"AvailabilityZone":              "zonsByName",
	"BillingFamily":                 "billingFamilyByName",
	"BusinessAccountFolder":         "businessAccountFolderByName",
	"BusinessAccount":               "businessAccountByName",
	"BusinessApplication":           "busAppsByName",
	"BusinessTransaction":           "busTransByName",
	"BusinessUser":                  "businessUserByName",
	"Chassis":                       "chasByName",
	"Cluster":                       "clustersByName",
	"ComputeTier":                   "computeTiersByName",
	"ContainerPlatformCluster":      "containerPlatformClustersByName",
	"Container":                     "containersByName",
	"ContainerPod":                  "containerPodsByName",
	"ContainerSpec":                 "containerSpecsByName",
	"DataCenter":                    "datacentersByName",
	"Database":                      "databaseByName",
	"DatabaseServer":                "databaseServerByName",
	"DatabaseServerTier":            "databaseServerTiersByName",
	"DatabaseTier":                  "databaseTiersByName",
	"DesktopPool":                   "desktopPoolByName",
	"DiskArray":                     "diskarrayByName",
	"DocumentCollection":            "dcByName",
	"Group":                         "groupsByName",
	"IOModule":                      "ioModuleByName",
	"Internet":                      "internetByName",
	"LogicalPool":                   "logicalPoolByName",
	"Namespace":                     "namespacesByName",
	"Network":                       "netsByName",
	"PhysicalMachine":               "pmsByName",
	"Region":                        "regsByName",
	"ResourceGroup":                 "resourceGroupByName",
	"Service":                       "servicesByName",
	"Storage":                       "storageByName",
	"StorageCluster":                "storageClustersByName",
	"StorageController":             "storagecontrollerByName",
	"StorageTier":                   "storageTierByName",
	"Switch":                        "switchByName",
	"ViewPod":                       "viewPodByName",
	"VirtualDataCenter":             "vdcsByName",
	"VirtualMachine":                "vmsByName",
	"VirtualMachineCluster":         "virtualMachineClustersByName",
	"VirtualMachineSpec":            "virtualMachineSpecsByName",
	"VirtualVolume":                 "virtualVolumeByName",
	"WorkloadController":            "workloadControllersByName",
	"WorkloadControllerByNamespace": "workloadControllersByNamespace",
}

// Map of entity types to their corresponding vendor ID filter types
var vendorIdFilterMap = map[string]string{
	"VirtualMachine": "vmsByVendorId",
	"VirtualVolume":  "volumeById",
}

// Parameters for searching Turbonomic's API
type SearchRequest struct {
	Name             string
	EntityType       string
	EnvironmentType  string
	CloudType        string
	OSNames          []string
	CaseSensitive    bool
	CommonReqParams  CommonReqParams
	SearchParameters map[string]string
}
type SearchRequestByVendorId struct {
	EntityType      string
	VendorId        string
	CaseSensitive   bool
	CommonReqParams CommonReqParams
}

// Criterion for a Turbonomic API search request
type Criteria struct {
	CaseSensitive bool   `json:"caseSensitive"`
	ExpType       string `json:"expType"`
	ExpVal        string `json:"expVal"`
	FilterType    string `json:"filterType"`
}

// Body for POST request of Turbonomic API search request
type SearchDTO struct {
	CriteriaList    []Criteria `json:"criteriaList"`
	LogicalOperator string     `json:"logicalOperator"`
	ClassName       string     `json:"className"`
	Scope           string     `json:"scope,omitempty"`
	EnvironmentType string     `json:"environmentType,omitempty"`
	CloudType       string     `json:"cloudType,omitempty"`
}

// Results of a search request to Turbonomic's API
type SearchResults []struct {
	UUID            string `json:"uuid"`
	DisplayName     string `json:"displayName"`
	ClassName       string `json:"className"`
	EnvironmentType string `json:"environmentType"`
	DiscoveredBy    struct {
		UUID        string `json:"uuid"`
		DisplayName string `json:"displayName"`
		Category    string `json:"category"`
		Type        string `json:"type"`
		Readonly    bool   `json:"readonly"`
	} `json:"discoveredBy"`
	VendorIds         map[string]string `json:"vendorIds"`
	State             string            `json:"state"`
	Severity          string            `json:"severity"`
	CostPrice         float64           `json:"costPrice"`
	SeverityBreakdown struct{}          `json:"severityBreakdown"`
	Template          struct {
		Price       float64 `json:"price"`
		Discovered  bool    `json:"discovered"`
		EnableMatch bool    `json:"enableMatch"`
		DisplayName string  `json:"displayName"`
	} `json:"template"`
	Aspects struct {
		VirtualMachineAspect struct {
			Os                string   `json:"os"`
			IP                []string `json:"ip"`
			NumVCPUs          int      `json:"numVCPUs"`
			EbsOptimized      bool     `json:"ebsOptimized"`
			ResourceID        string   `json:"resourceId"`
			CreationTimeStamp int      `json:"creationTimeStamp"`
			Type              string   `json:"type"`
		} `json:"virtualMachineAspect"`
		VirtualDisksAspect struct {
			VirtualDisks []struct {
				UUID        string `json:"uuid"`
				DisplayName string `json:"displayName"`
				Tier        string `json:"tier"`
				Stats       []struct {
					Name     string `json:"name"`
					Capacity struct {
						Max   int `json:"max"`
						Min   int `json:"min"`
						Avg   int `json:"avg"`
						Total int `json:"total"`
					} `json:"capacity"`
					Filters []struct {
						Type        string      `json:"type"`
						Value       string      `json:"value"`
						DisplayName interface{} `json:"displayName"`
					} `json:"filters"`
					Units  string `json:"units"`
					Values struct {
						Max   int `json:"max"`
						Min   int `json:"min"`
						Avg   int `json:"avg"`
						Total int `json:"total"`
					} `json:"values"`
					Value int `json:"value"`
				} `json:"stats"`
				AttachedVirtualMachine struct {
					UUID        string `json:"uuid"`
					DisplayName string `json:"displayName"`
					ClassName   string `json:"className"`
				} `json:"attachedVirtualMachine"`
				Provider struct {
					UUID        string `json:"uuid"`
					DisplayName string `json:"displayName"`
					ClassName   string `json:"className"`
				} `json:"provider"`
				DataCenter struct {
					UUID        string `json:"uuid"`
					DisplayName string `json:"displayName"`
					ClassName   string `json:"className"`
				} `json:"dataCenter"`
				EnvironmentType string `json:"environmentType"`
				LastModified    int64  `json:"lastModified"`
				BusinessAccount struct {
					UUID            string `json:"uuid"`
					DisplayName     string `json:"displayName"`
					ClassName       string `json:"className"`
					EnvironmentType string `json:"environmentType"`
					DiscoveredBy    struct {
						UUID        string `json:"uuid"`
						DisplayName string `json:"displayName"`
						Category    string `json:"category"`
						Type        string `json:"type"`
						Readonly    bool   `json:"readonly"`
					} `json:"discoveredBy"`
					VendorIds struct {
						Turbonomicamp string `json:"turbonomicamp"`
					} `json:"vendorIds"`
					State             string `json:"state"`
					Severity          string `json:"severity"`
					SeverityBreakdown struct {
						NORMAL int `json:"NORMAL"`
					} `json:"severityBreakdown"`
					Tags struct {
						Usage []string `json:"Usage"`
					} `json:"tags"`
					Staleness string `json:"staleness"`
				} `json:"businessAccount"`
				SnapshotID        string  `json:"snapshotId"`
				Encryption        string  `json:"encryption"`
				AttachmentState   string  `json:"attachmentState"`
				HourlyBilledOps   float64 `json:"hourlyBilledOps"`
				CreationTimeStamp int64   `json:"creationTimeStamp"`
				ResourceID        string  `json:"resourceId"`
			} `json:"virtualDisks"`
			Type string `json:"type"`
		} `json:"virtualDisksAspect"`
	} `json:"aspects"`
	Tags map[string][]string `json:"tags"`
}

// Retrives the results of a search of Turbonomic's API based on provided parameters
func (c *Client) SearchEntityByName(searchReq SearchRequest) (SearchResults, error) {

	// var filterType string
	filterType, err := c.getFilterType(searchReq.EntityType)
	if err != nil {
		return SearchResults{}, err
	}
	searchCriteria := c.getSearchCriteria(searchReq, filterType)

	return c.SearchEntities(searchCriteria, searchReq.CommonReqParams)
}

// Retrives the results of a search of Turbonomic's API based on provided parameters
func (c *Client) SearchEntityByVendorId(searchReq SearchRequestByVendorId) (SearchResults, error) {
	searchCriteria := c.getSearchVendorIdCriteria(searchReq)
	return c.SearchEntities(searchCriteria, searchReq.CommonReqParams)
}

func (c *Client) SearchEntities(
	searchCriteria SearchDTO, reqParams CommonReqParams) (SearchResults, error) {

	dtoBuf := new(bytes.Buffer)
	if err := json.NewEncoder(dtoBuf).Encode(searchCriteria); err != nil {
		return nil, err
	}

	restResp, err := c.request(RequestOptions{Method: "POST", Path: "/search", ReqDTO: dtoBuf,
		CommonReqParams: CommonReqParams{
			Headers:         reqParams.Headers,
			QueryParameters: reqParams.QueryParameters}})
	if err != nil {
		return nil, err
	}

	var searchResults SearchResults
	if err := json.Unmarshal(restResp, &searchResults); err != nil {
		return nil, err
	}

	return searchResults, err
}

// Helper function to enable the use of entity type as the filter instead of
// longer parameter names required by Turbonomic's API
func (c *Client) getFilterType(entityType string) (string, error) {

	filterType := entityNameMap[entityType]
	if filterType != "" {
		return filterType, nil
	}
	return "", fmt.Errorf("entity type of %s not supported", entityType)
}

// Create searchCriteria using searchRequest parameters
func (c *Client) getSearchCriteria(searchReq SearchRequest, filterType string) SearchDTO {
	criteriaList := []Criteria{
		{
			CaseSensitive: searchReq.CaseSensitive,
			ExpType:       "EQ",
			ExpVal:        searchReq.Name,
			FilterType:    filterType,
		},
	}
	if len(searchReq.CloudType) > 0 {
		criteriaList = append(criteriaList, Criteria{
			CaseSensitive: false,
			ExpType:       "EQ",
			ExpVal:        searchReq.CloudType,
			FilterType:    "vmsByCloudProvider",
		})
	}
	if len(searchReq.OSNames) > 0 {
		expValue := ""
		for i, osName := range searchReq.OSNames {
			if len(osName) > 0 {
				expValue = expValue + osName + ".*"
				if i != len(searchReq.OSNames)-1 {
					expValue = expValue + "|"
				}
			}
		}
		if len(expValue) > 0 {
			criteriaList = append(criteriaList, Criteria{
				CaseSensitive: false,
				ExpType:       "RXEQ",
				ExpVal:        expValue,
				FilterType:    "vmsByGuestName",
			})
		}
	}
	// SearchParameters carries additional filterType→value pairs added by the caller.
	// Each non-empty value is appended as a case-insensitive EQ criterion.
	for ft, val := range searchReq.SearchParameters {
		if len(val) > 0 {
			criteriaList = append(criteriaList, Criteria{
				CaseSensitive: false,
				ExpType:       "EQ",
				ExpVal:        val,
				FilterType:    ft,
			})
		}
	}

	searchCriteria := SearchDTO{
		EnvironmentType: searchReq.EnvironmentType,
		LogicalOperator: "AND",
		ClassName:       searchReq.EntityType,
		CriteriaList:    criteriaList,
	}
	return searchCriteria
}

func (c *Client) getSearchVendorIdCriteria(searchReq SearchRequestByVendorId) SearchDTO {
	var criteriaList []Criteria
	if len(searchReq.VendorId) > 0 {
		// Only add the criteria if the entity type has a mapping in vendorIdFilterMap
		if filterType, exists := vendorIdFilterMap[searchReq.EntityType]; exists {
			criteriaList = append(criteriaList, Criteria{
				CaseSensitive: searchReq.CaseSensitive,
				ExpType:       "RXEQ",
				ExpVal:        searchReq.VendorId,
				FilterType:    filterType,
			})
		}
	}
	searchCriteria := SearchDTO{
		LogicalOperator: "AND",
		ClassName:       searchReq.EntityType,
		CriteriaList:    criteriaList,
	}
	return searchCriteria
}
