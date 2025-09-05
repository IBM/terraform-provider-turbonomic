// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	turboclient "github.com/IBM/turbonomic-go-client"
)

var (
	google_storage_tiers_map = map[string]string{
		"Standard Persistent Disk": "pd-standard",
		"Balanced Persistent Disk": "pd-balanced",
		"SSD Persistent Disk":      "pd-ssd",
		"Extreme Persistent Disk":  "pd-extreme",
		"Hyperdisk Balanced":       "hyperdisk-balanced",
		"Hyperdisk Throughput":     "hyperdisk-throughput",
		"Hyperdisk Extreme":        "hyperdisk-extreme",
	}
)

// GoogleComputeDiskStateAdapter adapts GoogleComputeDiskEntityModel to VolumeStateUpdater interface
type GoogleComputeDiskStateAdapter struct {
	State *GoogleComputeDiskEntityModel
}

// HandleGoogleComputeDiskEntityName(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleGoogleComputeDiskCurrentState(ctx context.Context, state GoogleComputeDiskEntityModel, entities turboclient.SearchResults) (GoogleComputeDiskEntityModel, error) {
	name, err := getStorageTier(entities[0].Template.DisplayName)
	if err != nil {
		return state, fmt.Errorf("invalid storage tier found in display name: %v", entities[0].Template.DisplayName)
	}

	state.CurrentType = types.StringValue(strings.ToLower(name))
	return state, nil
}

// HandleGoogleComputeDiskAction is the default implementation.
func HandleGoogleComputeDiskAction(ctx context.Context, resp *datasource.ReadResponse, state GoogleComputeDiskEntityModel, actions turboclient.ActionResults) (GoogleComputeDiskEntityModel, error) {
	name, err := getStorageTier(actions[0].NewEntity.DisplayName)
	if err != nil {
		return state, fmt.Errorf("invalid storage tier found in display name: %v", actions[0].NewEntity.DisplayName)
	}

	state.NewType = types.StringValue(strings.ToLower(name))
	return state, nil
}

func getStorageTier(disk string) (string, error) {
	storageType, ok := google_storage_tiers_map[disk]
	if !ok {
		return "", fmt.Errorf("unknown storage type provided: %s", disk)
	}
	return storageType, nil
}

// HandleGoogleComputeDiskCommodityAction processes commodity actions for Google Compute Disk entities and updates the state with appropriate values.
// It maps statistics like StorageAccess, IOThroughput, and StorageAmount to their corresponding state fields.
//
// Parameters:
//   - ctx: The context for logging and cancellation
//   - commodityActions: The statistics response from Turbonomic containing commodity actions
//   - state: The state object to be updated (expected to be *GoogleComputeDiskEntityModel)
//
// Returns: error if any issues occur during processing
func HandleGoogleComputeDiskCommodityAction(ctx context.Context, commodityActions turboclient.StatsResponse, state interface{}) error {
	// Type assertion with proper error handling
	volumeState, ok := state.(*GoogleComputeDiskEntityModel)
	if !ok {
		errMsg := fmt.Sprintf("unexpected state type: %T, expected *GoogleComputeDiskEntityModel", state)
		tflog.Error(ctx, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Create an adapter that implements the VolumeStateUpdater interface
	adapter := &GoogleComputeDiskStateAdapter{
		State: volumeState,
	}

	// Use the generic handler
	return HandleGenericVolumeCommodityAction(ctx, commodityActions, adapter)
}

// set the current state fields with default values if they are empty
func setDefaultsGoogleComputeDiskToCurrentState(state *GoogleComputeDiskEntityModel) {
	state.CurrentType = applyDefaultIfEmptyGeneric(state.CurrentType, state.DefaultType)
	state.CurrentProvisionedIops = applyDefaultIfEmptyGeneric(state.CurrentProvisionedIops, state.DefaultProvisionedIops)
	state.CurrentProvisionedThroughput = applyDefaultIfEmptyGeneric(state.CurrentProvisionedThroughput, state.DefaultProvisionedThroughput)
	state.CurrentSize = applyDefaultIfEmptyGeneric(state.CurrentSize, state.DefaultSize)
}

// set the new state fields with default values if they are empty
func setDefaultsGoogleComputeDiskToNewState(state *GoogleComputeDiskEntityModel) {
	state.NewType = applyDefaultIfEmptyGeneric(state.NewType, state.DefaultType)
	state.NewProvisionedIops = applyDefaultIfEmptyGeneric(state.NewProvisionedIops, state.DefaultProvisionedIops)
	state.NewProvisionedThroughput = applyDefaultIfEmptyGeneric(state.NewProvisionedThroughput, state.DefaultProvisionedThroughput)
	state.NewSize = applyDefaultIfEmptyGeneric(state.NewSize, state.DefaultSize)
}

// apply current values to new values if action/stat projected value is not available
func setCurrentGoogleComputeDiskToNewState(state *GoogleComputeDiskEntityModel) {
	state.NewType = applyDefaultIfEmptyGeneric(state.NewType, state.CurrentType)
	state.NewProvisionedIops = applyDefaultIfEmptyGeneric(state.NewProvisionedIops, state.CurrentProvisionedIops)
	state.NewProvisionedThroughput = applyDefaultIfEmptyGeneric(state.NewProvisionedThroughput, state.CurrentProvisionedThroughput)
	state.NewSize = applyDefaultIfEmptyGeneric(state.NewSize, state.CurrentSize)
}
func (a *GoogleComputeDiskStateAdapter) UpdateIops(ctx context.Context, value float64, isNew bool) {
	// Round to nearest integer
	iopsValue := int64(math.Round(value))

	if isNew {
		a.State.NewProvisionedIops = types.Int64Value(iopsValue)
		tflog.Debug(ctx, fmt.Sprintf("Setting new IOPS to %.0f", value))
	} else {
		a.State.CurrentProvisionedIops = types.Int64Value(iopsValue)
		tflog.Debug(ctx, fmt.Sprintf("Setting current IOPS to %.0f", value))
	}
}

func (a *GoogleComputeDiskStateAdapter) UpdateThroughput(ctx context.Context, value float64, isNew bool) {
	throughputValue := convertKibitToMiBps(value)

	if isNew {
		a.State.NewProvisionedThroughput = types.Int64Value(int64(math.Round(throughputValue)))
		tflog.Debug(ctx, fmt.Sprintf("Setting new throughput to %.2f MiB/sec", throughputValue))
	} else {
		a.State.CurrentProvisionedThroughput = types.Int64Value(int64(math.Round(throughputValue)))
		tflog.Debug(ctx, fmt.Sprintf("Setting current throughput to %.2f MiB/sec", throughputValue))
	}
}

func (a *GoogleComputeDiskStateAdapter) UpdateSize(ctx context.Context, value float64, isNew bool) {
	//convert from MB to GiB
	sizeGiB := int64(convertMiBtoGiB(value))

	if isNew {
		a.State.NewSize = types.Int64Value(sizeGiB)
		tflog.Debug(ctx, fmt.Sprintf("Setting new size to %d GiB (converted from %.0f MiB)", sizeGiB, value))
	} else {
		a.State.CurrentSize = types.Int64Value(sizeGiB)
		tflog.Debug(ctx, fmt.Sprintf("Setting current size to %d GiB (converted from %.0f MiB)", sizeGiB, value))
	}
}

func (a *GoogleComputeDiskStateAdapter) GetEntityUuid() string {
	if a.State != nil && !a.State.EntityUuid.IsNull() {
		return a.State.EntityUuid.ValueString()
	}
	return "unknown"
}
