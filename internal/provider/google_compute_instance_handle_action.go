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

// HandleGoogleComputeInstanceEntityName(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleGoogleComputeInstanceCurrentState(ctx context.Context, state GoogleComputeInstanceEntityModel, entities turboclient.SearchResults) (GoogleComputeInstanceEntityModel, error) {
	state.CurrentMachineType = types.StringValue(strings.ToLower(entities[0].Template.DisplayName))
	return state, nil
}

// HandleGoogleComputeInstanceAction is the default implementation.
func HandleGoogleComputeInstanceAction(ctx context.Context, resp *datasource.ReadResponse, state GoogleComputeInstanceEntityModel, actions turboclient.ActionResults) (GoogleComputeInstanceEntityModel, error) {
	state.NewMachineType = types.StringValue(strings.ToLower(actions[0].NewEntity.DisplayName))
	return state, nil
}
