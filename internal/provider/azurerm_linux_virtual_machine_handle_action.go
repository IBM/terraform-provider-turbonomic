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

// HandleAzurermLinuxVirtualMachineEntityName(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleAzurermLinuxVirtualMachineCurrentState(ctx context.Context, state AzurermLinuxVirtualMachineEntityModel, entities turboclient.SearchResults) (AzurermLinuxVirtualMachineEntityModel, error) {
	state.CurrentSize = types.StringValue(strings.ToLower(entities[0].Template.DisplayName))
	return state, nil
}

// HandleAzurermLinuxVirtualMachineAction is the default implementation.
func HandleAzurermLinuxVirtualMachineAction(ctx context.Context, resp *datasource.ReadResponse, state AzurermLinuxVirtualMachineEntityModel, actions turboclient.ActionResults) (AzurermLinuxVirtualMachineEntityModel, error) {
	state.NewSize = types.StringValue(strings.ToLower(actions[0].NewEntity.DisplayName))
	return state, nil
}
