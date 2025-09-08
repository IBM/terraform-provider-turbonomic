// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	turboclient "github.com/IBM/turbonomic-go-client"
)

// HandleAwsDbInstanceEntityName(ctx, entity) is a custom function to extract the display name for setting it to current types.
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
