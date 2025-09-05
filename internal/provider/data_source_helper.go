// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	turboclient "github.com/IBM/turbonomic-go-client"
)

// Constants for entity types and filter types
const (
	VirtualVolumeEntityType = "VirtualVolume"
	RelationFilterType      = "relation"
)

/*
fetches turbonomic entities by given entity-name and entity-type and environment type. If there are multiple matches or no matches it returns error diagnostic object.
returns zero or one matching entity or throws an error
*/
func GetEntitiesByNameAndType(client turboclient.T8cClient, entityName, entityType, envType, cloudType string) (turboclient.SearchResults, *diag.ErrorDiagnostic) {
	if client == nil {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Internal error occurred while fetching entities")
		return nil, &errDiag
	}
	if len(entityName) == 0 {
		errDiag := diag.NewErrorDiagnostic("Invalid entity name specified", fmt.Sprintf("Received invalid name: %s",
			entityName))
		return nil, &errDiag
	}

	searchReq := turboclient.SearchRequest{
		Name:             entityName,
		CaseSensitive:    true,
		SearchParameters: map[string]string{"query_type": "EXACT"},
	}

	if len(entityType) > 0 {
		searchReq.EntityType = entityType
	}
	if len(envType) > 0 {
		searchReq.EnvironmentType = envType
	}
	if len(cloudType) > 0 {
		searchReq.CloudType = cloudType
	}

	entity, err := client.SearchEntityByName(searchReq)
	if err != nil {
		errDiag := diag.NewErrorDiagnostic("Unable to search Turbonomic", err.Error())
		return nil, &errDiag
	} else if len(entity) > 1 {
		errDiag := diag.NewErrorDiagnostic("Multiple Entities with provided name found", fmt.Sprintf("Multiple Entities with the name %s of type %s found in Turbonomic instance",
			entityName,
			entityType))
		return nil, &errDiag
	}

	return entity, nil
}

/*
fetches ready actions by a entity uuid and action type. If there are multiple actions, it returns error.
returns zero or one matching action or throws an error
*/
func GetActionsByEntityUUIDAndType(client turboclient.T8cClient, entityUuid, actionType string) (turboclient.ActionResults, *diag.ErrorDiagnostic) {
	if client == nil {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Internal error occurred while fetching action")
		return nil, &errDiag
	}
	if len(entityUuid) == 0 || len(actionType) == 0 {
		errDiag := diag.NewErrorDiagnostic("Invalid entity name or action specified", fmt.Sprintf("Received invalid action uuid: %s / type: %s",
			entityUuid,
			actionType))
		return nil, &errDiag
	}

	actions, err := client.GetActionsByUUID(turboclient.ActionsRequest{
		Uuid:        entityUuid,
		ActionState: []string{"READY"},
		ActionType:  []string{actionType},
	})
	if err != nil {
		errDiag := diag.NewErrorDiagnostic("Unable to retrieve actions from Turbonomic", err.Error())
		return nil, &errDiag
	} else if len(actions) > 1 {
		errDiag := diag.NewErrorDiagnostic("Multiple Entities with provided name found", fmt.Sprintf("Action with uuid: %s / type: %s returned more than one result",
			entityUuid,
			actionType))
		return nil, &errDiag
	}

	return actions, nil
}

/*
fetches actions by a entity uuid, action type and action state.
returns zero, one or more matching actions or throws an error
*/
func GetFilteredEntityActions(client turboclient.T8cClient, entityUuid string, actionType []string, actionState []string) (turboclient.ActionResults, *diag.ErrorDiagnostic) {
	if client == nil {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Internal error occurred while fetching action")
		return nil, &errDiag
	}
	if len(entityUuid) == 0 {
		errDiag := diag.NewErrorDiagnostic("Invalid entity name specified", fmt.Sprintf("Received invalid action uuid: %s / type: %s / state: %s",
			entityUuid,
			actionType,
			actionState))
		return nil, &errDiag
	}

	actions, err := client.GetActionsByUUID(turboclient.ActionsRequest{
		Uuid:        entityUuid,
		ActionState: actionState,
		ActionType:  actionType,
	})
	if err != nil {
		errDiag := diag.NewErrorDiagnostic("Unable to retrieve actions from Turbonomic", err.Error())
		return nil, &errDiag
	}

	return actions, nil
}

// CreateStatisticRequest creates a statistic request with the given parameters
// It provides a clean way to create statistic requests with consistent structure
//
// Parameters:
//   - name: The name of the statistic (e.g., StorageAccess, IOThroughput, StorageAmount)
//   - entityType: The type of entity (e.g., VirtualVolumeEntityType)
//   - filterType: The type of filter (e.g., RelationFilterType)
//   - filterValue: The value of the filter (e.g., FilterSold)
//
// Returns:
//   - turboclient.StatisticRequest: A properly configured statistic request
func CreateStatisticRequest(name string, entityType string, filterType string, filterValue string) turboclient.StatisticRequest {
	return turboclient.StatisticRequest{
		Name:              name,
		RelatedEntityType: entityType,
		Filters: []turboclient.Filter{
			{
				Type:  filterType,
				Value: filterValue,
			},
		},
	}
}

// CreateStatisticRequests creates multiple statistic requests with the same entity type and filter
// This is useful when you need to create multiple statistics for the same entity type
//
// Parameters:
//   - names: The names of the statistics (e.g., StorageAccess, IOThroughput, StorageAmount)
//   - entityType: The type of entity (e.g., VirtualVolumeEntityType)
//   - filterType: The type of filter (e.g., RelationFilterType)
//   - filterValue: The value of the filter (e.g., FilterSold)
//
// Returns:
//   - []turboclient.StatisticRequest: A slice of properly configured statistic requests
func CreateStatisticRequests(names []string, entityType string, filterType string, filterValue string) []turboclient.StatisticRequest {
	requests := make([]turboclient.StatisticRequest, len(names))
	for i, name := range names {
		requests[i] = CreateStatisticRequest(name, entityType, filterType, filterValue)
	}
	return requests
}

// DefaultStatistics returns the default set of statistics to retrieve for virtual volumes
func DefaultStatistics() []turboclient.StatisticRequest {
	// Use the constants defined in volume_commodity_action_helper.go
	statNames := []string{StorageAccess, StorageAmount, IOThroughput}
	return CreateStatisticRequests(statNames, VirtualVolumeEntityType, RelationFilterType, FilterSold)
}

/*
GetStatsByEntityUUID fetches stats for a given entity UUID with default statistics.
It returns stats results or an error diagnostic.

Parameters:
  - client: The Turbonomic client to use for API calls
  - entityUuid: The UUID of the entity to fetch stats for

Returns:
  - turboclient.StatsResponse: The stats response from the API
  - *diag.ErrorDiagnostic: An error diagnostic if the operation fails
*/
func GetStatsByEntityUUID(client turboclient.T8cClient, entityUuid string) (turboclient.StatsResponse, *diag.ErrorDiagnostic) {
	return GetStatsByEntityUUIDWithOptions(client, entityUuid, DefaultStatistics(), "+10m")
}

/*
GetStatsByEntityUUIDWithOptions fetches stats for a given entity UUID with customizable statistics and time range.
It returns stats results or an error diagnostic.

Parameters:
  - client: The Turbonomic client to use for API calls
  - entityUuid: The UUID of the entity to fetch stats for
  - statistics: The statistics to retrieve
  - endDate: The end date for the stats query (e.g., "+10m" for 10 minutes from now)

Returns:
  - turboclient.StatsResponse: The stats response from the API
  - *diag.ErrorDiagnostic: An error diagnostic if the operation fails
*/
func GetStatsByEntityUUIDWithOptions(
	client turboclient.T8cClient,
	entityUuid string,
	statistics []turboclient.StatisticRequest,
	endDate string,
) (turboclient.StatsResponse, *diag.ErrorDiagnostic) {
	// Validate inputs
	if client == nil {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Internal error occurred while fetching stats: nil client")
		return nil, &errDiag
	}

	if entityUuid == "" {
		errDiag := diag.NewErrorDiagnostic("Invalid entity UUID specified", "Received empty entity UUID")
		return nil, &errDiag
	}

	if len(statistics) == 0 {
		errDiag := diag.NewErrorDiagnostic("Invalid statistics specified", "No statistics provided for query")
		return nil, &errDiag
	}

	if endDate == "" {
		endDate = "+10m" // Default to 10 minutes from now if not specified
	}

	// Create stats request
	statsReq := turboclient.StatsRequest{
		EntityUUID: entityUuid,
		EndDate:    endDate,
		Statistics: statistics,
	}

	// Make API call
	stats, err := client.GetStats(statsReq)
	if err != nil {
		errDiag := diag.NewErrorDiagnostic(
			"Unable to retrieve stats from Turbonomic",
			fmt.Sprintf("Error fetching stats for entity UUID %s: %s", entityUuid, err.Error()),
		)
		return nil, &errDiag
	}

	if len(stats) == 0 {
		errDiag := diag.NewErrorDiagnostic(
			"No stats found",
			fmt.Sprintf("No stats found for entity UUID: %s", entityUuid),
		)
		return nil, &errDiag
	}

	return stats, nil
}
