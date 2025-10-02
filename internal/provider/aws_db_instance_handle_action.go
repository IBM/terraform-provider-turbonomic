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

// AwsDbInstanceStateAdapter adapts AwsDbInstanceEntityModel to VolumeStateUpdater interface
type AwsDbInstanceStateAdapter struct {
	State *AwsDbInstanceEntityModel
}

// HandleAwsDbInstanceCurrentState(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleAwsDbInstanceCurrentState(ctx context.Context, state AwsDbInstanceEntityModel, entities turboclient.SearchResults) (AwsDbInstanceEntityModel, error) {
	db_tiers := strings.Split(entities[0].Template.DisplayName, "-")
	if len(db_tiers) != 2 {
		return state, fmt.Errorf("unable to parse rds scale action: %v", entities[0].Template.DisplayName)
	}

	state.CurrentInstanceClass = types.StringValue(strings.ToLower(db_tiers[0]))
	state.CurrentStorageType = types.StringValue(strings.ToLower(db_tiers[1]))
	return state, nil
}

// HandleAwsDbInstanceAction is the default implementation.
func HandleAwsDbInstanceAction(ctx context.Context, resp *datasource.ReadResponse, state AwsDbInstanceEntityModel, actions turboclient.ActionResults) (AwsDbInstanceEntityModel, error) {
	for _, action := range actions[0].CompoundActions {
		if action.CurrentEntity.ClassName == "ComputeTier" {
			state.CurrentInstanceClass = types.StringValue(action.CurrentEntity.DisplayName)
			state.NewInstanceClass = types.StringValue(action.NewEntity.DisplayName)
		} else if action.CurrentEntity.ClassName == "StorageTier" {
			state.CurrentStorageType = types.StringValue(action.CurrentEntity.DisplayName)
			state.NewStorageType = types.StringValue(action.NewEntity.DisplayName)
		}
	}
	return state, nil
}

// HandleVolumeCommodityAction processes commodity actions for volume entities and updates the state with appropriate values.
// It maps statistics like StorageAccess, and StorageAmount to their corresponding state fields.
//
// Parameters:
//   - ctx: The context for logging and cancellation
//   - commodityActions: The statistics response from Turbonomic containing commodity actions
//   - state: The state object to be updated (expected to be *AwsDbInstanceEntityModel)
//
// Returns: error if any issues occur during processing
func HandleAwsDbInstanceCommodityAction(ctx context.Context, commodityActions turboclient.StatsResponse, state interface{}) error {
	// Type assertion with proper error handling
	volumeState, ok := state.(*AwsDbInstanceEntityModel)
	if !ok {
		errMsg := fmt.Sprintf("unexpected state type: %T, expected *AwsDbInstanceEntityModel", state)
		tflog.Error(ctx, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Create an adapter that implements the VolumeStateUpdater interface
	adapter := &AwsDbInstanceStateAdapter{
		State: volumeState,
	}

	// Use the generic handler
	return HandleGenericVolumeCommodityAction(ctx, commodityActions, adapter)
}

func (a *AwsDbInstanceStateAdapter) UpdateIops(ctx context.Context, value float64, isNew bool) {
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

func (a *AwsDbInstanceStateAdapter) UpdateThroughput(ctx context.Context, value float64, isNew bool) {
	// AWS RDS doesn't use throughput
	tflog.Debug(ctx, fmt.Sprintf("Throughput update not applicable for AWS RDS: %f", value))
}

func (a *AwsDbInstanceStateAdapter) UpdateSize(ctx context.Context, value float64, isNew bool) {
	//convert from MB to GiB
	sizeGiB := int64(convertMiBtoGiB(value))

	if isNew {
		a.State.NewAllocatedStorage = types.Int64Value(sizeGiB)
		tflog.Debug(ctx, fmt.Sprintf("Setting new allocated storage to %d GiB (converted from %.0f MB)", sizeGiB, value))
	} else {
		a.State.CurrentAllocatedStorage = types.Int64Value(sizeGiB)
		tflog.Debug(ctx, fmt.Sprintf("Setting current allocated storage to %d GiB (converted from %.0f MB)", sizeGiB, value))
	}
}

func (a *AwsDbInstanceStateAdapter) GetEntityUuid() string {
	if a.State != nil && !a.State.EntityUuid.IsNull() {
		return a.State.EntityUuid.ValueString()
	}
	return "unknown"
}
