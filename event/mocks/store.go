// Code generated by MockGen. DO NOT EDIT.
// Source: store.go

// Package mock_event is a generated GoMock package.
package mock_event

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	uuid "github.com/google/uuid"
	event "github.com/modernice/goes/event"
	time "github.com/modernice/goes/event/query/time"
	version "github.com/modernice/goes/event/query/version"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// Delete mocks base method.
func (m *MockStore) Delete(arg0 context.Context, arg1 event.Event) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockStoreMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockStore)(nil).Delete), arg0, arg1)
}

// Find mocks base method.
func (m *MockStore) Find(arg0 context.Context, arg1 uuid.UUID) (event.Event, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Find", arg0, arg1)
	ret0, _ := ret[0].(event.Event)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Find indicates an expected call of Find.
func (mr *MockStoreMockRecorder) Find(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Find", reflect.TypeOf((*MockStore)(nil).Find), arg0, arg1)
}

// Insert mocks base method.
func (m *MockStore) Insert(arg0 context.Context, arg1 ...event.Event) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Insert", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Insert indicates an expected call of Insert.
func (mr *MockStoreMockRecorder) Insert(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Insert", reflect.TypeOf((*MockStore)(nil).Insert), varargs...)
}

// Query mocks base method.
func (m *MockStore) Query(arg0 context.Context, arg1 event.Query) (<-chan event.Event, <-chan error, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Query", arg0, arg1)
	ret0, _ := ret[0].(<-chan event.Event)
	ret1, _ := ret[1].(<-chan error)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Query indicates an expected call of Query.
func (mr *MockStoreMockRecorder) Query(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockStore)(nil).Query), arg0, arg1)
}

// MockQuery is a mock of Query interface.
type MockQuery struct {
	ctrl     *gomock.Controller
	recorder *MockQueryMockRecorder
}

// MockQueryMockRecorder is the mock recorder for MockQuery.
type MockQueryMockRecorder struct {
	mock *MockQuery
}

// NewMockQuery creates a new mock instance.
func NewMockQuery(ctrl *gomock.Controller) *MockQuery {
	mock := &MockQuery{ctrl: ctrl}
	mock.recorder = &MockQueryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQuery) EXPECT() *MockQueryMockRecorder {
	return m.recorder
}

// AggregateIDs mocks base method.
func (m *MockQuery) AggregateIDs() []uuid.UUID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AggregateIDs")
	ret0, _ := ret[0].([]uuid.UUID)
	return ret0
}

// AggregateIDs indicates an expected call of AggregateIDs.
func (mr *MockQueryMockRecorder) AggregateIDs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AggregateIDs", reflect.TypeOf((*MockQuery)(nil).AggregateIDs))
}

// AggregateNames mocks base method.
func (m *MockQuery) AggregateNames() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AggregateNames")
	ret0, _ := ret[0].([]string)
	return ret0
}

// AggregateNames indicates an expected call of AggregateNames.
func (mr *MockQueryMockRecorder) AggregateNames() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AggregateNames", reflect.TypeOf((*MockQuery)(nil).AggregateNames))
}

// AggregateVersions mocks base method.
func (m *MockQuery) AggregateVersions() version.Constraints {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AggregateVersions")
	ret0, _ := ret[0].(version.Constraints)
	return ret0
}

// AggregateVersions indicates an expected call of AggregateVersions.
func (mr *MockQueryMockRecorder) AggregateVersions() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AggregateVersions", reflect.TypeOf((*MockQuery)(nil).AggregateVersions))
}

// Aggregates mocks base method.
func (m *MockQuery) Aggregates() []event.AggregateTuple {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Aggregates")
	ret0, _ := ret[0].([]event.AggregateTuple)
	return ret0
}

// Aggregates indicates an expected call of Aggregates.
func (mr *MockQueryMockRecorder) Aggregates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Aggregates", reflect.TypeOf((*MockQuery)(nil).Aggregates))
}

// IDs mocks base method.
func (m *MockQuery) IDs() []uuid.UUID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IDs")
	ret0, _ := ret[0].([]uuid.UUID)
	return ret0
}

// IDs indicates an expected call of IDs.
func (mr *MockQueryMockRecorder) IDs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IDs", reflect.TypeOf((*MockQuery)(nil).IDs))
}

// Names mocks base method.
func (m *MockQuery) Names() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Names")
	ret0, _ := ret[0].([]string)
	return ret0
}

// Names indicates an expected call of Names.
func (mr *MockQueryMockRecorder) Names() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Names", reflect.TypeOf((*MockQuery)(nil).Names))
}

// Sortings mocks base method.
func (m *MockQuery) Sortings() []event.SortOptions {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Sortings")
	ret0, _ := ret[0].([]event.SortOptions)
	return ret0
}

// Sortings indicates an expected call of Sortings.
func (mr *MockQueryMockRecorder) Sortings() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sortings", reflect.TypeOf((*MockQuery)(nil).Sortings))
}

// Times mocks base method.
func (m *MockQuery) Times() time.Constraints {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Times")
	ret0, _ := ret[0].(time.Constraints)
	return ret0
}

// Times indicates an expected call of Times.
func (mr *MockQueryMockRecorder) Times() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Times", reflect.TypeOf((*MockQuery)(nil).Times))
}
