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

// HandleAwsInstanceEntityName(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleAwsInstanceCurrentState(ctx context.Context, state AwsInstanceEntityModel, entities turboclient.SearchResults) (AwsInstanceEntityModel, error) {
	state.CurrentInstanceType = types.StringValue(strings.ToLower(entities[0].Template.DisplayName))
	return state, nil
}

// HandleAwsInstanceAction is the default implementation.
func HandleAwsInstanceAction(ctx context.Context, resp *datasource.ReadResponse, state AwsInstanceEntityModel, actions turboclient.ActionResults) (AwsInstanceEntityModel, error) {
	state.NewInstanceType = types.StringValue(strings.ToLower(actions[0].NewEntity.DisplayName))
	return state, nil
}
