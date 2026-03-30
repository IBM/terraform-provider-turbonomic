// Copyright (c) IBM Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	turboclient "github.com/IBM/turbonomic-go-client"
)

// MockT8cClient is a mock implementation of the T8cClient interface.
type MockT8cClient struct {
	mock.Mock
}

// SearchEntityByVendorId implements turboclient.T8cClient.
func (m *MockT8cClient) SearchEntityByVendorId(searchReq turboclient.SearchRequestByVendorId) (turboclient.SearchResults, error) {
	args := m.Called(searchReq)
	return args.Get(0).(turboclient.SearchResults), args.Error(1)
}

// GetStats implements turboclient.T8cClient.
func (m *MockT8cClient) GetStats(statsReq turboclient.StatsRequest) (turboclient.StatsResponse, error) {
	args := m.Called(statsReq)
	return args.Get(0).(turboclient.StatsResponse), args.Error(1)
}

// GetActionsByUUID mocks the GetActionsByUUID method.
func (m *MockT8cClient) GetActionsByUUID(actionReq turboclient.ActionsRequest) (turboclient.ActionResults, error) {
	args := m.Called(actionReq)
	return args.Get(0).(turboclient.ActionResults), args.Error(1)
}

// GetEntity mocks the GetEntity method.
func (m *MockT8cClient) GetEntity(reqOpts turboclient.EntityRequest) (*turboclient.EntityResults, error) {
	args := m.Called(reqOpts)
	return args.Get(0).(*turboclient.EntityResults), args.Error(1)
}

// GetEntityTags mocks the GetEntityTags method.
func (m *MockT8cClient) GetEntityTags(reqOpts turboclient.EntityRequest) ([]turboclient.Tag, error) {
	args := m.Called(reqOpts)
	return args.Get(0).([]turboclient.Tag), args.Error(1)
}

// TagEntity mocks the TagEntity method.
func (m *MockT8cClient) TagEntity(reqOpts turboclient.TagEntityRequest) ([]turboclient.Tag, error) {
	args := m.Called(reqOpts)
	return args.Get(0).([]turboclient.Tag), args.Error(1)
}

// SearchEntities mocks the SearchEntities method.
func (m *MockT8cClient) SearchEntities(searchCriteria turboclient.SearchDTO, reqParams turboclient.CommonReqParams) (turboclient.SearchResults, error) {
	args := m.Called(searchCriteria, reqParams)
	return args.Get(0).(turboclient.SearchResults), args.Error(1)
}

// SearchEntityByName mocks the SearchEntityByName method.
func (m *MockT8cClient) SearchEntityByName(searchReq turboclient.SearchRequest) (turboclient.SearchResults, error) {
	args := m.Called(searchReq)
	return args.Get(0).(turboclient.SearchResults), args.Error(1)
}

func TestGetEntitiesByName(t *testing.T) {
	entityName := "exampleEntity"
	entityType := "exampleType"
	envType := "exampleEnv"
	cloudType := "exampleCloud"

	t.Run("Client is nil", func(t *testing.T) {
		_, errDiag := GetEntitiesByName(nil, WithEntityName(entityName), WithEntityType(entityType), WithEnvironmentType(envType), WithCloudType(cloudType))
		assert.NotNil(t, errDiag)
	})

	t.Run("Entity name is empty", func(t *testing.T) {
		mockClient := new(MockT8cClient)
		_, errDiag := GetEntitiesByName(mockClient, WithEntityType(entityType), WithEnvironmentType(envType), WithCloudType(cloudType))
		assert.NotNil(t, errDiag)
		mockClient.AssertExpectations(t)

		_, errDiag = GetEntitiesByName(mockClient, WithEntityName(""), WithEntityType(entityType), WithEnvironmentType(envType), WithCloudType(cloudType))
		assert.NotNil(t, errDiag)
		mockClient.AssertExpectations(t)
	})

	t.Run("Successful search", func(t *testing.T) {
		mockClient := new(MockT8cClient)
		expected := turboclient.SearchResults{
			{UUID: "uuid"},
		}

		mockClient.On("SearchEntityByName", mock.Anything).Return(expected, nil).Once()

		entities, errDiag := GetEntitiesByName(mockClient, WithEntityName(entityName), WithEntityType(entityType), WithEnvironmentType(envType), WithCloudType(cloudType))
		assert.Nil(t, errDiag)
		assert.Equal(t, expected, entities)

		mockClient.AssertExpectations(t)
	})

	t.Run("Multiple matches found", func(t *testing.T) {
		mockClient := new(MockT8cClient)
		multiple := turboclient.SearchResults{
			{UUID: "uuid1"}, {UUID: "uuid2"},
		}

		mockClient.On("SearchEntityByName", mock.Anything).Return(multiple, nil).Once()

		_, errDiag := GetEntitiesByName(mockClient, WithEntityName(entityName), WithEntityType(entityType), WithEnvironmentType(envType), WithCloudType(cloudType))
		assert.NotNil(t, errDiag)

		mockClient.AssertExpectations(t)
	})
}

func TestGetActions(t *testing.T) {
	entityUUID := "exampleUuid"
	actionTypes := []string{"exampleAction"}

	t.Run("Client is nil", func(t *testing.T) {
		_, errDiag := GetActions(nil, WithEntityUuid(entityUUID), WithActionTypes(actionTypes))
		assert.NotNil(t, errDiag)
	})

	t.Run("Entity UUID is empty", func(t *testing.T) {
		client := new(MockT8cClient)
		_, errDiag := GetActions(client, WithActionTypes(actionTypes))
		assert.NotNil(t, errDiag)
		client.AssertExpectations(t)

		_, errDiag = GetActions(client, WithEntityUuid(""), WithActionTypes(actionTypes))
		assert.NotNil(t, errDiag)
		client.AssertExpectations(t)
	})

	t.Run("Action Type is empty", func(t *testing.T) {
		client := new(MockT8cClient)
		_, errDiag := GetActions(client, WithEntityUuid(entityUUID))
		assert.NotNil(t, errDiag)
		client.AssertExpectations(t)
	})

	t.Run("Client returns error", func(t *testing.T) {
		client := new(MockT8cClient)
		client.On("GetActionsByUUID", mock.Anything).Return(turboclient.ActionResults{}, assert.AnError).Once()

		_, errDiag := GetActions(client, WithEntityUuid(entityUUID), WithActionTypes(actionTypes))
		assert.NotNil(t, errDiag)

		client.AssertExpectations(t)
	})

	t.Run("Single successful action", func(t *testing.T) {
		client := new(MockT8cClient)
		expected := turboclient.ActionResults{
			{UUID: "action-uuid-1"},
		}

		client.On("GetActionsByUUID", mock.Anything).Return(expected, nil).Once()

		actions, errDiag := GetActions(client, WithEntityUuid(entityUUID), WithActionTypes(actionTypes))
		assert.Nil(t, errDiag)
		assert.Equal(t, expected, actions)

		client.AssertExpectations(t)
	})

	t.Run("Multiple actions returned", func(t *testing.T) {
		client := new(MockT8cClient)
		multiple := turboclient.ActionResults{
			{UUID: "action-uuid-1"},
			{UUID: "action-uuid-2"},
		}

		client.On("GetActionsByUUID", mock.Anything).Return(multiple, nil).Once()

		_, errDiag := GetActions(client, WithEntityUuid(entityUUID), WithActionTypes(actionTypes))
		assert.NotNil(t, errDiag) // Assume function treats multiple as error

		client.AssertExpectations(t)
	})
}

func TestGetStatsByEntityUUIDAndType(t *testing.T) {
	entityUUID := "test-entity-uuid"

	t.Run("Client is nil", func(t *testing.T) {
		_, errDiag := GetStatsByEntityUUIDAndType(nil, entityUUID, "VirtualVolume")
		assert.NotNil(t, errDiag)
		assert.Contains(t, errDiag.Detail(), "nil client")
	})

	t.Run("Entity UUID is empty", func(t *testing.T) {
		client := new(MockT8cClient)
		_, errDiag := GetStatsByEntityUUIDAndType(client, "", "VirtualVolume")
		assert.NotNil(t, errDiag)
		assert.Contains(t, errDiag.Detail(), "empty entity UUID")
		client.AssertExpectations(t)
	})

	t.Run("Client returns error", func(t *testing.T) {
		client := new(MockT8cClient)
		client.On("GetStats", mock.Anything).Return(turboclient.StatsResponse{}, assert.AnError).Once()

		_, errDiag := GetStatsByEntityUUIDAndType(client, entityUUID, "VirtualVolume")
		assert.NotNil(t, errDiag)
		assert.Contains(t, errDiag.Detail(), "Error fetching stats")

		client.AssertExpectations(t)
	})

	t.Run("Client returns empty response", func(t *testing.T) {
		client := new(MockT8cClient)
		client.On("GetStats", mock.Anything).Return(turboclient.StatsResponse{}, nil).Once()

		_, errDiag := GetStatsByEntityUUIDAndType(client, entityUUID, "VirtualVolume")
		assert.NotNil(t, errDiag)
		assert.Contains(t, errDiag.Detail(), "No stats found")

		client.AssertExpectations(t)
	})

	t.Run("Successful stats retrieval", func(t *testing.T) {
		client := new(MockT8cClient)

		// Create a mock response with minimal structure
		mockResponse := turboclient.StatsResponse{
			{
				DisplayName: "test-entity",
				Statistics: []turboclient.Statistic{
					{Name: StorageAccess},
					{Name: StorageAmount},
					{Name: IOThroughput},
				},
				Epoch: "CURRENT",
			},
		}

		// Verify that the request contains the expected parameters
		client.On("GetStats", mock.MatchedBy(func(req turboclient.StatsRequest) bool {
			// Check that the entity UUID matches
			if req.EntityUUID != entityUUID {
				return false
			}

			// Check that we have the expected number of statistics
			if len(req.Statistics) != 3 {
				return false
			}

			// Check that the statistics include the expected names
			statNames := make([]string, len(req.Statistics))
			for i, stat := range req.Statistics {
				statNames[i] = stat.Name
			}

			return contains(statNames, StorageAccess) &&
				contains(statNames, StorageAmount) &&
				contains(statNames, IOThroughput)
		})).Return(mockResponse, nil).Once()

		stats, errDiag := GetStatsByEntityUUIDAndType(client, entityUUID, "VirtualVolume")
		assert.Nil(t, errDiag)
		assert.Equal(t, mockResponse, stats)
		assert.Equal(t, 1, len(stats))
		assert.Equal(t, 3, len(stats[0].Statistics))
		assert.Equal(t, "CURRENT", stats[0].Epoch)

		client.AssertExpectations(t)
	})

	t.Run("Successful stats retrieval with EBS data", func(t *testing.T) {
		// Read the test data file
		jsonFile, err := os.Open("testdata/ebs_data_source/ebs_stats_valid_resp.json")
		if err != nil {
			t.Fatalf("Error opening test data file: %v", err)
		}
		defer func() {
			if err := jsonFile.Close(); err != nil {
				t.Errorf("Error closing test data file: %v", err)
			}
		}()

		jsonData, err := io.ReadAll(jsonFile)
		if err != nil {
			t.Fatalf("Error reading test data file: %v", err)
		}

		// We don't need an HTTP server since we're using the mock client directly

		// Create a client and call the function
		client := new(MockT8cClient)

		// Create a mock response using the JSON data
		var mockResponse turboclient.StatsResponse
		err = json.Unmarshal(jsonData, &mockResponse)
		if err != nil {
			t.Fatalf("Error unmarshaling JSON data: %v", err)
		}

		client.On("GetStats", mock.Anything).Return(mockResponse, nil).Once()

		stats, errDiag := GetStatsByEntityUUIDAndType(client, "vol-05f7c906f860b4d3c", "VirtualVolume")
		if errDiag != nil {
			t.Fatalf("Error getting stats: %v", errDiag)
		}

		// Verify the response
		assert.Equal(t, 3, len(stats), "Expected 3 stats entries (HISTORICAL, CURRENT, PROJECTED)")

		// Check the first entry (HISTORICAL)
		assert.Equal(t, "vol-05f7c906f860b4d3c", stats[0].DisplayName)
		assert.Equal(t, "HISTORICAL", stats[0].Epoch)
		assert.Equal(t, 3, len(stats[0].Statistics))

		// Check the second entry (CURRENT)
		assert.Equal(t, "vol-05f7c906f860b4d3c", stats[1].DisplayName)
		assert.Equal(t, "CURRENT", stats[1].Epoch)
		assert.Equal(t, 3, len(stats[1].Statistics))

		// Check the third entry (PROJECTED)
		assert.Equal(t, "vol-05f7c906f860b4d3c", stats[2].DisplayName)
		assert.Equal(t, "PROJECTED", stats[2].Epoch)
		assert.Equal(t, 6, len(stats[2].Statistics))

		// Verify specific statistics in the CURRENT entry
		currentStats := stats[1].Statistics
		var storageAccessStat, storageAmountStat, ioThroughputStat *turboclient.Statistic

		for i := range currentStats {
			switch currentStats[i].Name {
			case "StorageAccess":
				storageAccessStat = &currentStats[i]
			case "StorageAmount":
				storageAmountStat = &currentStats[i]
			case "IOThroughput":
				ioThroughputStat = &currentStats[i]
			}
		}

		// Check StorageAccess stat
		assert.NotNil(t, storageAccessStat, "StorageAccess stat should exist")
		assert.Equal(t, "IOPS", storageAccessStat.Units)

		// Check StorageAmount stat
		assert.NotNil(t, storageAmountStat, "StorageAmount stat should exist")
		assert.Equal(t, "MB", storageAmountStat.Units)

		// Check IOThroughput stat
		assert.NotNil(t, ioThroughputStat, "IOThroughput stat should exist")
		assert.Equal(t, "Kbit/sec", ioThroughputStat.Units)
	})
}

func TestGetStatisticsByEntityType(t *testing.T) {
	t.Run("VirtualVolume entity type", func(t *testing.T) {
		stats := GetStatisticsByEntityType("VirtualVolume")
		assert.Equal(t, 3, len(stats), "VirtualVolume should have 3 statistics")

		statNames := make([]string, len(stats))
		for i, stat := range stats {
			statNames[i] = stat.Name
		}

		assert.True(t, contains(statNames, StorageAccess), "Should include StorageAccess")
		assert.True(t, contains(statNames, StorageAmount), "Should include StorageAmount")
		assert.True(t, contains(statNames, IOThroughput), "Should include IOThroughput")
	})

	t.Run("DatabaseServer entity type", func(t *testing.T) {
		stats := GetStatisticsByEntityType("DatabaseServer")
		assert.Equal(t, 2, len(stats), "DatabaseServer should have 2 statistics")

		statNames := make([]string, len(stats))
		for i, stat := range stats {
			statNames[i] = stat.Name
		}

		assert.True(t, contains(statNames, StorageAccess), "Should include StorageAccess")
		assert.True(t, contains(statNames, StorageAmount), "Should include StorageAmount")
		assert.False(t, contains(statNames, IOThroughput), "Should NOT include IOThroughput")
	})

	t.Run("Unknown entity type", func(t *testing.T) {
		stats := GetStatisticsByEntityType("UnknownType")
		assert.Equal(t, 3, len(stats), "Unknown type should default to 3 statistics")

		statNames := make([]string, len(stats))
		for i, stat := range stats {
			statNames[i] = stat.Name
		}

		assert.True(t, contains(statNames, StorageAccess), "Should include StorageAccess")
		assert.True(t, contains(statNames, StorageAmount), "Should include StorageAmount")
		assert.True(t, contains(statNames, IOThroughput), "Should include IOThroughput")
	})
}

func TestApplyDefaultIfEmptyGeneric_StringValue(t *testing.T) {
	tests := []struct {
		name     string
		field    types.String
		def      types.String
		expected types.String
	}{
		{
			name:     "field not null and not empty",
			field:    types.StringValue("hello"),
			def:      types.StringValue("default"),
			expected: types.StringValue("hello"),
		},
		{
			name:     "field empty",
			field:    types.StringValue(""),
			def:      types.StringValue("default"),
			expected: types.StringValue("default"),
		},
		{
			name:     "field empty but default null",
			field:    types.StringValue(""),
			def:      types.StringNull(),
			expected: types.StringValue(""),
		},
		{
			name:     "field null",
			field:    types.StringNull(),
			def:      types.StringValue("default"),
			expected: types.StringValue("default"),
		},
		{
			name:     "field null and default null",
			field:    types.StringNull(),
			def:      types.StringNull(),
			expected: types.StringNull(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyDefaultIfEmptyGeneric[types.String](tt.field, tt.def)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestApplyDefaultIfEmptyGeneric_Int64Value(t *testing.T) {
	tests := []struct {
		name     string
		field    types.Int64
		def      types.Int64
		expected types.Int64
	}{
		{
			name:     "field not null",
			field:    types.Int64Value(42),
			def:      types.Int64Value(99),
			expected: types.Int64Value(42),
		},
		{
			name:     "field null",
			field:    types.Int64Null(),
			def:      types.Int64Value(99),
			expected: types.Int64Value(99),
		},
		{
			name:     "field null and default null",
			field:    types.Int64Null(),
			def:      types.Int64Null(),
			expected: types.Int64Null(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyDefaultIfEmptyGeneric[types.Int64](tt.field, tt.def)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestCanExecuteAction(t *testing.T) {
	tests := []struct {
		name           string
		actions        turboclient.ActionResults
		expectedResult bool
		expectedMsg    string
	}{
		{
			name: "RECOMMEND action mode",
			actions: turboclient.ActionResults{
				{
					ActionMode:             "RECOMMEND",
					ActionStateDescription: "READY_ACCEPT_AND_EXECUTE",
				},
			},
			expectedResult: false,
			expectedMsg:    "actionMode is set to RECOMMEND, Turbonomic action is not executable",
		},
		{
			name: "Manual mode with ready state",
			actions: turboclient.ActionResults{
				{
					ActionMode:             "MANUAL",
					ActionStateDescription: "READY_ACCEPT_AND_EXECUTE",
				},
			},
			expectedResult: true,
			expectedMsg:    "",
		},
		{
			name: "Automatic mode with ready state",
			actions: turboclient.ActionResults{
				{
					ActionMode:             "AUTOMATIC",
					ActionStateDescription: "READY_ACCEPT_AND_EXECUTE",
				},
			},
			expectedResult: true,
			expectedMsg:    "",
		},
		{
			name: "Scheduled action with active time window",
			actions: turboclient.ActionResults{
				{
					ActionMode:             "MANUAL",
					ActionStateDescription: "READY_ACCEPT_AND_WAIT_FOR_SCHEDULE",
					ActionSchedule: struct {
						UUID                               string `json:"uuid"`
						DisplayName                        string `json:"displayName"`
						NextOccurrence                     string `json:"nextOccurrence"`
						NextOccurrenceTimestamp            int64  `json:"nextOccurrenceTimestamp"`
						TimeZoneId                         string `json:"timeZoneId"`
						Mode                               string `json:"mode"`
						AcceptedByUserForMaintenanceWindow bool   `json:"acceptedByUserForMaintenanceWindow"`
						RemaingTimeActiveInMs              int64  `json:"remaingTimeActiveInMs"`
					}{
						UUID:                               "test-uuid",
						DisplayName:                        "Test Schedule",
						NextOccurrence:                     "2055-01-01T12:00:00Z",
						NextOccurrenceTimestamp:            1672574400000,
						TimeZoneId:                         "UTC",
						Mode:                               "MANUAL",
						AcceptedByUserForMaintenanceWindow: false,
						RemaingTimeActiveInMs:              1000,
					},
				},
			},
			expectedResult: true,
			expectedMsg:    "",
		},
		{
			name: "Scheduled action with inactive time window",
			actions: turboclient.ActionResults{
				{
					ActionMode:             "MANUAL",
					ActionStateDescription: "READY_ACCEPT_AND_WAIT_FOR_SCHEDULE",
					ActionSchedule: struct {
						UUID                               string `json:"uuid"`
						DisplayName                        string `json:"displayName"`
						NextOccurrence                     string `json:"nextOccurrence"`
						NextOccurrenceTimestamp            int64  `json:"nextOccurrenceTimestamp"`
						TimeZoneId                         string `json:"timeZoneId"`
						Mode                               string `json:"mode"`
						AcceptedByUserForMaintenanceWindow bool   `json:"acceptedByUserForMaintenanceWindow"`
						RemaingTimeActiveInMs              int64  `json:"remaingTimeActiveInMs"`
					}{
						UUID:                               "test-uuid",
						DisplayName:                        "Test Schedule",
						NextOccurrence:                     "2055-01-01T12:00:00Z",
						NextOccurrenceTimestamp:            1672574400000,
						TimeZoneId:                         "UTC",
						Mode:                               "MANUAL",
						AcceptedByUserForMaintenanceWindow: false,
						RemaingTimeActiveInMs:              0,
					},
				},
			},
			expectedResult: false,
			expectedMsg:    "scheduled action execution window is not active, next occurrence will be 2055-01-01T12:00:00Z",
		},
		{
			name: "Action with non-executable state",
			actions: turboclient.ActionResults{
				{
					ActionMode:             "MANUAL",
					ActionStateDescription: "SCHEDULED_WAITING_FOR_EXTERNAL_APPROVAL",
				},
			},
			expectedResult: false,
			expectedMsg:    "actionStateDescription is set to SCHEDULED_WAITING_FOR_EXTERNAL_APPROVAL, Turbonomic action is not executable",
		},
		// Empty action results case is handled separately
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, msg := canExecuteAction(tt.actions)
			if result != tt.expectedResult {
				t.Errorf("Expected result %v, got %v", tt.expectedResult, result)
			}
			if msg != tt.expectedMsg {
				t.Errorf("Expected message %q, got %q", tt.expectedMsg, msg)
			}
		})
	}
}

// TestCanExecuteActionEdgeCases tests edge cases for the canExecuteAction function
func TestCanExecuteActionEdgeCases(t *testing.T) {
	// Test with nil ActionResults
	t.Run("Nil action results", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic with nil ActionResults, but no panic occurred")
			}
		}()

		canExecuteAction(nil)
	})

	// Test with empty ActionResults
	t.Run("Empty action results", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic with empty ActionResults, but no panic occurred")
			}
		}()

		canExecuteAction(turboclient.ActionResults{})
	})
}
