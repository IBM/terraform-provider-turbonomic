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

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Integration test for compound actions using the entity_actions data source
func TestEntityActionDataSourceCompoundAction(t *testing.T) {

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
			name:                   "Entity Actions Compound Actions integration test",
			testEntity:             entityActionEntityName,
			testEntityUuid:         entityActionEntityUuid,
			testEntityType:         entityActionEntityType,
			expectedEntityName:     entityActionEntityName,
			actionUuid:             entityActionActionUuid,
			action0CurrentValue:    entityActionAction0CurrentValue,
			action0NewValue:        entityActionAction0NewValue,
			action0ReasonCommodity: entityActionAction0ReasonCommodity,
			action1CurrentValue:    entityActionAction1CurrentValue,
			action1NewValue:        entityActionAction1NewValue,
			action1ReasonCommodity: entityActionAction1ReasonCommodity,
			action2CurrentValue:    entityActionAction2CurrentValue,
			action2NewValue:        entityActionAction2NewValue,
			action2ReasonCommodity: entityActionAction2ReasonCommodity,
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
