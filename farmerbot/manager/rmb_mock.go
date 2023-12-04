// Code generated by MockGen. DO NOT EDIT.
// Source: manager/rmb.go

// Package manager is a generated GoMock package.
package manager

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
	pkg "github.com/threefoldtech/zos/pkg"
)

// MockRMB is a mock of RMB interface.
type MockRMB struct {
	ctrl     *gomock.Controller
	recorder *MockRMBMockRecorder
}

// MockRMBMockRecorder is the mock recorder for MockRMB.
type MockRMBMockRecorder struct {
	mock *MockRMB
}

// NewMockRMB creates a new mock instance.
func NewMockRMB(ctrl *gomock.Controller) *MockRMB {
	mock := &MockRMB{ctrl: ctrl}
	mock.recorder = &MockRMBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRMB) EXPECT() *MockRMBMockRecorder {
	return m.recorder
}

// GetStoragePools mocks base method.
func (m *MockRMB) GetStoragePools(ctx context.Context, nodeTwin uint32) ([]pkg.PoolMetrics, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStoragePools", ctx, nodeTwin)
	ret0, _ := ret[0].([]pkg.PoolMetrics)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStoragePools indicates an expected call of GetStoragePools.
func (mr *MockRMBMockRecorder) GetStoragePools(ctx, nodeTwin interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStoragePools", reflect.TypeOf((*MockRMB)(nil).GetStoragePools), ctx, nodeTwin)
}

// ListGPUs mocks base method.
func (m *MockRMB) ListGPUs(ctx context.Context, nodeTwin uint32) ([]models.GPU, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListGPUs", ctx, nodeTwin)
	ret0, _ := ret[0].([]models.GPU)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListGPUs indicates an expected call of ListGPUs.
func (mr *MockRMBMockRecorder) ListGPUs(ctx, nodeTwin interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListGPUs", reflect.TypeOf((*MockRMB)(nil).ListGPUs), ctx, nodeTwin)
}

// Statistics mocks base method.
func (m *MockRMB) Statistics(ctx context.Context, nodeTwin uint32) (models.ZosResourcesStatistics, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Statistics", ctx, nodeTwin)
	ret0, _ := ret[0].(models.ZosResourcesStatistics)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Statistics indicates an expected call of Statistics.
func (mr *MockRMBMockRecorder) Statistics(ctx, nodeTwin interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Statistics", reflect.TypeOf((*MockRMB)(nil).Statistics), ctx, nodeTwin)
}

// SystemVersion mocks base method.
func (m *MockRMB) SystemVersion(ctx context.Context, nodeTwin uint32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SystemVersion", ctx, nodeTwin)
	ret0, _ := ret[0].(error)
	return ret0
}

// SystemVersion indicates an expected call of SystemVersion.
func (mr *MockRMBMockRecorder) SystemVersion(ctx, nodeTwin interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SystemVersion", reflect.TypeOf((*MockRMB)(nil).SystemVersion), ctx, nodeTwin)
}