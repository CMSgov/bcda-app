// Code generated by mockery v2.6.0. DO NOT EDIT.

package queueing

import (
	models "github.com/CMSgov/bcda-app/bcda/models"
	mock "github.com/stretchr/testify/mock"
)

// MockEnqueuer is an autogenerated mock type for the Enqueuer type
type MockEnqueuer struct {
	mock.Mock
}

// AddAlrJob provides a mock function with given fields: job, priority
func (_m *MockEnqueuer) AddAlrJob(job models.JobAlrEnqueueArgs, priority int) error {
	ret := _m.Called(job, priority)

	var r0 error
	if rf, ok := ret.Get(0).(func(models.JobAlrEnqueueArgs, int) error); ok {
		r0 = rf(job, priority)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddJob provides a mock function with given fields: job, priority
func (_m *MockEnqueuer) AddJob(job models.JobEnqueueArgs, priority int) error {
	ret := _m.Called(job, priority)

	var r0 error
	if rf, ok := ret.Get(0).(func(models.JobEnqueueArgs, int) error); ok {
		r0 = rf(job, priority)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}