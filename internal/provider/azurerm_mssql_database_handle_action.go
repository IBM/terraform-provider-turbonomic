// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	turboclient "github.com/IBM/turbonomic-go-client"
)

// HandleAzurermMssqlDatabaseEntityName(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleAzurermMssqlDatabaseCurrentState(ctx context.Context, state AzurermMssqlDatabaseEntityModel, entities turboclient.SearchResults) (AzurermMssqlDatabaseEntityModel, error) {
	state.CurrentSkuName = types.StringValue(strings.ToLower(entities[0].Template.DisplayName))
	return state, nil
}

// HandleAzurermMssqlDatabaseAction is the default implementation.
func HandleAzurermMssqlDatabaseAction(ctx context.Context, resp *datasource.ReadResponse, state AzurermMssqlDatabaseEntityModel, actions turboclient.ActionResults) (AzurermMssqlDatabaseEntityModel, error) {
	state.NewSkuName = types.StringValue(strings.ToLower(actions[0].NewEntity.DisplayName))
	return state, nil
}
