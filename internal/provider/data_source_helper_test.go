package provider

import (
	"testing"

	turboclient "github.com/IBM/turbonomic-go-client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockT8cClient is a mock implementation of the T8cClient interface.
type MockT8cClient struct {
	mock.Mock
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

func TestGetEntitiesByNameAndType(t *testing.T) {
	entityName := "exampleEntity"
	entityType := "exampleType"
	envType := "exampleEnv"
	cloudType := "exampleCloud"

	t.Run("Client is nil", func(t *testing.T) {
		_, errDiag := GetEntitiesByNameAndType(nil, entityName, entityType, envType, cloudType)
		assert.NotNil(t, errDiag)
	})

	t.Run("Entity name is empty", func(t *testing.T) {
		mockClient := new(MockT8cClient)
		_, errDiag := GetEntitiesByNameAndType(mockClient, "", entityType, envType, cloudType)
		assert.NotNil(t, errDiag)
		mockClient.AssertExpectations(t)
	})

	t.Run("Successful search", func(t *testing.T) {
		mockClient := new(MockT8cClient)
		expected := turboclient.SearchResults{{UUID: "uuid"}}

		mockClient.On("SearchEntityByName", mock.Anything).Return(expected, nil).Once()

		entities, errDiag := GetEntitiesByNameAndType(mockClient, entityName, entityType, envType, cloudType)
		assert.Nil(t, errDiag)
		assert.Equal(t, expected, entities)

		mockClient.AssertExpectations(t)
	})

	t.Run("Multiple matches found", func(t *testing.T) {
		mockClient := new(MockT8cClient)
		multiple := turboclient.SearchResults{{UUID: "uuid1"}, {UUID: "uuid2"}}

		mockClient.On("SearchEntityByName", mock.Anything).Return(multiple, nil).Once()

		_, errDiag := GetEntitiesByNameAndType(mockClient, entityName, entityType, envType, cloudType)
		assert.NotNil(t, errDiag)

		mockClient.AssertExpectations(t)
	})
}

func TestGetActionsByEntityUUIDAndType(t *testing.T) {
	entityUUID := "exampleUuid"
	actionType := "exampleAction"

	t.Run("Client is nil", func(t *testing.T) {
		_, errDiag := GetActionsByEntityUUIDAndType(nil, entityUUID, actionType)
		assert.NotNil(t, errDiag)
	})

	t.Run("Entity UUID is empty", func(t *testing.T) {
		client := new(MockT8cClient)
		_, errDiag := GetActionsByEntityUUIDAndType(client, "", actionType)
		assert.NotNil(t, errDiag)
		client.AssertExpectations(t)
	})

	t.Run("Action Type is empty", func(t *testing.T) {
		client := new(MockT8cClient)
		_, errDiag := GetActionsByEntityUUIDAndType(client, entityUUID, "")
		assert.NotNil(t, errDiag)
		client.AssertExpectations(t)
	})

	t.Run("Client returns error", func(t *testing.T) {
		client := new(MockT8cClient)
		client.On("GetActionsByUUID", mock.Anything).Return(turboclient.ActionResults{}, assert.AnError).Once()

		_, errDiag := GetActionsByEntityUUIDAndType(client, entityUUID, actionType)
		assert.NotNil(t, errDiag)

		client.AssertExpectations(t)
	})

	t.Run("Single successful action", func(t *testing.T) {
		client := new(MockT8cClient)
		expected := turboclient.ActionResults{
			{UUID: "action-uuid-1"},
		}

		client.On("GetActionsByUUID", mock.Anything).Return(expected, nil).Once()

		actions, errDiag := GetActionsByEntityUUIDAndType(client, entityUUID, actionType)
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

		_, errDiag := GetActionsByEntityUUIDAndType(client, entityUUID, actionType)
		assert.NotNil(t, errDiag) // Assume function treats multiple as error

		client.AssertExpectations(t)
	})
}
