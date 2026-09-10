// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/IBM/turbonomic-go-client/api/generated"
)

// GroupsClient provides CRUD operations for managing Turbonomic groups
type GroupsClient struct {
	client *Client
}

// Groups returns a client for managing groups
func (c *Client) Groups() *GroupsClient {
	return &GroupsClient{client: c}
}

// List retrieves all groups with optional pagination
func (gc *GroupsClient) List(ctx context.Context, cursor *string) ([]generated.GroupApiDTO, *string, error) {
	params := &generated.GetGroupsParams{}
	if cursor != nil {
		params.Cursor = cursor
	}

	resp, err := gc.client.api.GetGroups(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("list groups failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result []generated.GroupApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode groups response: %w", err)
	}

	// Extract cursor from response headers if available
	var nextCursor *string
	if cursorHeader := resp.Header.Get("X-Next-Cursor"); cursorHeader != "" {
		nextCursor = &cursorHeader
	}

	return result, nextCursor, nil
}

// Get retrieves a specific group by UUID
func (gc *GroupsClient) Get(ctx context.Context, groupUUID string, includeAspects bool) (*generated.GroupApiDTO, error) {
	params := &generated.GetGroupByUuidParams{
		IncludeAspects: &includeAspects,
	}

	resp, err := gc.client.api.GetGroupByUuid(ctx, groupUUID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get group failed with status %d: %s", resp.StatusCode, string(body))
	}

	var group generated.GroupApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return nil, fmt.Errorf("failed to decode group response: %w", err)
	}

	return &group, nil
}

// Create creates a new group
func (gc *GroupsClient) Create(ctx context.Context, group generated.GroupApiDTO) (*generated.GroupApiDTO, error) {
	resp, err := gc.client.api.CreateGroup(ctx, generated.CreateGroupJSONRequestBody(group))
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create group failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createdGroup generated.GroupApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&createdGroup); err != nil {
		return nil, fmt.Errorf("failed to decode created group response: %w", err)
	}

	return &createdGroup, nil
}

// Update updates an existing group
func (gc *GroupsClient) Update(ctx context.Context, groupUUID string, group generated.GroupApiDTO) (*generated.GroupApiDTO, error) {
	resp, err := gc.client.api.EditGroup(ctx, groupUUID, generated.EditGroupJSONRequestBody(group))
	if err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update group failed with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedGroup generated.GroupApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&updatedGroup); err != nil {
		return nil, fmt.Errorf("failed to decode updated group response: %w", err)
	}

	return &updatedGroup, nil
}

// Delete deletes a group by UUID
func (gc *GroupsClient) Delete(ctx context.Context, groupUUID string) error {
	resp, err := gc.client.api.DeleteGroup(ctx, groupUUID)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete group failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Search searches for groups based on query criteria
func (gc *GroupsClient) Search(ctx context.Context, query generated.GroupQueryApiDTO) ([]generated.GroupApiDTO, error) {
	resp, err := gc.client.api.SearchGroups(ctx, generated.SearchGroupsJSONRequestBody(query))
	if err != nil {
		return nil, fmt.Errorf("failed to search groups: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search groups failed with status %d: %s", resp.StatusCode, string(body))
	}

	var groups []generated.GroupApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return groups, nil
}

// GetMembers retrieves members of a group
func (gc *GroupsClient) GetMembers(ctx context.Context, groupUUID string, cursor *string) ([]generated.ServiceEntityApiDTO, *string, error) {
	params := &generated.GetMembersByGroupUuidParams{}
	if cursor != nil {
		params.Cursor = cursor
	}

	resp, err := gc.client.api.GetMembersByGroupUuid(ctx, groupUUID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("get group members failed with status %d: %s", resp.StatusCode, string(body))
	}

	var members []generated.ServiceEntityApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, nil, fmt.Errorf("failed to decode members response: %w", err)
	}

	// Extract cursor from response headers if available
	var nextCursor *string
	if cursorHeader := resp.Header.Get("X-Next-Cursor"); cursorHeader != "" {
		nextCursor = &cursorHeader
	}

	return members, nextCursor, nil
}

// GetEntities retrieves entities in a group
func (gc *GroupsClient) GetEntities(ctx context.Context, groupUUID string, cursor *string) ([]generated.ServiceEntityApiDTO, *string, error) {
	params := &generated.GetEntitiesByGroupUuidParams{}
	if cursor != nil {
		params.Cursor = cursor
	}

	resp, err := gc.client.api.GetEntitiesByGroupUuid(ctx, groupUUID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get group entities: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("get group entities failed with status %d: %s", resp.StatusCode, string(body))
	}

	var entities []generated.ServiceEntityApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&entities); err != nil {
		return nil, nil, fmt.Errorf("failed to decode entities response: %w", err)
	}

	// Extract cursor from response headers if available
	var nextCursor *string
	if cursorHeader := resp.Header.Get("X-Next-Cursor"); cursorHeader != "" {
		nextCursor = &cursorHeader
	}

	return entities, nextCursor, nil
}

// Count returns the count of groups based on criteria
func (gc *GroupsClient) Count(ctx context.Context, countRequest generated.GroupCountRequestApiDTO) (int, error) {
	resp, err := gc.client.api.CountGroups(ctx, generated.CountGroupsJSONRequestBody(countRequest))
	if err != nil {
		return 0, fmt.Errorf("failed to count groups: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("count groups failed with status %d: %s", resp.StatusCode, string(body))
	}

	var count int
	if err := json.NewDecoder(resp.Body).Decode(&count); err != nil {
		return 0, fmt.Errorf("failed to decode count response: %w", err)
	}

	return count, nil
}

// GetFields retrieves available fields for a group type
// groupTypeValue should be one of: "BillingFamily", "BusinessAccountFolder", "Cluster", "Group", "NodePool", "ResourceGroup", "StorageCluster", "VirtualMachineCluster"
func (gc *GroupsClient) GetFields(ctx context.Context, groupTypeValue string) ([]generated.FieldApiDTO, error) {
	// Note: There's a naming conflict in the generated code where GroupType is both a constant and a type.
	// We work around this by using the string value directly via JSON marshaling.
	request := generated.GroupMetadataRequestApiDTO{}
	// Use JSON to set the field to avoid the naming conflict
	type tempStruct struct {
		GroupType string `json:"groupType"`
	}
	temp := tempStruct{GroupType: groupTypeValue}
	data, _ := json.Marshal(temp)
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal group type: %w", err)
	}

	resp, err := gc.client.api.GroupFields(ctx, generated.GroupFieldsJSONRequestBody(request))
	if err != nil {
		return nil, fmt.Errorf("failed to get group fields: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get group fields failed with status %d: %s", resp.StatusCode, string(body))
	}

	var fields []generated.FieldApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return nil, fmt.Errorf("failed to decode fields response: %w", err)
	}

	return fields, nil
}
