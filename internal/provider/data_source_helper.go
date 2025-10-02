// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	turboclient "github.com/IBM/turbonomic-go-client"
)

// Constants for entity types and filter types
const (
	VirtualVolumeEntityType  = "VirtualVolume"
	DatabaseServerEntityType = "DatabaseServer"
	RelationFilterType       = "relation"
)

type SearchRequestWithOptions struct {
	turboclient.SearchRequest
	showVendorID bool
}

type EntityOption func(*SearchRequestWithOptions)

func WithEntityName(name string) EntityOption {
	return func(o *SearchRequestWithOptions) {
		o.Name = name
	}
}

func WithEntityType(entityType string) EntityOption {
	return func(o *SearchRequestWithOptions) {
		o.EntityType = entityType
	}
}

func WithEnvironmentType(env string) EntityOption {
	return func(o *SearchRequestWithOptions) {
		o.EnvironmentType = env
	}
}

func WithCloudType(cloud string) EntityOption {
	return func(o *SearchRequestWithOptions) {
		o.CloudType = cloud
	}
}

func WithOSNames(osName ...string) EntityOption {
	return func(o *SearchRequestWithOptions) {
		o.OSNames = osName
	}
}

func ShowVendorIdString(ok bool) EntityOption {
	return func(o *SearchRequestWithOptions) {
		o.showVendorID = ok
	}
}

/*
fetches turbonomic entities by given entity-name and entity-type and environment type. If there are multiple matches or no matches it returns error diagnostic object.
returns zero or one matching entity or throws an error
*/
func GetEntitiesByName(client turboclient.T8cClient, options ...EntityOption) (turboclient.SearchResults, *diag.ErrorDiagnostic) {
	if client == nil {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Internal error occurred while fetching entities")
		return nil, &errDiag
	}

	opts := SearchRequestWithOptions{
		SearchRequest: turboclient.SearchRequest{
			CaseSensitive: true,
		},
	}

	for _, o := range options {
		o(&opts)
	}

	if len(opts.Name) == 0 {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Empty entity name specified")
		return nil, &errDiag
	}

	entity, err := client.SearchEntityByName(opts.SearchRequest)
	if err != nil {
		errDiag := diag.NewErrorDiagnostic("Unable to search Turbonomic", err.Error())
		return nil, &errDiag
	} else if len(entity) > 1 {
		errDiag := diag.NewErrorDiagnostic(
			"Multiple Entities with provided name found",
			fmt.Sprintf("Multiple Entities with the name %s of type %s found in Turbonomic instance.%s",
				opts.Name,
				opts.EntityType,
				getVendorIdsString(opts.showVendorID, entity)))
		return nil, &errDiag
	}

	return entity, nil
}

func getVendorIdsString(showVendorID bool, entity turboclient.SearchResults) string {
	if showVendorID {
		return fmt.Sprintf(" Please include vendor_id in the search. Available VendorIds: [%s]", strings.Join(ExtractVendorIdValues(entity), ", "))
	}
	return ""
}

type EntityOptionWithVendorId func(*turboclient.SearchRequestByVendorId)

func WithVendorId(vendorId string) EntityOptionWithVendorId {
	return func(o *turboclient.SearchRequestByVendorId) {
		o.VendorId = vendorId
	}
}

func WithEntityTypeForVendorId(entityType string) EntityOptionWithVendorId {
	return func(o *turboclient.SearchRequestByVendorId) {
		o.EntityType = entityType
	}
}

/*
GetEntitiesByVendorId searches for entities in Turbonomic by vendor ID and entity type.
Returns:
  - turboclient.SearchResults: The search results containing matching entities
  - *diag.ErrorDiagnostic: An error diagnostic if the operation fails
*/
func GetEntitiesByVendorId(client turboclient.T8cClient, options ...EntityOptionWithVendorId) (turboclient.SearchResults, *diag.ErrorDiagnostic) {
	if client == nil {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Internal error occurred while fetching entities")
		return nil, &errDiag
	}

	opts := turboclient.SearchRequestByVendorId{
		CaseSensitive: false,
	}

	for _, o := range options {
		o(&opts)
	}

	if len(opts.VendorId) == 0 {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Empty vendor id specified")
		return nil, &errDiag
	}

	entity, err := client.SearchEntityByVendorId(opts)
	if err != nil {
		errDiag := diag.NewErrorDiagnostic("Unable to search Turbonomic", err.Error())
		return nil, &errDiag
	}

	return entity, nil
}

type ActionOption func(*turboclient.ActionsRequest)

func WithEntityUuid(uuid string) ActionOption {
	return func(o *turboclient.ActionsRequest) {
		o.Uuid = uuid
	}
}

func WithActionTypes(actionType []string) ActionOption {
	return func(o *turboclient.ActionsRequest) {
		o.ActionType = actionType
	}
}

func WithActionState(actionState []string) ActionOption {
	return func(o *turboclient.ActionsRequest) {
		o.ActionState = actionState
	}
}

/*
fetches ready actions by a entity uuid and action type. If there are multiple actions, it returns error.
returns zero or one matching action or throws an error
*/
func GetActions(client turboclient.T8cClient, options ...ActionOption) (turboclient.ActionResults, *diag.ErrorDiagnostic) {
	if client == nil {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Internal error occurred while fetching action")
		return nil, &errDiag
	}

	opts := turboclient.ActionsRequest{
		ActionState: []string{"READY"},
	}

	for _, o := range options {
		o(&opts)
	}

	if len(opts.Uuid) == 0 || len(opts.ActionType) == 0 {
		errDiag := diag.NewErrorDiagnostic("Invalid entity name or action type specified", fmt.Sprintf("Received invalid action uuid: %s / type: %v",
			opts.Uuid,
			opts.ActionType))
		return nil, &errDiag
	}

	actions, err := client.GetActionsByUUID(opts)
	if err != nil {
		errDiag := diag.NewErrorDiagnostic("Unable to retrieve actions from Turbonomic", err.Error())
		return nil, &errDiag
	} else if len(actions) > 1 {
		errDiag := diag.NewErrorDiagnostic("Multiple Entities with provided name found", fmt.Sprintf("Action with uuid: %s / type: %v returned more than one result",
			opts.Uuid,
			opts.ActionType))
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

// GetStatisticsByEntityType returns the appropriate statistics based on entity type
// It configures the statistics differently depending on the entity type:
func GetStatisticsByEntityType(entityType string) []turboclient.StatisticRequest {
	var statNames []string

	switch entityType {
	case VirtualVolumeEntityType:
		statNames = []string{StorageAccess, StorageAmount, IOThroughput}
	case DatabaseServerEntityType:
		// For database servers, we don't include IOThroughput
		statNames = []string{StorageAccess, StorageAmount}
	default:
		statNames = []string{StorageAccess, StorageAmount, IOThroughput}
	}

	return CreateStatisticRequests(statNames, entityType, RelationFilterType, FilterSold)
}

// StatsOption defines a function that configures a StatsRequest
type StatsOption func(*turboclient.StatsRequest)

// WithEntityUUID sets the entity UUID in the stats request
func WithEntityUUID(uuid string) StatsOption {
	return func(o *turboclient.StatsRequest) {
		o.EntityUUID = uuid
	}
}

// WithStatistics sets the statistics to retrieve in the stats request
func WithStatistics(statistics []turboclient.StatisticRequest) StatsOption {
	return func(o *turboclient.StatsRequest) {
		o.Statistics = statistics
	}
}

// WithEndDate sets the end date for the stats query
// e.g., "+10m" for 10 minutes from now
func WithEndDate(endDate string) StatsOption {
	return func(o *turboclient.StatsRequest) {
		o.EndDate = endDate
	}
}

/*
GetStatsByEntityUUIDAndType fetches stats for a given entity UUID with statistics based on entity type.
It returns stats results or an error diagnostic.

Parameters:
  - client: The Turbonomic client to use for API calls
  - entityUuid: The UUID of the entity to fetch stats for
  - entityType: The entity type to determine which statistics to fetch

Returns:
  - turboclient.StatsResponse: The stats response from the API
  - *diag.ErrorDiagnostic: An error diagnostic if the operation fails
*/
func GetStatsByEntityUUIDAndType(client turboclient.T8cClient, entityUuid string, entityType string) (turboclient.StatsResponse, *diag.ErrorDiagnostic) {
	return GetStats(
		client,
		WithEntityUUID(entityUuid),
		WithStatistics(GetStatisticsByEntityType(entityType)),
		WithEndDate("+10m"),
	)
}

/*
GetStats fetches stats for a given entity UUID with customizable options.
It returns stats results or an error diagnostic.

Parameters:
  - client: The Turbonomic client to use for API calls
  - options: A variadic list of StatsOption functions to configure the request

Options:
  - WithEntityUUID: Sets the entity UUID in the request
  - WithStatistics: Sets the statistics to retrieve
  - WithEndDate: Sets the end date for the stats query (e.g., "+10m" for 10 minutes from now)
  - WithEntityTypeStats: Sets statistics based on entity type

Returns:
  - turboclient.StatsResponse: The stats response from the API
  - *diag.ErrorDiagnostic: An error diagnostic if the operation fails
*/
func GetStats(
	client turboclient.T8cClient,
	options ...StatsOption,
) (turboclient.StatsResponse, *diag.ErrorDiagnostic) {
	// Validate client
	if client == nil {
		errDiag := diag.NewErrorDiagnostic("Internal error", "Internal error occurred while fetching stats: nil client")
		return nil, &errDiag
	}

	// Create stats request with default values
	statsReq := turboclient.StatsRequest{
		EndDate: "+10m", // Default to 10 minutes from now if not specified
	}

	// Apply all options
	for _, o := range options {
		o(&statsReq)
	}

	// Validate required fields
	if statsReq.EntityUUID == "" {
		errDiag := diag.NewErrorDiagnostic("Invalid entity UUID specified", "Received empty entity UUID")
		return nil, &errDiag
	}

	if len(statsReq.Statistics) == 0 {
		errDiag := diag.NewErrorDiagnostic("Invalid statistics specified", "No statistics provided for query")
		return nil, &errDiag
	}

	// Make API call
	stats, err := client.GetStats(statsReq)
	if err != nil {
		errDiag := diag.NewErrorDiagnostic(
			"Unable to retrieve stats from Turbonomic",
			fmt.Sprintf("Error fetching stats for entity UUID %s: %s", statsReq.EntityUUID, err.Error()),
		)
		return nil, &errDiag
	}

	if len(stats) == 0 {
		errDiag := diag.NewErrorDiagnostic(
			"No stats found",
			fmt.Sprintf("No stats found for entity UUID: %s", statsReq.EntityUUID),
		)
		return nil, &errDiag
	}

	return stats, nil
}

func TagEntity(client *turboclient.Client, uuid string) error {
	// tag VM entity with "optimized by" tag if not already tagged
	if len(uuid) > 0 {
		entityTagsReq := turboclient.EntityRequest{
			Uuid: uuid}

		entityTags, err := client.GetEntityTags(entityTagsReq)
		if err != nil {
			return fmt.Errorf("Unable to retrieve entity tags from Turbonomic: %v", err)
		}

		var alreadyTagged bool = false
		for _, item := range entityTags {
			if item.Key == OptimizedByTagName {
				if slices.Contains(item.Values, OptimizedByTagValue) {
					alreadyTagged = true
					break
				}
			}
		}

		if !alreadyTagged {
			tagEntityReq := turboclient.TagEntityRequest{
				Uuid: uuid,
				Tags: []turboclient.Tag{
					{
						Key:    OptimizedByTagName,
						Values: []string{OptimizedByTagValue},
					},
				},
			}

			_, err := client.TagEntity(tagEntityReq)
			if err != nil {
				if strings.Contains(err.Error(), TagAlreadyExistsErrorMsg) {
					return nil
				}
				return fmt.Errorf("Unable to tag an entity in Turbonomic: %v", err)
			}
		}
	}
	return nil
}

// Nullable is an interface for types that can be null
type Nullable interface {
	IsNull() bool
}

// StringValue is an interface for string types that can provide their string value
type StringValue interface {
	ValueString() string
}

// applyDefaultIfEmptyGeneric is a generic function that returns the default value if the provided field is null,
// otherwise returns the original field value. For string types, it also checks if the string is empty.
//
// Parameters:
//   - field: The original value to check (must implement Nullable)
//   - def: The default value to use if the original is null
//
// Returns:
//   - The original value if it's not null (and not empty for strings), otherwise the default value
func applyDefaultIfEmptyGeneric[T Nullable](field, def T) T {
	// Special handling for string types
	if strField, ok := any(field).(StringValue); ok {
		// For string types, check if it's null OR empty
		if field.IsNull() || len(strField.ValueString()) == 0 {
			// Only apply default if it's not null
			if !def.IsNull() {
				return def
			}
		}
		return field
	}

	// For non-string types, just check if it's null
	if field.IsNull() {
		return def
	}
	return field
}

// ExtractVendorIdValues extracts all vendor ID values from search results
// It returns a slice of strings containing all vendor ID values
func ExtractVendorIdValues(entities turboclient.SearchResults) []string {
	var vendorIdValues []string

	for _, entity := range entities {
		// Extract only the values from the VendorIds map
		for _, value := range entity.VendorIds {
			vendorIdValues = append(vendorIdValues, value)
		}
	}

	return vendorIdValues
}
