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

// HandleAzurermWindowsVirtualMachineEntityName(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleAzurermWindowsVirtualMachineCurrentState(ctx context.Context, state AzurermWindowsVirtualMachineEntityModel, entities turboclient.SearchResults) (AzurermWindowsVirtualMachineEntityModel, error) {
	state.CurrentSize = types.StringValue(strings.ToLower(entities[0].Template.DisplayName))
	return state, nil
}

// HandleAzurermWindowsVirtualMachineAction is the default implementation.
func HandleAzurermWindowsVirtualMachineAction(ctx context.Context, resp *datasource.ReadResponse, state AzurermWindowsVirtualMachineEntityModel, actions turboclient.ActionResults) (AzurermWindowsVirtualMachineEntityModel, error) {
	state.NewSize = types.StringValue(strings.ToLower(actions[0].NewEntity.DisplayName))
	return state, nil
}
