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

// This is a sample file that needs to be populated in order to run integration tests.
// All tests are pulling data from this file.

package turboclient

var TurboHost string = ""
var TurboUser string = ""
var TurboPass string = ""
var DoNotVerify bool = true

type TestEntity struct {
	uuid        string
	displayName string
	className   string
}

type TestSearch struct {
	entityName      string
	entityType      string
	environmentType string
	caseSensitive   bool
	queryParameters map[string]string
	uuids           []string
	cloudType       string
	osName          []string
	vendorId        string
}

type TestAction struct {
	uuid         string
	actionStates []string
	actionTypes  []string
	displayName  string
}

type TestStats struct {
	uuid        string
	endDate     string
	statistics  []StatisticRequest
	displayName string
}

var EntityTests = []TestEntity{
	{},
	{},
	{},
}

var SearchTests = []TestSearch{
	// AWS VM "terraform-demo-2"
	{"terraform-demo-2", "VirtualMachine", "CLOUD", true, map[string]string{"query_type": "EXACT"}, []string{"76035434868567"}, "AWS", []string{}, ""},
	// 2 AWS VMs with "terraform-demo-1" name
	{"terraform-demo-1", "VirtualMachine", "CLOUD", true, map[string]string{}, []string{"76035434868600", "76068494507792"}, "AWS", []string{}, ""},
	// "terraform-demo-1" not found if filtered by OS = Windows
	{"terraform-demo-1", "VirtualMachine", "CLOUD", true, map[string]string{}, []string{}, "AWS", []string{"Windows"}, ""},
	// both "terraform-demo-1" found if filtered by OS = Linux
	{"terraform-demo-1", "VirtualMachine", "CLOUD", true, map[string]string{}, []string{"76035434868600", "76068494507792"}, "AWS", []string{"Linux"}, ""},
	// Azure VM "terraformDemo1" with OS = Linux
	{"terraformDemo1", "VirtualMachine", "CLOUD", true, map[string]string{}, []string{"76114164892153"}, "AZURE", []string{"Linux"}, ""},
	// GCP VM "terraform-demo-instance-1", not filtered by OS name
	{"terraform-demo-instance-1", "VirtualMachine", "CLOUD", true, map[string]string{}, []string{"76104347422523"}, "GCP", []string{}, ""},
	// GCP VM "terraform-demo-instance-1", not filtered by empty OS name
	{"terraform-demo-instance-1", "VirtualMachine", "CLOUD", true, map[string]string{}, []string{"76104347422523"}, "GCP", []string{""}, ""},
	// GCP VM "terraform-demo-instance-1", not filtered by environment type and OS name
	{"terraform-demo-instance-1", "VirtualMachine", "", true, map[string]string{}, []string{"76104347422523"}, "GCP", []string{}, ""},
	// GCP VM "terraform-demo-instance-1", not filtered by environment type, filtered by partial OS name
	{"terraform-demo-instance-1", "VirtualMachine", "", true, map[string]string{}, []string{"76104347422523"}, "GCP", []string{"Lin"}, ""},
	// GCP VM "terraform-demo-instance-1", filtered out by OS name
	{"terraform-demo-instance-1", "VirtualMachine", "", true, map[string]string{}, []string{}, "GCP", []string{"Windows"}, ""},
	// GCP VM "terraform-demo-instance-1", not filtered by environment type, cloud type and OS name
	{"terraform-demo-instance-1", "VirtualMachine", "", true, map[string]string{}, []string{"76104347422523"}, "", []string{}, ""},
	// GCP VM "terraform-demo-instance-1", not filtered by environment type and cloud type, filtered out by OS name
	{"terraform-demo-instance-1", "VirtualMachine", "", true, map[string]string{}, []string{}, "", []string{"Windows"}, ""},
	// GCP VM "terraform-demo-instance-1", filtered by cloud type and multiple OS names
	{"terraform-demo-instance-1", "VirtualMachine", "", true, map[string]string{}, []string{"76104347422523"}, "GCP", []string{"Linux", "Rhel"}, ""},
	// AWS VM search by VendorId
	{"", "VirtualMachine", "CLOUD", true, map[string]string{}, []string{"76035434868567"}, "", []string{}, "i-12345678"},
}

var ActionTests = []TestAction{
	{},
	{},
}

var StatsTests = []TestStats{
	{},
}
