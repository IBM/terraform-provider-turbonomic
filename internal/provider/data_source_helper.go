// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"

	turboclient "github.com/IBM/turbonomic-go-client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
