// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	entityActionSearchResponseEmpty = "empty_array_response.json"
	entityActionActionResponseEmpty = "empty_array_response.json"
	nonExistingEntity               = "NonExistingEntity"
	entityTypeContainer             = "Container"
	entityActionSearchResponseHost  = "entity_action_search_host_success.json"
	entityActionPMName              = "esx71.example.com"
	entityTypePM                    = "PhysicalMachine"
	entityTypePMUuid                = "75878937361513"
	actionTypesFilterProvision      = "PROVISION"

	entityActionSearchResponseVM    = "entity_action_search_VM_resize.json"
	entityActionMultiActionResponse = "entity_action_multiple_actions.json"
	multiActionEntity               = "resize_test_pc24"
	multiActionEntityType           = "VirtualMachine"
	multiActionEntityUuid           = "75930461864800"
	multiAction0Uuid                = "638883006725506"
	multiAction0CurrentValue        = "2621440.0"
	multiAction0NewValue            = "3670016.0"
	multiAction1Uuid                = "638929431495649"
	multiAction1CurrentValue        = "1.0"
	multiAction1NewValue            = "2.0"

	entityActionSearchResponseWC       = "entity_action_search_wc_success.json"
	entityActionCompoundActionResponse = "entity_action_compound_actions.json"
	compoundActionEntity               = "licensing-service-instance"
	compoundActionEntityType           = "WorkloadController"
	compoundActionEntityUuid           = "76084922964120"
	compoundActionUuid                 = "639054921252942"
	compoundAction0CurrentValue        = "262144.0"
	compoundAction0NewValue            = "491520.0"
	compoundAction0ReasonCommodity     = "VMemRequest"
	compoundAction1CurrentValue        = "1048576.0"
	compoundAction1NewValue            = "1310720.0"
	compoundAction1ReasonCommodity     = "VMem"
	compoundAction2CurrentValue        = "200.0"
	compoundAction2NewValue            = "10.0"
	compoundAction2ReasonCommodity     = "VCPURequest"

	entityActionMoveSearchResponse = "entity_action_search_VM_move.json"
	entityActionMoveActionResponse = "entity_action_action_move.json"
	moveActionEntity               = "hv16-susevm1"
	moveActionEntityType           = "VirtualMachine"
	moveActionEntityUuid           = "75878938189618"
	moveActionUuid                 = "638870531859700"
	moveActionCurrentValue         = "75878938189574"
	moveActionNewValue             = "75878938189577"

	entityActionReconfigureSearchResponse = "entity_action_seach_databaseServer_reconfigure.json"
	entityActionReconfigureActionResponse = "entity_action_action_reconfigure.json"
	reconfigureActionEntity               = "hv16-susevm1"
	reconfigureActionEntityType           = "DatabaseServer"
	reconfigureActionEntityUuid           = "76105706434662"
	reconfigureActionUuid                 = "639055660460449"
	reconfigureActionDetails              = "Reconfigure Database Server MySQL MySQL db to provide VMem, VCPU"
	reconfigureActionReasonCommodities0   = "VMem"
	reconfigureActionReasonCommodities1   = "VCPU"
	reconfigureActionDiscoveredByType     = "Dynatrace"
)

// Test entity action data source where the entity does not exist
func TestEntityActionDataSourceNoInstance(t *testing.T) {

	mockServer := createLocalServer(t,
		loadTestFile(t, entityActionSearchResponseEmpty),
		loadTestFile(t, entityActionActionResponseEmpty),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name               string
		testEntity         string
		expectedEntityName string
		testEntityType     string
	}{
		{
			name:               "Non existing Container",
			testEntity:         nonExistingEntity,
			expectedEntityName: nonExistingEntity,
			testEntityType:     entityTypeContainer,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_entity_actions" "test" {
							    entity_name  = "` + tc.testEntity + `"
                                entity_type = "` + tc.testEntityType + `"
						    }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckNoResourceAttr("data.turbonomic_entity_actions.test", "entity_type"),
							resource.TestCheckNoResourceAttr("data.turbonomic_entity_actions.test", "action_types"),
							resource.TestCheckNoResourceAttr("data.turbonomic_entity_actions.test", "actions"),
							resource.TestCheckNoResourceAttr("data.turbonomic_entity_actions.test", "entity_uuid"),
							resource.TestCheckNoResourceAttr("data.turbonomic_entity_actions.test", "environment_type"),
							resource.TestCheckNoResourceAttr("data.turbonomic_entity_actions.test", "states"),
						),
					},
				},
			})
		})
	}
}

// Test entity action data source where the entity exists but has no actions with action filter
func TestEntityActionDataSourceNoAction(t *testing.T) {

	mockServer := createLocalServer(t,
		loadTestFile(t, entityActionDir, entityActionSearchResponseHost),
		loadTestFile(t, entityActionActionResponseEmpty),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name               string
		testEntity         string
		testEntityUuid     string
		expectedEntityName string
		testEntityType     string
		actionTypes        string
	}{
		{
			name:               "No action host",
			testEntity:         entityActionPMName,
			testEntityUuid:     entityTypePMUuid,
			expectedEntityName: entityActionPMName,
			testEntityType:     entityTypePM,
			actionTypes:        actionTypesFilterProvision,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_entity_actions" "test" {
							    entity_name  = "` + tc.testEntity + `"
                                entity_type  = "` + tc.testEntityType + `"
                                action_types = ["` + tc.actionTypes + `"]

						    }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_type", tc.testEntityType),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "action_types.0", tc.actionTypes),
							resource.TestCheckNoResourceAttr("data.turbonomic_entity_actions.test", "actions"),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_uuid", tc.testEntityUuid),
							resource.TestCheckNoResourceAttr("data.turbonomic_entity_actions.test", "environment_type"),
							resource.TestCheckNoResourceAttr("data.turbonomic_entity_actions.test", "states"),
						),
					},
				},
			})
		})
	}
}

// Test entity action data source where the entity exists and has multiple resize actions
func TestEntityActionDataSourceMultiActions(t *testing.T) {

	mockServer := createLocalServer(t,
		loadTestFile(t, entityActionDir, entityActionSearchResponseVM),
		loadTestFile(t, entityActionDir, entityActionMultiActionResponse),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                string
		testEntity          string
		testEntityUuid      string
		testEntityType      string
		expectedEntityName  string
		action0Uuid         string
		action0CurrentValue string
		action0NewValue     string
		action1Uuid         string
		action1CurrentValue string
		action1NewValue     string
	}{
		{
			name:                "VM Multi Actions",
			testEntity:          multiActionEntity,
			testEntityUuid:      multiActionEntityUuid,
			testEntityType:      multiActionEntityType,
			expectedEntityName:  multiActionEntity,
			action0Uuid:         multiAction0Uuid,
			action0CurrentValue: multiAction0CurrentValue,
			action0NewValue:     multiAction0NewValue,
			action1Uuid:         multiAction1Uuid,
			action1CurrentValue: multiAction1CurrentValue,
			action1NewValue:     multiAction1NewValue,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_entity_actions" "test" {
							    entity_name  = "` + tc.testEntity + `"
                                entity_type = "` + tc.testEntityType + `"
						    }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_type", tc.testEntityType),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_uuid", tc.testEntityUuid),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.uuid", tc.action0Uuid),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.current_value", tc.action0CurrentValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.new_value", tc.action0NewValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.1.uuid", tc.action1Uuid),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.1.current_value", tc.action1CurrentValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.1.new_value", tc.action1NewValue),
						),
					},
				},
			})
		})
	}
}

// Test entity action data source where the entity exists and has compound actions
func TestEntityActionDataSourceCompoundAction(t *testing.T) {

	mockServer := createLocalServer(t,
		loadTestFile(t, entityActionDir, entityActionSearchResponseWC),
		loadTestFile(t, entityActionDir, entityActionCompoundActionResponse),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name                   string
		testEntity             string
		testEntityUuid         string
		testEntityType         string
		expectedEntityName     string
		actionUuid             string
		action0CurrentValue    string
		action0NewValue        string
		action0ReasonCommodity string
		action1CurrentValue    string
		action1NewValue        string
		action1ReasonCommodity string
		action2CurrentValue    string
		action2NewValue        string
		action2ReasonCommodity string
	}{
		{
			name:                   "Workload Controller Compound Actions",
			testEntity:             compoundActionEntity,
			testEntityUuid:         compoundActionEntityUuid,
			testEntityType:         compoundActionEntityType,
			expectedEntityName:     compoundActionEntity,
			actionUuid:             compoundActionUuid,
			action0CurrentValue:    compoundAction0CurrentValue,
			action0NewValue:        compoundAction0NewValue,
			action0ReasonCommodity: compoundAction0ReasonCommodity,
			action1CurrentValue:    compoundAction1CurrentValue,
			action1NewValue:        compoundAction1NewValue,
			action1ReasonCommodity: compoundAction1ReasonCommodity,
			action2CurrentValue:    compoundAction2CurrentValue,
			action2NewValue:        compoundAction2NewValue,
			action2ReasonCommodity: compoundAction2ReasonCommodity,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_entity_actions" "test" {
							    entity_name  = "` + tc.testEntity + `"
                                entity_type = "` + tc.testEntityType + `"
						    }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_type", tc.testEntityType),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_uuid", tc.testEntityUuid),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.uuid", tc.actionUuid),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.compound_actions.0.current_value", tc.action0CurrentValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.compound_actions.0.new_value", tc.action0NewValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.compound_actions.0.risk.reason_commodities.0", tc.action0ReasonCommodity),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.compound_actions.1.current_value", tc.action1CurrentValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.compound_actions.1.new_value", tc.action1NewValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.compound_actions.1.risk.reason_commodities.0", tc.action1ReasonCommodity),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.compound_actions.2.current_value", tc.action2CurrentValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.compound_actions.2.new_value", tc.action2NewValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.compound_actions.2.risk.reason_commodities.0", tc.action2ReasonCommodity),
						),
					},
				},
			})
		})
	}
}

// Test entity action data source where the entity exists and has a move action
func TestEntityActionDataSourceMoveAction(t *testing.T) {

	mockServer := createLocalServer(t,
		loadTestFile(t, entityActionDir, entityActionMoveSearchResponse),
		loadTestFile(t, entityActionDir, entityActionMoveActionResponse),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name               string
		testEntity         string
		testEntityUuid     string
		testEntityType     string
		expectedEntityName string
		actionUuid         string
		currentValue       string
		newValue           string
	}{
		{
			name:               "Test VirtualMachine move Actions",
			testEntity:         moveActionEntity,
			testEntityUuid:     moveActionEntityUuid,
			testEntityType:     moveActionEntityType,
			expectedEntityName: moveActionEntity,
			actionUuid:         moveActionUuid,
			currentValue:       moveActionCurrentValue,
			newValue:           moveActionNewValue,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_entity_actions" "test" {
							    entity_name  = "` + tc.testEntity + `"
                                entity_type = "` + tc.testEntityType + `"
						    }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_type", tc.testEntityType),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_uuid", tc.testEntityUuid),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.uuid", tc.actionUuid),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.current_value", tc.currentValue),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.new_value", tc.newValue),
						),
					},
				},
			})
		})
	}
}

// Test entity action data source where the entity exists and has a reconfigure action
func TestEntityActionDataSourceReconfigureAction(t *testing.T) {

	mockServer := createLocalServer(t,
		loadTestFile(t, entityActionDir, entityActionReconfigureSearchResponse),
		loadTestFile(t, entityActionDir, entityActionReconfigureActionResponse),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagsRespTestData),
		loadTestFile(t, entityTagTestDataBaseDir, entityTagRespTestData))
	defer mockServer.Close()

	providerConfig := fmt.Sprintf(config, strings.TrimLeft(mockServer.URL, "htps:/"))

	for _, tc := range []struct {
		name               string
		testEntity         string
		testEntityUuid     string
		testEntityType     string
		expectedEntityName string
		actionUuid         string
		actionDetails      string
		reasonCommodity0   string
		reasonCommodity1   string
		discoveredByType   string
	}{
		{
			name:               "Test DatabaseServer reconfigure action ",
			testEntity:         reconfigureActionEntity,
			testEntityUuid:     reconfigureActionEntityUuid,
			testEntityType:     reconfigureActionEntityType,
			expectedEntityName: reconfigureActionEntity,
			actionUuid:         reconfigureActionUuid,
			actionDetails:      reconfigureActionDetails,
			reasonCommodity0:   reconfigureActionReasonCommodities0,
			reasonCommodity1:   reconfigureActionReasonCommodities1,
			discoveredByType:   reconfigureActionDiscoveredByType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerConfig +
							`data "turbonomic_entity_actions" "test" {
							    entity_name  = "` + tc.testEntity + `"
                                entity_type = "` + tc.testEntityType + `"
						    }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_name", tc.expectedEntityName),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_type", tc.testEntityType),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "entity_uuid", tc.testEntityUuid),

							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.uuid", tc.actionUuid),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.details", tc.actionDetails),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.risk.reason_commodities.0", tc.reasonCommodity0),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.risk.reason_commodities.1", tc.reasonCommodity1),
							resource.TestCheckResourceAttr("data.turbonomic_entity_actions.test", "actions.0.target.discovered_by.type", tc.discoveredByType),
						),
					},
				},
			})
		})
	}
}
