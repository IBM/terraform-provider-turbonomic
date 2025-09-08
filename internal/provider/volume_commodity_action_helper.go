// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	turboclient "github.com/IBM/turbonomic-go-client"
)

const (
	// Epoch constants
	EpochProjected = "PROJECTED"
	EpochCurrent   = "CURRENT"

	// Filter constants
	FilterSold    = "sold"
	StorageAccess = "StorageAccess"
	IOThroughput  = "IOThroughput"
	StorageAmount = "StorageAmount"
)

// VolumeStateUpdater is a generic interface for updating volume state
type VolumeStateUpdater interface {
	UpdateIops(ctx context.Context, value float64, isNew bool)
	UpdateThroughput(ctx context.Context, value float64, isNew bool)
	UpdateSize(ctx context.Context, value float64, isNew bool)
	GetEntityUuid() string
}

// HandleGenericVolumeCommodityAction processes commodity actions for volume entities and updates the state with appropriate values.
// It maps statistics like StorageAccess, IOThroughput, and StorageAmount to their corresponding state fields.
//
// Parameters:
//   - ctx: The context for logging and cancellation
//   - commodityActions: The statistics response from Turbonomic containing commodity actions
//   - state: The state object to be updated (must implement VolumeStateUpdater)
//
// Returns: error if any issues occur during processing
func HandleGenericVolumeCommodityAction(ctx context.Context, commodityActions turboclient.StatsResponse, state VolumeStateUpdater) error {
	// Early return with appropriate logging if no commodity actions are provided
	if len(commodityActions) == 0 {
		entityUuid := state.GetEntityUuid()
		errDetail := fmt.Sprintf("no matching commodity actions found for entity id: %s", entityUuid)
		tflog.Trace(ctx, errDetail)
		return nil
	}

	// Process all commodity actions
	for _, commodityAction := range commodityActions {
		for _, statistics := range commodityAction.Statistics {
			// Process based on statistic name
			switch statistics.Name {
			case StorageAccess:
				processStatistic(ctx, commodityAction.Epoch, statistics, state.UpdateIops)
			case IOThroughput:
				processStatistic(ctx, commodityAction.Epoch, statistics, state.UpdateThroughput)
			case StorageAmount:
				processStatistic(ctx, commodityAction.Epoch, statistics, state.UpdateSize)
			default:
				tflog.Debug(ctx, fmt.Sprintf("Ignoring unknown statistic: %s", statistics.Name))
			}
		}
	}

	return nil
}

// processStatistic processes a single statistic and updates the state using the provided updater function
func processStatistic(ctx context.Context, epoch string, statistics turboclient.Statistic, updater func(context.Context, float64, bool)) {
	// Get the raw value
	rawValue := statistics.Capacity.Avg

	switch epoch {
	case EpochProjected:
		for _, filter := range statistics.Filters {
			if filter.Value == FilterSold {
				updater(ctx, rawValue, true) // true for new value
				break
			}
		}
	case EpochCurrent:
		updater(ctx, rawValue, false) // false for current value
	}
}
