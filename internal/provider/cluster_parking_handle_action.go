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
	turboclient "github.com/IBM/turbonomic-go-client"
)

// parkingNewState infers the recommended state from the current parking state and
// whether a smart parking schedule recommendation is active.
//
// Logic:
//   - If state == "RUNNING" and a smart schedule is active → recommend "STOPPED" (park it).
//   - If state == "STOPPED"  and a smart schedule is active → recommend "RUNNING" (unpark it).
//   - Otherwise, the recommendation matches the current state (no change).
func parkingNewState(currentState string, rec map[string]interface{}) string {
	if len(rec) == 0 {
		return currentState
	}
	switch currentState {
	case "RUNNING":
		return "STOPPED"
	case "STOPPED":
		return "RUNNING"
	default:
		return currentState
	}
}

// parkingSearchCluster searches for a ContainerPlatformCluster by name and
// returns its UUID, or "" if not found. Uses the containerPlatformClustersByName
// filter (already in the go-client entityNameMap).
func parkingSearchCluster(client turboclient.T8cClient, name string) (string, string) {
	entities, errDiag := GetEntitiesByName(client,
		WithEntityName(name),
		WithEntityType("ContainerPlatformCluster"),
	)
	if errDiag != nil {
		return "", errDiag.Detail()
	}
	if len(entities) == 0 {
		return "", ""
	}
	return entities[0].UUID, ""
}
