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

// HandleCloudEntityRecommendationCurrentState(ctx, entity) is a custom function to extract the display name for setting it to current types.
func HandleCloudEntityRecommendationCurrentState(ctx context.Context, state CloudEntityRecommendationEntityModel, entities turboclient.SearchResults) (CloudEntityRecommendationEntityModel, error) {
	state.CurrentSize = types.StringValue(strings.ToLower(entities[0].Template.DisplayName))
	return state, nil
}

// HandleCloudEntityRecommendationAction is the default implementation.
func HandleCloudEntityRecommendationAction(ctx context.Context, resp *datasource.ReadResponse, state CloudEntityRecommendationEntityModel, actions turboclient.ActionResults) (CloudEntityRecommendationEntityModel, error) {
	state.NewSize = types.StringValue(strings.ToLower(actions[0].NewEntity.DisplayName))
	return state, nil
}
