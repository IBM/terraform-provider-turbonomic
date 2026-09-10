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

// nodePoolActionDelta returns the net node count change implied by a set of
// PROVISION/SUSPEND actions for a single node VM. Each PROVISION means +1;
// each SUSPEND means -1. SCALE actions are ignored (they are right-sizing, not
// node count changes).
func nodePoolActionDelta(actions turboclient.ActionResults) int {
	delta := 0
	for _, a := range actions {
		switch a.ActionType {
		case "PROVISION":
			delta++
		case "SUSPEND":
			delta--
		}
	}
	return delta
}

// firstActionMeta returns the actionType, actionState and actionMode from the
// first PROVISION or SUSPEND action found, or empty strings if none exist.
func firstActionMeta(actions turboclient.ActionResults) (actionType, actionState, actionMode string) {
	for _, a := range actions {
		if a.ActionType == "PROVISION" || a.ActionType == "SUSPEND" {
			return a.ActionType, a.ActionState, a.ActionMode
		}
	}
	return "", "", ""
}

// filterVMsByTag returns the subset of entities whose Tags map contains the
// given key with the given value as one of its values.
func filterVMsByTag(entities turboclient.SearchResults, tagKey, tagValue string) turboclient.SearchResults {
	var out turboclient.SearchResults
	for _, e := range entities {
		if vals, ok := e.Tags[tagKey]; ok {
			for _, v := range vals {
				if v == tagValue {
					out = append(out, e)
					break
				}
			}
		}
	}
	return out
}
