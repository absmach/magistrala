// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package bootstrap

import (
	"context"

	"github.com/absmach/magistrala/bootstrap"
	mock "github.com/stretchr/testify/mock"
)

// NewMockConfigRepository creates a new instance of MockConfigRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockConfigRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockConfigRepository {
	mock := &MockConfigRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// MockConfigRepository is an autogenerated mock type for the ConfigRepository type
type MockConfigRepository struct {
	mock.Mock
}

type MockConfigRepository_Expecter struct {
	mock *mock.Mock
}

func (_m *MockConfigRepository) EXPECT() *MockConfigRepository_Expecter {
	return &MockConfigRepository_Expecter{mock: &_m.Mock}
}

// ChangeState provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) ChangeState(ctx context.Context, domainID string, id string, state bootstrap.State) error {
	ret := _mock.Called(ctx, domainID, id, state)

	if len(ret) == 0 {
		panic("no return value specified for ChangeState")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, bootstrap.State) error); ok {
		r0 = returnFunc(ctx, domainID, id, state)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockConfigRepository_ChangeState_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ChangeState'
type MockConfigRepository_ChangeState_Call struct {
	*mock.Call
}

// ChangeState is a helper method to define mock.On call
//   - ctx
//   - domainID
//   - id
//   - state
func (_e *MockConfigRepository_Expecter) ChangeState(ctx interface{}, domainID interface{}, id interface{}, state interface{}) *MockConfigRepository_ChangeState_Call {
	return &MockConfigRepository_ChangeState_Call{Call: _e.mock.On("ChangeState", ctx, domainID, id, state)}
}

func (_c *MockConfigRepository_ChangeState_Call) Run(run func(ctx context.Context, domainID string, id string, state bootstrap.State)) *MockConfigRepository_ChangeState_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(bootstrap.State))
	})
	return _c
}

func (_c *MockConfigRepository_ChangeState_Call) Return(err error) *MockConfigRepository_ChangeState_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockConfigRepository_ChangeState_Call) RunAndReturn(run func(ctx context.Context, domainID string, id string, state bootstrap.State) error) *MockConfigRepository_ChangeState_Call {
	_c.Call.Return(run)
	return _c
}

// ConnectClient provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) ConnectClient(ctx context.Context, channelID string, clientID string) error {
	ret := _mock.Called(ctx, channelID, clientID)

	if len(ret) == 0 {
		panic("no return value specified for ConnectClient")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = returnFunc(ctx, channelID, clientID)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockConfigRepository_ConnectClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ConnectClient'
type MockConfigRepository_ConnectClient_Call struct {
	*mock.Call
}

// ConnectClient is a helper method to define mock.On call
//   - ctx
//   - channelID
//   - clientID
func (_e *MockConfigRepository_Expecter) ConnectClient(ctx interface{}, channelID interface{}, clientID interface{}) *MockConfigRepository_ConnectClient_Call {
	return &MockConfigRepository_ConnectClient_Call{Call: _e.mock.On("ConnectClient", ctx, channelID, clientID)}
}

func (_c *MockConfigRepository_ConnectClient_Call) Run(run func(ctx context.Context, channelID string, clientID string)) *MockConfigRepository_ConnectClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockConfigRepository_ConnectClient_Call) Return(err error) *MockConfigRepository_ConnectClient_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockConfigRepository_ConnectClient_Call) RunAndReturn(run func(ctx context.Context, channelID string, clientID string) error) *MockConfigRepository_ConnectClient_Call {
	_c.Call.Return(run)
	return _c
}

// DisconnectClient provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) DisconnectClient(ctx context.Context, channelID string, clientID string) error {
	ret := _mock.Called(ctx, channelID, clientID)

	if len(ret) == 0 {
		panic("no return value specified for DisconnectClient")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = returnFunc(ctx, channelID, clientID)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockConfigRepository_DisconnectClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DisconnectClient'
type MockConfigRepository_DisconnectClient_Call struct {
	*mock.Call
}

// DisconnectClient is a helper method to define mock.On call
//   - ctx
//   - channelID
//   - clientID
func (_e *MockConfigRepository_Expecter) DisconnectClient(ctx interface{}, channelID interface{}, clientID interface{}) *MockConfigRepository_DisconnectClient_Call {
	return &MockConfigRepository_DisconnectClient_Call{Call: _e.mock.On("DisconnectClient", ctx, channelID, clientID)}
}

func (_c *MockConfigRepository_DisconnectClient_Call) Run(run func(ctx context.Context, channelID string, clientID string)) *MockConfigRepository_DisconnectClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockConfigRepository_DisconnectClient_Call) Return(err error) *MockConfigRepository_DisconnectClient_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockConfigRepository_DisconnectClient_Call) RunAndReturn(run func(ctx context.Context, channelID string, clientID string) error) *MockConfigRepository_DisconnectClient_Call {
	_c.Call.Return(run)
	return _c
}

// ListExisting provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) ListExisting(ctx context.Context, domainID string, ids []string) ([]bootstrap.Channel, error) {
	ret := _mock.Called(ctx, domainID, ids)

	if len(ret) == 0 {
		panic("no return value specified for ListExisting")
	}

	var r0 []bootstrap.Channel
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, []string) ([]bootstrap.Channel, error)); ok {
		return returnFunc(ctx, domainID, ids)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, []string) []bootstrap.Channel); ok {
		r0 = returnFunc(ctx, domainID, ids)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]bootstrap.Channel)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, []string) error); ok {
		r1 = returnFunc(ctx, domainID, ids)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockConfigRepository_ListExisting_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListExisting'
type MockConfigRepository_ListExisting_Call struct {
	*mock.Call
}

// ListExisting is a helper method to define mock.On call
//   - ctx
//   - domainID
//   - ids
func (_e *MockConfigRepository_Expecter) ListExisting(ctx interface{}, domainID interface{}, ids interface{}) *MockConfigRepository_ListExisting_Call {
	return &MockConfigRepository_ListExisting_Call{Call: _e.mock.On("ListExisting", ctx, domainID, ids)}
}

func (_c *MockConfigRepository_ListExisting_Call) Run(run func(ctx context.Context, domainID string, ids []string)) *MockConfigRepository_ListExisting_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([]string))
	})
	return _c
}

func (_c *MockConfigRepository_ListExisting_Call) Return(channels []bootstrap.Channel, err error) *MockConfigRepository_ListExisting_Call {
	_c.Call.Return(channels, err)
	return _c
}

func (_c *MockConfigRepository_ListExisting_Call) RunAndReturn(run func(ctx context.Context, domainID string, ids []string) ([]bootstrap.Channel, error)) *MockConfigRepository_ListExisting_Call {
	_c.Call.Return(run)
	return _c
}

// Remove provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) Remove(ctx context.Context, domainID string, id string) error {
	ret := _mock.Called(ctx, domainID, id)

	if len(ret) == 0 {
		panic("no return value specified for Remove")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = returnFunc(ctx, domainID, id)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockConfigRepository_Remove_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Remove'
type MockConfigRepository_Remove_Call struct {
	*mock.Call
}

// Remove is a helper method to define mock.On call
//   - ctx
//   - domainID
//   - id
func (_e *MockConfigRepository_Expecter) Remove(ctx interface{}, domainID interface{}, id interface{}) *MockConfigRepository_Remove_Call {
	return &MockConfigRepository_Remove_Call{Call: _e.mock.On("Remove", ctx, domainID, id)}
}

func (_c *MockConfigRepository_Remove_Call) Run(run func(ctx context.Context, domainID string, id string)) *MockConfigRepository_Remove_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockConfigRepository_Remove_Call) Return(err error) *MockConfigRepository_Remove_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockConfigRepository_Remove_Call) RunAndReturn(run func(ctx context.Context, domainID string, id string) error) *MockConfigRepository_Remove_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveChannel provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) RemoveChannel(ctx context.Context, id string) error {
	ret := _mock.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for RemoveChannel")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = returnFunc(ctx, id)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockConfigRepository_RemoveChannel_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveChannel'
type MockConfigRepository_RemoveChannel_Call struct {
	*mock.Call
}

// RemoveChannel is a helper method to define mock.On call
//   - ctx
//   - id
func (_e *MockConfigRepository_Expecter) RemoveChannel(ctx interface{}, id interface{}) *MockConfigRepository_RemoveChannel_Call {
	return &MockConfigRepository_RemoveChannel_Call{Call: _e.mock.On("RemoveChannel", ctx, id)}
}

func (_c *MockConfigRepository_RemoveChannel_Call) Run(run func(ctx context.Context, id string)) *MockConfigRepository_RemoveChannel_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockConfigRepository_RemoveChannel_Call) Return(err error) *MockConfigRepository_RemoveChannel_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockConfigRepository_RemoveChannel_Call) RunAndReturn(run func(ctx context.Context, id string) error) *MockConfigRepository_RemoveChannel_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveClient provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) RemoveClient(ctx context.Context, id string) error {
	ret := _mock.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for RemoveClient")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = returnFunc(ctx, id)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockConfigRepository_RemoveClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveClient'
type MockConfigRepository_RemoveClient_Call struct {
	*mock.Call
}

// RemoveClient is a helper method to define mock.On call
//   - ctx
//   - id
func (_e *MockConfigRepository_Expecter) RemoveClient(ctx interface{}, id interface{}) *MockConfigRepository_RemoveClient_Call {
	return &MockConfigRepository_RemoveClient_Call{Call: _e.mock.On("RemoveClient", ctx, id)}
}

func (_c *MockConfigRepository_RemoveClient_Call) Run(run func(ctx context.Context, id string)) *MockConfigRepository_RemoveClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockConfigRepository_RemoveClient_Call) Return(err error) *MockConfigRepository_RemoveClient_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockConfigRepository_RemoveClient_Call) RunAndReturn(run func(ctx context.Context, id string) error) *MockConfigRepository_RemoveClient_Call {
	_c.Call.Return(run)
	return _c
}

// RetrieveAll provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) RetrieveAll(ctx context.Context, domainID string, clientIDs []string, filter bootstrap.Filter, offset uint64, limit uint64) bootstrap.ConfigsPage {
	ret := _mock.Called(ctx, domainID, clientIDs, filter, offset, limit)

	if len(ret) == 0 {
		panic("no return value specified for RetrieveAll")
	}

	var r0 bootstrap.ConfigsPage
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, []string, bootstrap.Filter, uint64, uint64) bootstrap.ConfigsPage); ok {
		r0 = returnFunc(ctx, domainID, clientIDs, filter, offset, limit)
	} else {
		r0 = ret.Get(0).(bootstrap.ConfigsPage)
	}
	return r0
}

// MockConfigRepository_RetrieveAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RetrieveAll'
type MockConfigRepository_RetrieveAll_Call struct {
	*mock.Call
}

// RetrieveAll is a helper method to define mock.On call
//   - ctx
//   - domainID
//   - clientIDs
//   - filter
//   - offset
//   - limit
func (_e *MockConfigRepository_Expecter) RetrieveAll(ctx interface{}, domainID interface{}, clientIDs interface{}, filter interface{}, offset interface{}, limit interface{}) *MockConfigRepository_RetrieveAll_Call {
	return &MockConfigRepository_RetrieveAll_Call{Call: _e.mock.On("RetrieveAll", ctx, domainID, clientIDs, filter, offset, limit)}
}

func (_c *MockConfigRepository_RetrieveAll_Call) Run(run func(ctx context.Context, domainID string, clientIDs []string, filter bootstrap.Filter, offset uint64, limit uint64)) *MockConfigRepository_RetrieveAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([]string), args[3].(bootstrap.Filter), args[4].(uint64), args[5].(uint64))
	})
	return _c
}

func (_c *MockConfigRepository_RetrieveAll_Call) Return(configsPage bootstrap.ConfigsPage) *MockConfigRepository_RetrieveAll_Call {
	_c.Call.Return(configsPage)
	return _c
}

func (_c *MockConfigRepository_RetrieveAll_Call) RunAndReturn(run func(ctx context.Context, domainID string, clientIDs []string, filter bootstrap.Filter, offset uint64, limit uint64) bootstrap.ConfigsPage) *MockConfigRepository_RetrieveAll_Call {
	_c.Call.Return(run)
	return _c
}

// RetrieveByExternalID provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) RetrieveByExternalID(ctx context.Context, externalID string) (bootstrap.Config, error) {
	ret := _mock.Called(ctx, externalID)

	if len(ret) == 0 {
		panic("no return value specified for RetrieveByExternalID")
	}

	var r0 bootstrap.Config
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string) (bootstrap.Config, error)); ok {
		return returnFunc(ctx, externalID)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string) bootstrap.Config); ok {
		r0 = returnFunc(ctx, externalID)
	} else {
		r0 = ret.Get(0).(bootstrap.Config)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = returnFunc(ctx, externalID)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockConfigRepository_RetrieveByExternalID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RetrieveByExternalID'
type MockConfigRepository_RetrieveByExternalID_Call struct {
	*mock.Call
}

// RetrieveByExternalID is a helper method to define mock.On call
//   - ctx
//   - externalID
func (_e *MockConfigRepository_Expecter) RetrieveByExternalID(ctx interface{}, externalID interface{}) *MockConfigRepository_RetrieveByExternalID_Call {
	return &MockConfigRepository_RetrieveByExternalID_Call{Call: _e.mock.On("RetrieveByExternalID", ctx, externalID)}
}

func (_c *MockConfigRepository_RetrieveByExternalID_Call) Run(run func(ctx context.Context, externalID string)) *MockConfigRepository_RetrieveByExternalID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockConfigRepository_RetrieveByExternalID_Call) Return(config bootstrap.Config, err error) *MockConfigRepository_RetrieveByExternalID_Call {
	_c.Call.Return(config, err)
	return _c
}

func (_c *MockConfigRepository_RetrieveByExternalID_Call) RunAndReturn(run func(ctx context.Context, externalID string) (bootstrap.Config, error)) *MockConfigRepository_RetrieveByExternalID_Call {
	_c.Call.Return(run)
	return _c
}

// RetrieveByID provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) RetrieveByID(ctx context.Context, domainID string, id string) (bootstrap.Config, error) {
	ret := _mock.Called(ctx, domainID, id)

	if len(ret) == 0 {
		panic("no return value specified for RetrieveByID")
	}

	var r0 bootstrap.Config
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) (bootstrap.Config, error)); ok {
		return returnFunc(ctx, domainID, id)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string) bootstrap.Config); ok {
		r0 = returnFunc(ctx, domainID, id)
	} else {
		r0 = ret.Get(0).(bootstrap.Config)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = returnFunc(ctx, domainID, id)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockConfigRepository_RetrieveByID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RetrieveByID'
type MockConfigRepository_RetrieveByID_Call struct {
	*mock.Call
}

// RetrieveByID is a helper method to define mock.On call
//   - ctx
//   - domainID
//   - id
func (_e *MockConfigRepository_Expecter) RetrieveByID(ctx interface{}, domainID interface{}, id interface{}) *MockConfigRepository_RetrieveByID_Call {
	return &MockConfigRepository_RetrieveByID_Call{Call: _e.mock.On("RetrieveByID", ctx, domainID, id)}
}

func (_c *MockConfigRepository_RetrieveByID_Call) Run(run func(ctx context.Context, domainID string, id string)) *MockConfigRepository_RetrieveByID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockConfigRepository_RetrieveByID_Call) Return(config bootstrap.Config, err error) *MockConfigRepository_RetrieveByID_Call {
	_c.Call.Return(config, err)
	return _c
}

func (_c *MockConfigRepository_RetrieveByID_Call) RunAndReturn(run func(ctx context.Context, domainID string, id string) (bootstrap.Config, error)) *MockConfigRepository_RetrieveByID_Call {
	_c.Call.Return(run)
	return _c
}

// Save provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) Save(ctx context.Context, cfg bootstrap.Config, chsConnIDs []string) (string, error) {
	ret := _mock.Called(ctx, cfg, chsConnIDs)

	if len(ret) == 0 {
		panic("no return value specified for Save")
	}

	var r0 string
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, bootstrap.Config, []string) (string, error)); ok {
		return returnFunc(ctx, cfg, chsConnIDs)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, bootstrap.Config, []string) string); ok {
		r0 = returnFunc(ctx, cfg, chsConnIDs)
	} else {
		r0 = ret.Get(0).(string)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, bootstrap.Config, []string) error); ok {
		r1 = returnFunc(ctx, cfg, chsConnIDs)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockConfigRepository_Save_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Save'
type MockConfigRepository_Save_Call struct {
	*mock.Call
}

// Save is a helper method to define mock.On call
//   - ctx
//   - cfg
//   - chsConnIDs
func (_e *MockConfigRepository_Expecter) Save(ctx interface{}, cfg interface{}, chsConnIDs interface{}) *MockConfigRepository_Save_Call {
	return &MockConfigRepository_Save_Call{Call: _e.mock.On("Save", ctx, cfg, chsConnIDs)}
}

func (_c *MockConfigRepository_Save_Call) Run(run func(ctx context.Context, cfg bootstrap.Config, chsConnIDs []string)) *MockConfigRepository_Save_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(bootstrap.Config), args[2].([]string))
	})
	return _c
}

func (_c *MockConfigRepository_Save_Call) Return(s string, err error) *MockConfigRepository_Save_Call {
	_c.Call.Return(s, err)
	return _c
}

func (_c *MockConfigRepository_Save_Call) RunAndReturn(run func(ctx context.Context, cfg bootstrap.Config, chsConnIDs []string) (string, error)) *MockConfigRepository_Save_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) Update(ctx context.Context, cfg bootstrap.Config) error {
	ret := _mock.Called(ctx, cfg)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, bootstrap.Config) error); ok {
		r0 = returnFunc(ctx, cfg)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockConfigRepository_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type MockConfigRepository_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx
//   - cfg
func (_e *MockConfigRepository_Expecter) Update(ctx interface{}, cfg interface{}) *MockConfigRepository_Update_Call {
	return &MockConfigRepository_Update_Call{Call: _e.mock.On("Update", ctx, cfg)}
}

func (_c *MockConfigRepository_Update_Call) Run(run func(ctx context.Context, cfg bootstrap.Config)) *MockConfigRepository_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(bootstrap.Config))
	})
	return _c
}

func (_c *MockConfigRepository_Update_Call) Return(err error) *MockConfigRepository_Update_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockConfigRepository_Update_Call) RunAndReturn(run func(ctx context.Context, cfg bootstrap.Config) error) *MockConfigRepository_Update_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateCert provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) UpdateCert(ctx context.Context, domainID string, clientID string, clientCert string, clientKey string, caCert string) (bootstrap.Config, error) {
	ret := _mock.Called(ctx, domainID, clientID, clientCert, clientKey, caCert)

	if len(ret) == 0 {
		panic("no return value specified for UpdateCert")
	}

	var r0 bootstrap.Config
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, string, string, string) (bootstrap.Config, error)); ok {
		return returnFunc(ctx, domainID, clientID, clientCert, clientKey, caCert)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, string, string, string) bootstrap.Config); ok {
		r0 = returnFunc(ctx, domainID, clientID, clientCert, clientKey, caCert)
	} else {
		r0 = ret.Get(0).(bootstrap.Config)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, string, string, string, string, string) error); ok {
		r1 = returnFunc(ctx, domainID, clientID, clientCert, clientKey, caCert)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockConfigRepository_UpdateCert_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateCert'
type MockConfigRepository_UpdateCert_Call struct {
	*mock.Call
}

// UpdateCert is a helper method to define mock.On call
//   - ctx
//   - domainID
//   - clientID
//   - clientCert
//   - clientKey
//   - caCert
func (_e *MockConfigRepository_Expecter) UpdateCert(ctx interface{}, domainID interface{}, clientID interface{}, clientCert interface{}, clientKey interface{}, caCert interface{}) *MockConfigRepository_UpdateCert_Call {
	return &MockConfigRepository_UpdateCert_Call{Call: _e.mock.On("UpdateCert", ctx, domainID, clientID, clientCert, clientKey, caCert)}
}

func (_c *MockConfigRepository_UpdateCert_Call) Run(run func(ctx context.Context, domainID string, clientID string, clientCert string, clientKey string, caCert string)) *MockConfigRepository_UpdateCert_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(string), args[4].(string), args[5].(string))
	})
	return _c
}

func (_c *MockConfigRepository_UpdateCert_Call) Return(config bootstrap.Config, err error) *MockConfigRepository_UpdateCert_Call {
	_c.Call.Return(config, err)
	return _c
}

func (_c *MockConfigRepository_UpdateCert_Call) RunAndReturn(run func(ctx context.Context, domainID string, clientID string, clientCert string, clientKey string, caCert string) (bootstrap.Config, error)) *MockConfigRepository_UpdateCert_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateChannel provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) UpdateChannel(ctx context.Context, c bootstrap.Channel) error {
	ret := _mock.Called(ctx, c)

	if len(ret) == 0 {
		panic("no return value specified for UpdateChannel")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, bootstrap.Channel) error); ok {
		r0 = returnFunc(ctx, c)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockConfigRepository_UpdateChannel_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateChannel'
type MockConfigRepository_UpdateChannel_Call struct {
	*mock.Call
}

// UpdateChannel is a helper method to define mock.On call
//   - ctx
//   - c
func (_e *MockConfigRepository_Expecter) UpdateChannel(ctx interface{}, c interface{}) *MockConfigRepository_UpdateChannel_Call {
	return &MockConfigRepository_UpdateChannel_Call{Call: _e.mock.On("UpdateChannel", ctx, c)}
}

func (_c *MockConfigRepository_UpdateChannel_Call) Run(run func(ctx context.Context, c bootstrap.Channel)) *MockConfigRepository_UpdateChannel_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(bootstrap.Channel))
	})
	return _c
}

func (_c *MockConfigRepository_UpdateChannel_Call) Return(err error) *MockConfigRepository_UpdateChannel_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockConfigRepository_UpdateChannel_Call) RunAndReturn(run func(ctx context.Context, c bootstrap.Channel) error) *MockConfigRepository_UpdateChannel_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateConnections provides a mock function for the type MockConfigRepository
func (_mock *MockConfigRepository) UpdateConnections(ctx context.Context, domainID string, id string, channels []bootstrap.Channel, connections []string) error {
	ret := _mock.Called(ctx, domainID, id, channels, connections)

	if len(ret) == 0 {
		panic("no return value specified for UpdateConnections")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, string, string, []bootstrap.Channel, []string) error); ok {
		r0 = returnFunc(ctx, domainID, id, channels, connections)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// MockConfigRepository_UpdateConnections_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateConnections'
type MockConfigRepository_UpdateConnections_Call struct {
	*mock.Call
}

// UpdateConnections is a helper method to define mock.On call
//   - ctx
//   - domainID
//   - id
//   - channels
//   - connections
func (_e *MockConfigRepository_Expecter) UpdateConnections(ctx interface{}, domainID interface{}, id interface{}, channels interface{}, connections interface{}) *MockConfigRepository_UpdateConnections_Call {
	return &MockConfigRepository_UpdateConnections_Call{Call: _e.mock.On("UpdateConnections", ctx, domainID, id, channels, connections)}
}

func (_c *MockConfigRepository_UpdateConnections_Call) Run(run func(ctx context.Context, domainID string, id string, channels []bootstrap.Channel, connections []string)) *MockConfigRepository_UpdateConnections_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].([]bootstrap.Channel), args[4].([]string))
	})
	return _c
}

func (_c *MockConfigRepository_UpdateConnections_Call) Return(err error) *MockConfigRepository_UpdateConnections_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *MockConfigRepository_UpdateConnections_Call) RunAndReturn(run func(ctx context.Context, domainID string, id string, channels []bootstrap.Channel, connections []string) error) *MockConfigRepository_UpdateConnections_Call {
	_c.Call.Return(run)
	return _c
}
