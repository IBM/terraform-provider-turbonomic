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

// TargetsClient provides CRUD operations for managing Turbonomic targets
type TargetsClient struct {
	client *Client
}

// Targets returns a client for managing targets
func (c *Client) Targets() *TargetsClient {
	return &TargetsClient{client: c}
}

// List retrieves all targets with optional filtering and pagination
func (tc *TargetsClient) List(ctx context.Context, params *generated.GetTargetsParams) ([]generated.TargetApiDTO, *string, error) {
	if params == nil {
		params = &generated.GetTargetsParams{}
	}

	resp, err := tc.client.api.GetTargets(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list targets: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("list targets failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result []generated.TargetApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode targets response: %w", err)
	}

	// Extract cursor from response headers if available
	var nextCursor *string
	if cursorHeader := resp.Header.Get("X-Next-Cursor"); cursorHeader != "" {
		nextCursor = &cursorHeader
	}

	return result, nextCursor, nil
}

// Get retrieves a specific target by UUID
func (tc *TargetsClient) Get(ctx context.Context, targetUUID string, params *generated.GetTargetParams) (*generated.TargetApiDTO, error) {
	if params == nil {
		params = &generated.GetTargetParams{}
	}

	resp, err := tc.client.api.GetTarget(ctx, targetUUID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get target: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get target failed with status %d: %s", resp.StatusCode, string(body))
	}

	var target generated.TargetApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return nil, fmt.Errorf("failed to decode target response: %w", err)
	}

	return &target, nil
}

// Create creates a new target
func (tc *TargetsClient) Create(ctx context.Context, target generated.TargetApiDTO, params *generated.AddTargetParams) (*generated.TargetApiDTO, error) {
	if params == nil {
		params = &generated.AddTargetParams{}
	}

	resp, err := tc.client.api.AddTarget(ctx, params, generated.AddTargetJSONRequestBody(target))
	if err != nil {
		return nil, fmt.Errorf("failed to create target: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create target failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createdTarget generated.TargetApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&createdTarget); err != nil {
		return nil, fmt.Errorf("failed to decode created target response: %w", err)
	}

	return &createdTarget, nil
}

// Update updates an existing target
func (tc *TargetsClient) Update(ctx context.Context, targetUUID string, target generated.TargetApiDTO) (*generated.TargetApiDTO, error) {
	resp, err := tc.client.api.EditTarget(ctx, targetUUID, generated.EditTargetJSONRequestBody(target))
	if err != nil {
		return nil, fmt.Errorf("failed to update target: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update target failed with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedTarget generated.TargetApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&updatedTarget); err != nil {
		return nil, fmt.Errorf("failed to decode updated target response: %w", err)
	}

	return &updatedTarget, nil
}

// Delete deletes a target by UUID
func (tc *TargetsClient) Delete(ctx context.Context, targetUUID string) error {
	resp, err := tc.client.api.DeleteTarget(ctx, targetUUID)
	if err != nil {
		return fmt.Errorf("failed to delete target: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete target failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Execute runs a validation or rediscovery operation on a target
func (tc *TargetsClient) Execute(ctx context.Context, targetUUID string, params *generated.ExecuteOnTargetParams) (*generated.TargetApiDTO, error) {
	if params == nil {
		params = &generated.ExecuteOnTargetParams{}
	}

	resp, err := tc.client.api.ExecuteOnTarget(ctx, targetUUID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute on target: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("execute on target failed with status %d: %s", resp.StatusCode, string(body))
	}

	var target generated.TargetApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return nil, fmt.Errorf("failed to decode execute target response: %w", err)
	}

	return &target, nil
}

// GetEntities retrieves entities discovered by a target
func (tc *TargetsClient) GetEntities(ctx context.Context, targetUUID string, cursor *string) ([]generated.ServiceEntityApiDTO, *string, error) {
	params := &generated.GetEntitiesByTargetUuidParams{}
	if cursor != nil {
		params.Cursor = cursor
	}

	resp, err := tc.client.api.GetEntitiesByTargetUuid(ctx, targetUUID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get target entities: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("get target entities failed with status %d: %s", resp.StatusCode, string(body))
	}

	var entities []generated.ServiceEntityApiDTO
	if err := json.NewDecoder(resp.Body).Decode(&entities); err != nil {
		return nil, nil, fmt.Errorf("failed to decode target entities response: %w", err)
	}

	// Extract cursor from response headers if available
	var nextCursor *string
	if cursorHeader := resp.Header.Get("X-Next-Cursor"); cursorHeader != "" {
		nextCursor = &cursorHeader
	}

	return entities, nextCursor, nil
}
