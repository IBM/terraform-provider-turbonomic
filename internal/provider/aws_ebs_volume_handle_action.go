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

// AwsEbsVolumeStateAdapter adapts AwsEbsVolumeEntityModel to VolumeStateUpdater interface
type AwsEbsVolumeStateAdapter struct {
	State *AwsEbsVolumeEntityModel
}

// HandleAwsEbsVolumeCurrentState(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleAwsEbsVolumeCurrentState(ctx context.Context, state AwsEbsVolumeEntityModel, entities turboclient.SearchResults) (AwsEbsVolumeEntityModel, error) {
	state.CurrentType = types.StringValue(strings.ToLower(entities[0].Template.DisplayName))
	return state, nil
}

// HandleAwsEbsVolumeAction is the default implementation.
func HandleAwsEbsVolumeAction(ctx context.Context, resp *datasource.ReadResponse, state AwsEbsVolumeEntityModel, actions turboclient.ActionResults) (AwsEbsVolumeEntityModel, error) {
	// for each compound action, update the state
	// for now it will only pickup storage tier update in compound action and update the state
	for _, act := range actions[0].CompoundActions {
		if len(act.CurrentEntity.DisplayName) == 0 || len(act.NewEntity.DisplayName) == 0 {
			errDetail := "error while parsing action dto"
			tflog.Error(ctx, errDetail)
			resp.Diagnostics.AddError("turbonomic api error", errDetail)
			continue
		}

		state.CurrentType = types.StringValue(strings.ToLower(act.CurrentEntity.DisplayName))
		state.NewType = types.StringValue(strings.ToLower(act.NewEntity.DisplayName))
	}
	return state, nil
}

// HandleVolumeCommodityAction processes commodity actions for volume entities and updates the state with appropriate values.
// It maps statistics like StorageAccess, IOThroughput, and StorageAmount to their corresponding state fields.
//
// Parameters:
//   - ctx: The context for logging and cancellation
//   - commodityActions: The statistics response from Turbonomic containing commodity actions
//   - state: The state object to be updated (expected to be *AwsEbsVolumeEntityModel)
//
// Returns: error if any issues occur during processing
func HandleAwsEbsVolumeCommodityAction(ctx context.Context, commodityActions turboclient.StatsResponse, state interface{}) error {
	// Type assertion with proper error handling
	volumeState, ok := state.(*AwsEbsVolumeEntityModel)
	if !ok {
		errMsg := fmt.Sprintf("unexpected state type: %T, expected *AwsEbsVolumeEntityModel", state)
		tflog.Error(ctx, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Create an adapter that implements the VolumeStateUpdater interface
	adapter := &AwsEbsVolumeStateAdapter{
		State: volumeState,
	}

	// Use the generic handler
	return HandleGenericVolumeCommodityAction(ctx, commodityActions, adapter)
}

// set the current state fields with default values if they are empty
func setDefaultsAwsEbsVolumeToCurrentState(state *AwsEbsVolumeEntityModel) {
	state.CurrentType = applyDefaultIfEmptyGeneric(state.CurrentType, state.DefaultType)
	state.CurrentIops = applyDefaultIfEmptyGeneric(state.CurrentIops, state.DefaultIops)
	state.CurrentThroughput = applyDefaultIfEmptyGeneric(state.CurrentThroughput, state.DefaultThroughput)
	state.CurrentSize = applyDefaultIfEmptyGeneric(state.CurrentSize, state.DefaultSize)
}

// set the new state fields with default values if they are empty
func setDefaultsAwsEbsVolumeToNewState(state *AwsEbsVolumeEntityModel) {
	state.NewType = applyDefaultIfEmptyGeneric(state.NewType, state.DefaultType)
	state.NewIops = applyDefaultIfEmptyGeneric(state.NewIops, state.DefaultIops)
	state.NewThroughput = applyDefaultIfEmptyGeneric(state.NewThroughput, state.DefaultThroughput)
	state.NewSize = applyDefaultIfEmptyGeneric(state.NewSize, state.DefaultSize)
}

// apply current values to new values if action/stat projected value is not available
func setCurrentAwsEbsVolumeToNewState(state *AwsEbsVolumeEntityModel) {
	state.NewType = applyDefaultIfEmptyGeneric(state.NewType, state.CurrentType)
	state.NewIops = applyDefaultIfEmptyGeneric(state.NewIops, state.CurrentIops)
	state.NewThroughput = applyDefaultIfEmptyGeneric(state.NewThroughput, state.CurrentThroughput)
	state.NewSize = applyDefaultIfEmptyGeneric(state.NewSize, state.CurrentSize)
}

func (a *AwsEbsVolumeStateAdapter) UpdateIops(ctx context.Context, value float64, isNew bool) {
	// Round to nearest integer
	iopsValue := int64(math.Round(value))

	if isNew {
		a.State.NewIops = types.Int64Value(iopsValue)
		tflog.Debug(ctx, fmt.Sprintf("Setting new IOPS to %d", iopsValue))
	} else {
		a.State.CurrentIops = types.Int64Value(iopsValue)
		tflog.Debug(ctx, fmt.Sprintf("Setting current IOPS to %d", iopsValue))
	}
}

func (a *AwsEbsVolumeStateAdapter) UpdateThroughput(ctx context.Context, value float64, isNew bool) {
	throughputValue := convertKibitToMiBps(value)

	if isNew {
		a.State.NewThroughput = types.Int64Value(int64(math.Round(throughputValue)))
		tflog.Debug(ctx, fmt.Sprintf("Setting new throughput to %.2f MiB/sec", throughputValue))
	} else {
		a.State.CurrentThroughput = types.Int64Value(int64(math.Round(throughputValue)))
		tflog.Debug(ctx, fmt.Sprintf("Setting current throughput to %.2f MiB/sec", throughputValue))
	}
}

func (a *AwsEbsVolumeStateAdapter) UpdateSize(ctx context.Context, value float64, isNew bool) {
	//convert from MB to GiB
	sizeGiB := int64(convertMiBtoGiB(value))

	if isNew {
		a.State.NewSize = types.Int64Value(sizeGiB)
		tflog.Debug(ctx, fmt.Sprintf("Setting new size to %d GiB (converted from %.0f MB)", sizeGiB, value))
	} else {
		a.State.CurrentSize = types.Int64Value(sizeGiB)
		tflog.Debug(ctx, fmt.Sprintf("Setting current size to %d GiB (converted from %.0f MB)", sizeGiB, value))
	}
}

func (a *AwsEbsVolumeStateAdapter) GetEntityUuid() string {
	if a.State != nil && !a.State.EntityUuid.IsNull() {
		return a.State.EntityUuid.ValueString()
	}
	return "unknown"
}
