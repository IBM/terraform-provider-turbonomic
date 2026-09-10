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
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
	turboclient "github.com/IBM/turbonomic-go-client"
)

// commodityKey classifies a compound action's commodity type from reasonCommodities.
// Returns one of: "cpu_request", "cpu_limit", "memory_request", "memory_limit", or "".
func commodityKey(reasonCommodities []string) string {
	if len(reasonCommodities) == 0 {
		return ""
	}
	switch reasonCommodities[0] {
	case "VCPURequest":
		return "cpu_request"
	case "VCPU", "VCPULimit":
		return "cpu_limit"
	case "VMemRequest":
		return "memory_request"
	case "VMem", "VMemLimit":
		return "memory_limit"
	}
	return ""
}

// formatCPU converts a millicores float string to a Kubernetes CPU string (e.g. "200m").
func formatCPU(valStr string) (string, error) {
	v, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return "", fmt.Errorf("parse cpu millicores %q: %w", valStr, err)
	}
	return fmt.Sprintf("%dm", int64(math.Round(v))), nil
}

// formatMemory converts a KB float string to a Kubernetes memory string (e.g. "256Mi").
func formatMemory(valStr string) (string, error) {
	v, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return "", fmt.Errorf("parse memory KB %q: %w", valStr, err)
	}
	return fmt.Sprintf("%dMi", int64(math.Round(v/1024))), nil
}

// containerResources accumulates current/new values for a single container.
type containerResources struct {
	CurrentCPURequest    string
	NewCPURequest        string
	CurrentCPULimit      string
	NewCPULimit          string
	CurrentMemoryRequest string
	NewMemoryRequest     string
	CurrentMemoryLimit   string
	NewMemoryLimit       string
}

// HandleKubernetesWorkloadResizeAction processes compoundActions from a RESIZE action
// and returns per-container resource recommendations keyed by container name.
func HandleKubernetesWorkloadResizeAction(actions turboclient.ActionResults) (map[string]*containerResources, error) {
	containers := make(map[string]*containerResources)

	for _, action := range actions {
		if action.ActionType != "RESIZE" {
			continue
		}
		for _, ca := range action.CompoundActions {
			if ca.Target.ClassName != "ContainerSpec" {
				continue
			}
			name := ca.Target.DisplayName
			if _, ok := containers[name]; !ok {
				containers[name] = &containerResources{}
			}
			cr := containers[name]

			key := commodityKey(ca.Risk.ReasonCommodities)
			if key == "" {
				continue
			}

			switch ca.ValueUnits {
			case "mCores":
				cur, err := formatCPU(ca.CurrentValue)
				if err != nil {
					return nil, err
				}
				nw, err := formatCPU(ca.NewValue)
				if err != nil {
					return nil, err
				}
				switch key {
				case "cpu_request":
					cr.CurrentCPURequest = cur
					cr.NewCPURequest = nw
				case "cpu_limit":
					cr.CurrentCPULimit = cur
					cr.NewCPULimit = nw
				}
			case "KB":
				cur, err := formatMemory(ca.CurrentValue)
				if err != nil {
					return nil, err
				}
				nw, err := formatMemory(ca.NewValue)
				if err != nil {
					return nil, err
				}
				switch key {
				case "memory_request":
					cr.CurrentMemoryRequest = cur
					cr.NewMemoryRequest = nw
				case "memory_limit":
					cr.CurrentMemoryLimit = cur
					cr.NewMemoryLimit = nw
				}
			}
		}
	}
	return containers, nil
}

// extractReplicaCountFromAspects reads controllerReplicaCount from the action Target.Aspects RawMessage.
func extractReplicaCountFromAspects(aspects json.RawMessage) (int64, bool) {
	if len(aspects) == 0 {
		return 0, false
	}
	var a struct {
		WorkloadControllerAspect struct {
			ControllerReplicaCount int64 `json:"controllerReplicaCount"`
		} `json:"workloadControllerAspect"`
	}
	if err := json.Unmarshal(aspects, &a); err != nil {
		return 0, false
	}
	return a.WorkloadControllerAspect.ControllerReplicaCount, true
}

// HandleKubernetesWorkloadScaleAction extracts replica counts from a SCALE action.
// Returns currentReplicas, newReplicas, ok.
func HandleKubernetesWorkloadScaleAction(actions turboclient.ActionResults) (int64, int64, bool) {
	for _, action := range actions {
		if action.ActionType != "SCALE" {
			continue
		}
		cur, err := strconv.ParseFloat(action.CurrentValue, 64)
		if err != nil {
			continue
		}
		nw, err := strconv.ParseFloat(action.NewValue, 64)
		if err != nil {
			continue
		}
		return int64(math.Round(cur)), int64(math.Round(nw)), true
	}
	return 0, 0, false
}

// containerToTF converts containerResources to the Terraform object type.
func containerToTF(name string, cr *containerResources) KubernetesWorkloadContainerModel {
	return KubernetesWorkloadContainerModel{
		Name:                 types.StringValue(name),
		CurrentCPURequest:    types.StringValue(cr.CurrentCPURequest),
		NewCPURequest:        types.StringValue(cr.NewCPURequest),
		CurrentCPULimit:      types.StringValue(cr.CurrentCPULimit),
		NewCPULimit:          types.StringValue(cr.NewCPULimit),
		CurrentMemoryRequest: types.StringValue(cr.CurrentMemoryRequest),
		NewMemoryRequest:     types.StringValue(cr.NewMemoryRequest),
		CurrentMemoryLimit:   types.StringValue(cr.CurrentMemoryLimit),
		NewMemoryLimit:       types.StringValue(cr.NewMemoryLimit),
	}
}
