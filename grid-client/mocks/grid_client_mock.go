// Code generated by MockGen. DO NOT EDIT.
// Source: ./grid-proxy/pkg/client/grid_client.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

// MockDBClient is a mock of DBClient interface.
type MockDBClient struct {
	ctrl     *gomock.Controller
	recorder *MockDBClientMockRecorder
}

// MockDBClientMockRecorder is the mock recorder for MockDBClient.
type MockDBClientMockRecorder struct {
	mock *MockDBClient
}

// NewMockDBClient creates a new mock instance.
func NewMockDBClient(ctrl *gomock.Controller) *MockDBClient {
	mock := &MockDBClient{ctrl: ctrl}
	mock.recorder = &MockDBClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDBClient) EXPECT() *MockDBClientMockRecorder {
	return m.recorder
}

// Contract mocks base method.
func (m *MockDBClient) Contract(ctx context.Context, contractID uint32) (types.Contract, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Contract", ctx, contractID)
	ret0, _ := ret[0].(types.Contract)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Contract indicates an expected call of Contract.
func (mr *MockDBClientMockRecorder) Contract(ctx, contractID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Contract", reflect.TypeOf((*MockDBClient)(nil).Contract), ctx, contractID)
}

// ContractBills mocks base method.
func (m *MockDBClient) ContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]types.ContractBilling, uint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ContractBills", ctx, contractID, limit)
	ret0, _ := ret[0].([]types.ContractBilling)
	ret1, _ := ret[1].(uint)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ContractBills indicates an expected call of ContractBills.
func (mr *MockDBClientMockRecorder) ContractBills(ctx, contractID, limit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ContractBills", reflect.TypeOf((*MockDBClient)(nil).ContractBills), ctx, contractID, limit)
}

// Contracts mocks base method.
func (m *MockDBClient) Contracts(ctx context.Context, filter types.ContractFilter, pagination types.Limit) ([]types.Contract, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Contracts", ctx, filter, pagination)
	ret0, _ := ret[0].([]types.Contract)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Contracts indicates an expected call of Contracts.
func (mr *MockDBClientMockRecorder) Contracts(ctx, filter, pagination interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Contracts", reflect.TypeOf((*MockDBClient)(nil).Contracts), ctx, filter, pagination)
}

// Counters mocks base method.
func (m *MockDBClient) Counters(ctx context.Context, filter types.StatsFilter) (types.Counters, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Counters", ctx, filter)
	ret0, _ := ret[0].(types.Counters)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Counters indicates an expected call of Counters.
func (mr *MockDBClientMockRecorder) Counters(ctx, filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Counters", reflect.TypeOf((*MockDBClient)(nil).Counters), ctx, filter)
}

// Farms mocks base method.
func (m *MockDBClient) Farms(ctx context.Context, filter types.FarmFilter, pagination types.Limit) ([]types.Farm, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Farms", ctx, filter, pagination)
	ret0, _ := ret[0].([]types.Farm)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Farms indicates an expected call of Farms.
func (mr *MockDBClientMockRecorder) Farms(ctx, filter, pagination interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Farms", reflect.TypeOf((*MockDBClient)(nil).Farms), ctx, filter, pagination)
}

// Node mocks base method.
func (m *MockDBClient) Node(ctx context.Context, nodeID uint32) (types.NodeWithNestedCapacity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Node", ctx, nodeID)
	ret0, _ := ret[0].(types.NodeWithNestedCapacity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Node indicates an expected call of Node.
func (mr *MockDBClientMockRecorder) Node(ctx, nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Node", reflect.TypeOf((*MockDBClient)(nil).Node), ctx, nodeID)
}

// NodeStatus mocks base method.
func (m *MockDBClient) NodeStatus(ctx context.Context, nodeID uint32) (types.NodeStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NodeStatus", ctx, nodeID)
	ret0, _ := ret[0].(types.NodeStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NodeStatus indicates an expected call of NodeStatus.
func (mr *MockDBClientMockRecorder) NodeStatus(ctx, nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NodeStatus", reflect.TypeOf((*MockDBClient)(nil).NodeStatus), ctx, nodeID)
}

// Nodes mocks base method.
func (m *MockDBClient) Nodes(ctx context.Context, filter types.NodeFilter, pagination types.Limit) ([]types.Node, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Nodes", ctx, filter, pagination)
	ret0, _ := ret[0].([]types.Node)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Nodes indicates an expected call of Nodes.
func (mr *MockDBClientMockRecorder) Nodes(ctx, filter, pagination interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Nodes", reflect.TypeOf((*MockDBClient)(nil).Nodes), ctx, filter, pagination)
}

// Twins mocks base method.
func (m *MockDBClient) Twins(ctx context.Context, filter types.TwinFilter, pagination types.Limit) ([]types.Twin, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Twins", ctx, filter, pagination)
	ret0, _ := ret[0].([]types.Twin)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Twins indicates an expected call of Twins.
func (mr *MockDBClientMockRecorder) Twins(ctx, filter, pagination interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Twins", reflect.TypeOf((*MockDBClient)(nil).Twins), ctx, filter, pagination)
}

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// Contract mocks base method.
func (m *MockClient) Contract(ctx context.Context, contractID uint32) (types.Contract, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Contract", ctx, contractID)
	ret0, _ := ret[0].(types.Contract)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Contract indicates an expected call of Contract.
func (mr *MockClientMockRecorder) Contract(ctx, contractID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Contract", reflect.TypeOf((*MockClient)(nil).Contract), ctx, contractID)
}

// ContractBills mocks base method.
func (m *MockClient) ContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]types.ContractBilling, uint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ContractBills", ctx, contractID, limit)
	ret0, _ := ret[0].([]types.ContractBilling)
	ret1, _ := ret[1].(uint)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ContractBills indicates an expected call of ContractBills.
func (mr *MockClientMockRecorder) ContractBills(ctx, contractID, limit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ContractBills", reflect.TypeOf((*MockClient)(nil).ContractBills), ctx, contractID, limit)
}

// Contracts mocks base method.
func (m *MockClient) Contracts(ctx context.Context, filter types.ContractFilter, pagination types.Limit) ([]types.Contract, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Contracts", ctx, filter, pagination)
	ret0, _ := ret[0].([]types.Contract)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Contracts indicates an expected call of Contracts.
func (mr *MockClientMockRecorder) Contracts(ctx, filter, pagination interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Contracts", reflect.TypeOf((*MockClient)(nil).Contracts), ctx, filter, pagination)
}

// Counters mocks base method.
func (m *MockClient) Counters(ctx context.Context, filter types.StatsFilter) (types.Counters, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Counters", ctx, filter)
	ret0, _ := ret[0].(types.Counters)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Counters indicates an expected call of Counters.
func (mr *MockClientMockRecorder) Counters(ctx, filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Counters", reflect.TypeOf((*MockClient)(nil).Counters), ctx, filter)
}

// Farms mocks base method.
func (m *MockClient) Farms(ctx context.Context, filter types.FarmFilter, pagination types.Limit) ([]types.Farm, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Farms", ctx, filter, pagination)
	ret0, _ := ret[0].([]types.Farm)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Farms indicates an expected call of Farms.
func (mr *MockClientMockRecorder) Farms(ctx, filter, pagination interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Farms", reflect.TypeOf((*MockClient)(nil).Farms), ctx, filter, pagination)
}

// Node mocks base method.
func (m *MockClient) Node(ctx context.Context, nodeID uint32) (types.NodeWithNestedCapacity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Node", ctx, nodeID)
	ret0, _ := ret[0].(types.NodeWithNestedCapacity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Node indicates an expected call of Node.
func (mr *MockClientMockRecorder) Node(ctx, nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Node", reflect.TypeOf((*MockClient)(nil).Node), ctx, nodeID)
}

// NodeStatus mocks base method.
func (m *MockClient) NodeStatus(ctx context.Context, nodeID uint32) (types.NodeStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NodeStatus", ctx, nodeID)
	ret0, _ := ret[0].(types.NodeStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NodeStatus indicates an expected call of NodeStatus.
func (mr *MockClientMockRecorder) NodeStatus(ctx, nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NodeStatus", reflect.TypeOf((*MockClient)(nil).NodeStatus), ctx, nodeID)
}

// Nodes mocks base method.
func (m *MockClient) Nodes(ctx context.Context, filter types.NodeFilter, pagination types.Limit) ([]types.Node, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Nodes", ctx, filter, pagination)
	ret0, _ := ret[0].([]types.Node)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Nodes indicates an expected call of Nodes.
func (mr *MockClientMockRecorder) Nodes(ctx, filter, pagination interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Nodes", reflect.TypeOf((*MockClient)(nil).Nodes), ctx, filter, pagination)
}

// Ping mocks base method.
func (m *MockClient) Ping() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Ping")
	ret0, _ := ret[0].(error)
	return ret0
}

// Ping indicates an expected call of Ping.
func (mr *MockClientMockRecorder) Ping() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ping", reflect.TypeOf((*MockClient)(nil).Ping))
}

// Twins mocks base method.
func (m *MockClient) Twins(ctx context.Context, filter types.TwinFilter, pagination types.Limit) ([]types.Twin, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Twins", ctx, filter, pagination)
	ret0, _ := ret[0].([]types.Twin)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Twins indicates an expected call of Twins.
func (mr *MockClientMockRecorder) Twins(ctx, filter, pagination interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Twins", reflect.TypeOf((*MockClient)(nil).Twins), ctx, filter, pagination)
}
