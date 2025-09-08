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

type DiskType string

const (
	StandardHDD      DiskType = "Standard HDD"
	StandardSSD      DiskType = "Standard SSD"
	PremiumSSD       DiskType = "Premium SSD"
	UltraDisk        DiskType = "Ultra SSD"
	PremiumSSDv2     DiskType = "Premium SSD v2"
	ZoneRedundantSSD DiskType = "Zone-redundant SSD"
)

var (
	diskTypeToStorageType = map[DiskType]string{
		StandardHDD:      "Standard_LRS",
		StandardSSD:      "StandardSSD_LRS",
		PremiumSSD:       "Premium_LRS",
		UltraDisk:        "UltraSSD_LRS",
		PremiumSSDv2:     "PremiumV2_LRS",
		ZoneRedundantSSD: "StandardSSD_ZRS",
	}
)

// AzurermManagedDiskStateAdapter adapts AzurermManagedDiskEntityModel to VolumeStateUpdater interface
type AzurermManagedDiskStateAdapter struct {
	State *AzurermManagedDiskEntityModel
}

// HandleAzurermManagedDiskEntityName(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleAzurermManagedDiskCurrentState(ctx context.Context, state AzurermManagedDiskEntityModel, entities turboclient.SearchResults) (AzurermManagedDiskEntityModel, error) {
	storageType, err := getStorageType(entities[0].Template.DisplayName)
	if err != nil {
		return state, fmt.Errorf("invalid storage type found in display name: %v", entities[0].Template.DisplayName)
	}

	state.CurrentStorageAccountType = types.StringValue(storageType)
	return state, nil
}

// HandleAzurermManagedDiskAction is the default implementation.
func HandleAzurermManagedDiskAction(ctx context.Context, resp *datasource.ReadResponse, state AzurermManagedDiskEntityModel, actions turboclient.ActionResults) (AzurermManagedDiskEntityModel, error) {
	name, err := getStorageType(actions[0].NewEntity.DisplayName)
	if err != nil {
		return state, fmt.Errorf("invalid storage type found in display name: %v", actions[0].NewEntity.DisplayName)
	}

	state.NewStorageAccountType = types.StringValue(name)
	return state, nil
}

func getStorageType(disk string) (string, error) {
	storageType, ok := diskTypeToStorageType[DiskType(
		strings.TrimPrefix(
			disk, "Managed ",
		),
	)]
	if !ok {
		return "", fmt.Errorf("unknown storage type provided: %s", disk)
	}
	return storageType, nil
}

// HandleVolumeCommodityAction processes commodity actions for volume entities and updates the state with appropriate values.
// It maps statistics like StorageAccess, IOThroughput, and StorageAmount to their corresponding state fields.
//
// Parameters:
//   - ctx: The context for logging and cancellation
//   - commodityActions: The statistics response from Turbonomic containing commodity actions
//   - state: The state object to be updated (expected to be *AzurermManagedDiskEntityModel)
//
// Returns: error if any issues occur during processing
func HandleAzurermManagedDiskCommodityAction(ctx context.Context, commodityActions turboclient.StatsResponse, state interface{}) error {
	// Type assertion with proper error handling
	volumeState, ok := state.(*AzurermManagedDiskEntityModel)
	if !ok {
		errMsg := fmt.Sprintf("unexpected state type: %T, expected *AzurermManagedDiskEntityModel", state)
		tflog.Error(ctx, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Create an adapter that implements the VolumeStateUpdater interface
	adapter := &AzurermManagedDiskStateAdapter{
		State: volumeState,
	}

	// Use the generic handler
	return HandleGenericVolumeCommodityAction(ctx, commodityActions, adapter)
}

// set the current state fields with default values if they are empty
func setDefaultsAzurermManagedDiskToCurrentState(state *AzurermManagedDiskEntityModel) {
	state.CurrentStorageAccountType = applyDefaultIfEmptyGeneric(state.CurrentStorageAccountType, state.DefaultStorageAccountType)
	state.CurrentDiskIopsReadWrite = applyDefaultIfEmptyGeneric(state.CurrentDiskIopsReadWrite, state.DefaultDiskIopsReadWrite)
	state.CurrentDiskMbpsReadWrite = applyDefaultIfEmptyGeneric(state.CurrentDiskMbpsReadWrite, state.DefaultDiskMbpsReadWrite)
	state.CurrentDiskSizeGb = applyDefaultIfEmptyGeneric(state.CurrentDiskSizeGb, state.DefaultDiskSizeGb)
}

// set the new state fields with default values if they are empty
func setDefaultsAzurermManagedDiskToNewState(state *AzurermManagedDiskEntityModel) {
	state.NewStorageAccountType = applyDefaultIfEmptyGeneric(state.NewStorageAccountType, state.DefaultStorageAccountType)
	state.NewDiskIopsReadWrite = applyDefaultIfEmptyGeneric(state.NewDiskIopsReadWrite, state.DefaultDiskIopsReadWrite)
	state.NewDiskMbpsReadWrite = applyDefaultIfEmptyGeneric(state.NewDiskMbpsReadWrite, state.DefaultDiskMbpsReadWrite)
	state.NewDiskSizeGb = applyDefaultIfEmptyGeneric(state.NewDiskSizeGb, state.DefaultDiskSizeGb)
}

// apply current values to new values if action/stat projected value is not available
func setCurrentAzurermManagedDiskToNewState(state *AzurermManagedDiskEntityModel) {
	state.NewStorageAccountType = applyDefaultIfEmptyGeneric(state.NewStorageAccountType, state.CurrentStorageAccountType)
	state.NewDiskIopsReadWrite = applyDefaultIfEmptyGeneric(state.NewDiskIopsReadWrite, state.CurrentDiskIopsReadWrite)
	state.NewDiskMbpsReadWrite = applyDefaultIfEmptyGeneric(state.NewDiskMbpsReadWrite, state.CurrentDiskMbpsReadWrite)
	state.NewDiskSizeGb = applyDefaultIfEmptyGeneric(state.NewDiskSizeGb, state.CurrentDiskSizeGb)
}

func (a *AzurermManagedDiskStateAdapter) UpdateIops(ctx context.Context, value float64, isNew bool) {
	// Round to nearest integer
	iopsValue := int64(math.Round(value))

	if isNew {
		a.State.NewDiskIopsReadWrite = types.Int64Value(iopsValue)
		tflog.Debug(ctx, fmt.Sprintf("Setting new IOPS to %.0f", value))
	} else {
		a.State.CurrentDiskIopsReadWrite = types.Int64Value(iopsValue)
		tflog.Debug(ctx, fmt.Sprintf("Setting current IOPS to %.0f", value))
	}
}

func (a *AzurermManagedDiskStateAdapter) UpdateThroughput(ctx context.Context, value float64, isNew bool) {
	throughputValue := convertKibitToMiBps(value)
	roundedValue := types.Int64Value(int64(math.Round(throughputValue)))

	if isNew {
		a.State.NewDiskMbpsReadWrite = roundedValue
		tflog.Debug(ctx, fmt.Sprintf("Setting new throughput to %.2f MiB/sec", throughputValue))
	} else {
		a.State.CurrentDiskMbpsReadWrite = roundedValue
		tflog.Debug(ctx, fmt.Sprintf("Setting current throughput to %.2f MiB/sec", throughputValue))
	}
}

func (a *AzurermManagedDiskStateAdapter) UpdateSize(ctx context.Context, value float64, isNew bool) {
	//convert from MB to GiB
	sizeGiB := int64(convertMiBtoGiB(value))

	if isNew {
		a.State.NewDiskSizeGb = types.Int64Value(sizeGiB)
		tflog.Debug(ctx, fmt.Sprintf("Setting new size to %d GiB (converted from %.0f MiB)", sizeGiB, value))
	} else {
		a.State.CurrentDiskSizeGb = types.Int64Value(sizeGiB)
		tflog.Debug(ctx, fmt.Sprintf("Setting current size to %d GiB (converted from %.0f MiB)", sizeGiB, value))
	}
}

func (a *AzurermManagedDiskStateAdapter) GetEntityUuid() string {
	if a.State != nil && !a.State.EntityUuid.IsNull() {
		return a.State.EntityUuid.ValueString()
	}
	return "unknown"
}
