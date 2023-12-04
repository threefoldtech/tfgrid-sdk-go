// Code generated by MockGen. DO NOT EDIT.
// Source: models/substrate.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	gomock "github.com/golang/mock/gomock"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

// MockSub is a mock of Sub interface.
type MockSub struct {
	ctrl     *gomock.Controller
	recorder *MockSubMockRecorder
}

// MockSubMockRecorder is the mock recorder for MockSub.
type MockSubMockRecorder struct {
	mock *MockSub
}

// NewMockSub creates a new mock instance.
func NewMockSub(ctrl *gomock.Controller) *MockSub {
	mock := &MockSub{ctrl: ctrl}
	mock.recorder = &MockSubMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSub) EXPECT() *MockSubMockRecorder {
	return m.recorder
}

// GetDedicatedNodePrice mocks base method.
func (m *MockSub) GetDedicatedNodePrice(nodeID uint32) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDedicatedNodePrice", nodeID)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDedicatedNodePrice indicates an expected call of GetDedicatedNodePrice.
func (mr *MockSubMockRecorder) GetDedicatedNodePrice(nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDedicatedNodePrice", reflect.TypeOf((*MockSub)(nil).GetDedicatedNodePrice), nodeID)
}

// GetFarm mocks base method.
func (m *MockSub) GetFarm(id uint32) (*substrate.Farm, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFarm", id)
	ret0, _ := ret[0].(*substrate.Farm)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFarm indicates an expected call of GetFarm.
func (mr *MockSubMockRecorder) GetFarm(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFarm", reflect.TypeOf((*MockSub)(nil).GetFarm), id)
}

// GetNode mocks base method.
func (m *MockSub) GetNode(nodeID uint32) (*substrate.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNode", nodeID)
	ret0, _ := ret[0].(*substrate.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNode indicates an expected call of GetNode.
func (mr *MockSubMockRecorder) GetNode(nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNode", reflect.TypeOf((*MockSub)(nil).GetNode), nodeID)
}

// GetNodeRentContract mocks base method.
func (m *MockSub) GetNodeRentContract(nodeID uint32) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeRentContract", nodeID)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodeRentContract indicates an expected call of GetNodeRentContract.
func (mr *MockSubMockRecorder) GetNodeRentContract(nodeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeRentContract", reflect.TypeOf((*MockSub)(nil).GetNodeRentContract), nodeID)
}

// GetNodes mocks base method.
func (m *MockSub) GetNodes(farmID uint32) ([]uint32, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodes", farmID)
	ret0, _ := ret[0].([]uint32)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodes indicates an expected call of GetNodes.
func (mr *MockSubMockRecorder) GetNodes(farmID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodes", reflect.TypeOf((*MockSub)(nil).GetNodes), farmID)
}

// SetNodePowerTarget mocks base method.
func (m *MockSub) SetNodePowerTarget(identity substrate.Identity, nodeID uint32, up bool) (types.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetNodePowerTarget", identity, nodeID, up)
	ret0, _ := ret[0].(types.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetNodePowerTarget indicates an expected call of SetNodePowerTarget.
func (mr *MockSubMockRecorder) SetNodePowerTarget(identity, nodeID, up interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetNodePowerTarget", reflect.TypeOf((*MockSub)(nil).SetNodePowerTarget), identity, nodeID, up)
}