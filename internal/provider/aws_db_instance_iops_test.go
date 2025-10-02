// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	turboclient "github.com/IBM/turbonomic-go-client"
)

func TestErrorHandlingInCommodityAction(t *testing.T) {
	// Create context for testing
	ctx := context.Background()

	// Test with nil state
	err := HandleAwsDbInstanceCommodityAction(ctx, turboclient.StatsResponse{}, nil)
	assert.Error(t, err)

	// Test with wrong state type
	wrongState := &struct{}{}
	err = HandleAwsDbInstanceCommodityAction(ctx, turboclient.StatsResponse{}, wrongState)
	assert.Error(t, err)

	// Test with empty stats response
	state := &AwsDbInstanceEntityModel{}
	err = HandleAwsDbInstanceCommodityAction(ctx, turboclient.StatsResponse{}, state)
	assert.NoError(t, err) // Should not error, just not update anything
}

func TestDefaultValueApplication(t *testing.T) {
	testCases := []struct {
		name                        string
		currentInstanceClass        types.String
		newInstanceClass            types.String
		defaultInstanceClass        types.String
		currentStorageType          types.String
		newStorageType              types.String
		defaultStorageType          types.String
		currentIops                 types.Int64
		newIops                     types.Int64
		defaultIops                 types.Int64
		currentAllocatedStorage     types.Int64
		newAllocatedStorage         types.Int64
		defaultAllocatedStorage     types.Int64
		expectedNewInstanceClass    types.String
		expectedNewStorageType      types.String
		expectedNewIops             types.Int64
		expectedNewAllocatedStorage types.Int64
	}{
		{
			name:                        "All values present",
			currentInstanceClass:        types.StringValue("db.t3.micro"),
			newInstanceClass:            types.StringValue("db.t3.small"),
			defaultInstanceClass:        types.StringValue("db.t3.medium"),
			currentStorageType:          types.StringValue("gp2"),
			newStorageType:              types.StringValue("gp3"),
			defaultStorageType:          types.StringValue("io1"),
			currentIops:                 types.Int64Value(1000),
			newIops:                     types.Int64Value(3000),
			defaultIops:                 types.Int64Value(5000),
			currentAllocatedStorage:     types.Int64Value(20),
			newAllocatedStorage:         types.Int64Value(50),
			defaultAllocatedStorage:     types.Int64Value(100),
			expectedNewInstanceClass:    types.StringValue("db.t3.small"),
			expectedNewStorageType:      types.StringValue("gp3"),
			expectedNewIops:             types.Int64Value(3000),
			expectedNewAllocatedStorage: types.Int64Value(50),
		},
		{
			name:                        "Missing new values",
			currentInstanceClass:        types.StringValue("db.t3.micro"),
			newInstanceClass:            types.StringNull(),
			defaultInstanceClass:        types.StringValue("db.t3.medium"),
			currentStorageType:          types.StringValue("gp2"),
			newStorageType:              types.StringNull(),
			defaultStorageType:          types.StringValue("io1"),
			currentIops:                 types.Int64Value(1000),
			newIops:                     types.Int64Null(),
			defaultIops:                 types.Int64Value(5000),
			currentAllocatedStorage:     types.Int64Value(20),
			newAllocatedStorage:         types.Int64Null(),
			defaultAllocatedStorage:     types.Int64Value(100),
			expectedNewInstanceClass:    types.StringValue("db.t3.medium"),
			expectedNewStorageType:      types.StringValue("io1"),
			expectedNewIops:             types.Int64Value(5000),
			expectedNewAllocatedStorage: types.Int64Value(100),
		},
		{
			name:                        "Missing current and new values",
			currentInstanceClass:        types.StringNull(),
			newInstanceClass:            types.StringNull(),
			defaultInstanceClass:        types.StringValue("db.t3.medium"),
			currentStorageType:          types.StringNull(),
			newStorageType:              types.StringNull(),
			defaultStorageType:          types.StringValue("io1"),
			currentIops:                 types.Int64Null(),
			newIops:                     types.Int64Null(),
			defaultIops:                 types.Int64Value(5000),
			currentAllocatedStorage:     types.Int64Null(),
			newAllocatedStorage:         types.Int64Null(),
			defaultAllocatedStorage:     types.Int64Value(100),
			expectedNewInstanceClass:    types.StringValue("db.t3.medium"),
			expectedNewStorageType:      types.StringValue("io1"),
			expectedNewIops:             types.Int64Value(5000),
			expectedNewAllocatedStorage: types.Int64Value(100),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create state with test values
			state := &AwsDbInstanceEntityModel{
				CurrentInstanceClass:    tc.currentInstanceClass,
				NewInstanceClass:        tc.newInstanceClass,
				DefaultInstanceClass:    tc.defaultInstanceClass,
				CurrentStorageType:      tc.currentStorageType,
				NewStorageType:          tc.newStorageType,
				DefaultStorageType:      tc.defaultStorageType,
				CurrentIops:             tc.currentIops,
				NewIops:                 tc.newIops,
				DefaultIops:             tc.defaultIops,
				CurrentAllocatedStorage: tc.currentAllocatedStorage,
				NewAllocatedStorage:     tc.newAllocatedStorage,
				DefaultAllocatedStorage: tc.defaultAllocatedStorage,
			}

			// Apply defaults
			setDefaultsAwsDbInstanceToNewState(state)

			// Check if defaults were applied correctly
			assert.Equal(t, tc.expectedNewInstanceClass.ValueString(), state.NewInstanceClass.ValueString())
			assert.Equal(t, tc.expectedNewStorageType.ValueString(), state.NewStorageType.ValueString())
			assert.Equal(t, tc.expectedNewIops.ValueInt64(), state.NewIops.ValueInt64())
			assert.Equal(t, tc.expectedNewAllocatedStorage.ValueInt64(), state.NewAllocatedStorage.ValueInt64())
		})
	}
}

func TestNullValueHandling(t *testing.T) {
	// Create state with null values
	state := &AwsDbInstanceEntityModel{
		CurrentInstanceClass:    types.StringNull(),
		NewInstanceClass:        types.StringNull(),
		DefaultInstanceClass:    types.StringValue("db.t3.medium"),
		CurrentStorageType:      types.StringNull(),
		NewStorageType:          types.StringNull(),
		DefaultStorageType:      types.StringValue("gp2"),
		CurrentIops:             types.Int64Null(),
		NewIops:                 types.Int64Null(),
		DefaultIops:             types.Int64Value(3000),
		CurrentAllocatedStorage: types.Int64Null(),
		NewAllocatedStorage:     types.Int64Null(),
		DefaultAllocatedStorage: types.Int64Value(50),
	}

	// Test setting defaults for current state
	setDefaultsAwsDbInstanceToCurrentState(state)

	// Check if defaults were applied correctly
	assert.Equal(t, "db.t3.medium", state.CurrentInstanceClass.ValueString())
	assert.Equal(t, "gp2", state.CurrentStorageType.ValueString())
	assert.Equal(t, int64(3000), state.CurrentIops.ValueInt64())
	assert.Equal(t, int64(50), state.CurrentAllocatedStorage.ValueInt64())

	// Reset new values to null
	state.NewInstanceClass = types.StringNull()
	state.NewStorageType = types.StringNull()
	state.NewIops = types.Int64Null()
	state.NewAllocatedStorage = types.Int64Null()

	// Test setting current values to new state
	setCurrentAwsDbInstanceToNewState(state)

	// Check if current values were applied to new state
	assert.Equal(t, "db.t3.medium", state.NewInstanceClass.ValueString())
	assert.Equal(t, "gp2", state.NewStorageType.ValueString())
	assert.Equal(t, int64(3000), state.NewIops.ValueInt64())
	assert.Equal(t, int64(50), state.NewAllocatedStorage.ValueInt64())
}
