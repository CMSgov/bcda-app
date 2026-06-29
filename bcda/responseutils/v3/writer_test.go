package responseutils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/r4"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ResponseUtilsWriterTestSuite struct {
	suite.Suite
	rr *httptest.ResponseRecorder
}

func (s *ResponseUtilsWriterTestSuite) SetupTest() {
	s.rr = httptest.NewRecorder()
}

func TestResponseUtilsWriterTestSuite(t *testing.T) {
	suite.Run(t, new(ResponseUtilsWriterTestSuite))
}

func (s *ResponseUtilsWriterTestSuite) TestResponseWriterException() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	rw.Exception(ctx, s.rr, http.StatusAccepted, responseutils.RequestErr, "TestResponseWriterExcepton")

	var respOO r4.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), r4.IssueSeverityError, respOO.Issue[0].Severity)
	assert.Equal(s.T(), r4.IssueTypeCodeException, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Diagnostics)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))
	// Verify Details.Text exists
	assert.NotNil(s.T(), respOO.Issue[0].Details)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Details.Text)
	// Verify Details.Coding does not exist (v3 should not include coding)
	assert.Nil(s.T(), respOO.Issue[0].Details.Coding)
	assert.Empty(s.T(), respOO.Issue[0].Details.Coding)
}

func (s *ResponseUtilsWriterTestSuite) TestResponseWriterNotFound() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	rw.NotFound(ctx, s.rr, http.StatusAccepted, responseutils.RequestErr, "TestResponseWriterNotFound")

	var respOO r4.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), r4.IssueSeverityError, respOO.Issue[0].Severity)
	assert.Equal(s.T(), r4.IssueTypeCodeNotFound, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestResponseWriterNotFound", respOO.Issue[0].Diagnostics)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))
	// Verify Details.Text exists
	assert.NotNil(s.T(), respOO.Issue[0].Details)
	assert.Equal(s.T(), "TestResponseWriterNotFound", respOO.Issue[0].Details.Text)
	// Verify Details.Coding does not exist (v3 should not include coding)
	assert.Nil(s.T(), respOO.Issue[0].Details.Coding)
	assert.Empty(s.T(), respOO.Issue[0].Details.Coding)
}

func (s *ResponseUtilsWriterTestSuite) TestResponseWriterOpOutcome() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	rw.OpOutcome(ctx, s.rr, http.StatusBadRequest, responseutils.RequestErr, "TestResponseWriterExcepton")

	var respOO r4.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
	assert.Equal(s.T(), r4.IssueSeverityError, respOO.Issue[0].Severity)
	assert.Equal(s.T(), r4.IssueTypeCodeStructure, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Diagnostics)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))
	// Verify Details.Text exists
	assert.NotNil(s.T(), respOO.Issue[0].Details)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Details.Text)
	// Verify Details.Coding does not exist (v3 should not include coding)
	assert.Nil(s.T(), respOO.Issue[0].Details.Coding)
	assert.Empty(s.T(), respOO.Issue[0].Details.Coding)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateOpOutcome() {
	rw := NewFhirResponseWriter()
	oo := rw.CreateOpOutcome(r4.IssueSeverityError, r4.IssueTypeCodeException, responseutils.RequestErr, "TestCreateOpOutcome")
	assert.Equal(s.T(), r4.IssueSeverityError, oo.Issue[0].Severity)
	assert.Equal(s.T(), r4.IssueTypeCodeException, oo.Issue[0].Code)
	assert.Equal(s.T(), "TestCreateOpOutcome", oo.Issue[0].Diagnostics)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteError() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	oo := rw.CreateOpOutcome(r4.IssueSeverityError, r4.IssueTypeCodeException, responseutils.RequestErr, "TestCreateOpOutcome")
	rw.WriteError(ctx, oo, s.rr, http.StatusAccepted)

	var respOO r4.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), r4.IssueSeverityError, respOO.Issue[0].Severity)
	assert.Equal(s.T(), oo.Issue[0].Severity, respOO.Issue[0].Severity)
	assert.Equal(s.T(), r4.IssueTypeCodeException, respOO.Issue[0].Code)
	assert.Equal(s.T(), oo.Issue[0].Code, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestCreateOpOutcome", respOO.Issue[0].Diagnostics)
}


func (s *ResponseUtilsWriterTestSuite) TestWriteJobsBundle() {
	rw := NewFhirResponseWriter()
	jobs := []*models.Job{
		{
			ID:         1,
			ACOID:      uuid.NewUUID(),
			RequestURL: "https://www.requesturl.com",
			Status:     models.JobStatusCompleted,
			CreatedAt:  time.Now().Add(-24 * time.Hour).Truncate(time.Second),
			UpdatedAt:  time.Now().Truncate(time.Second),
		},
	}
	jb := rw.CreateJobsBundle(jobs, constants.TestAPIUrl)
	rw.WriteBundleResponse(jb, s.rr)

	var bundle r4.Bundle
	err := json.Unmarshal(s.rr.Body.Bytes(), &bundle)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	u, err := safecast.ToUint32(len(bundle.Entry))
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), u, bundle.Total)
	assert.Equal(s.T(), "searchset", bundle.Type)

	var task r4.Task
	taskBytes, err := json.Marshal(bundle.Entry[0].Resource)
	assert.NoError(s.T(), err)
	assert.NoError(s.T(), json.Unmarshal(taskBytes, &task))

	assert.Equal(s.T(), jobs[0].CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), task.ExecutionPeriod.Start)
	assert.Equal(s.T(), jobs[0].UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"), task.ExecutionPeriod.End)
	assert.Equal(s.T(), "https://www.api.com/api/v3/jobs", task.Identifier[0].System)
	assert.Equal(s.T(), "official", task.Identifier[0].Use)
	assert.Equal(s.T(), fmt.Sprint(jobs[0].ID), task.Identifier[0].Value)
	assert.Equal(s.T(), "BULK FHIR Export", task.Input[0].Type.Text)
	assert.Equal(s.T(), "GET "+jobs[0].RequestURL, task.Input[0].Value.(string))
	assert.Equal(s.T(), r4.TaskIntentOrder, task.Intent)
	assert.Equal(s.T(), r4.TaskStatusCompleted, task.Status)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateJobsBundle() {
	rw := NewFhirResponseWriter()
	jb := rw.CreateJobsBundle(nil, constants.TestAPIUrl)

	assert.Equal(s.T(), uint32(0), jb.Total)
	assert.Equal(s.T(), "searchset", jb.Type)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateJobsBundleEntry() {
	rw := NewFhirResponseWriter()
	job := models.Job{
		ID:         1,
		ACOID:      uuid.NewUUID(),
		RequestURL: "https://www.requesturl.com",
		Status:     models.JobStatusCompleted,
		CreatedAt:  time.Now().Add(-24 * time.Hour).Truncate(time.Second),
		UpdatedAt:  time.Now().Truncate(time.Second),
	}
	jbe := rw.CreateJobsBundleEntry(&job, constants.TestAPIUrl).Resource.(*r4.Task)

	assert.Equal(s.T(), job.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), jbe.ExecutionPeriod.Start)
	assert.Equal(s.T(), job.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"), jbe.ExecutionPeriod.End)
	assert.Equal(s.T(), "https://www.api.com/api/v3/jobs", jbe.Identifier[0].System)
	assert.Equal(s.T(), "official", jbe.Identifier[0].Use)
	assert.Equal(s.T(), fmt.Sprint(job.ID), jbe.Identifier[0].Value)
	assert.Equal(s.T(), "BULK FHIR Export", jbe.Input[0].Type.Text)
	assert.Equal(s.T(), "GET "+job.RequestURL, jbe.Input[0].Value.(string))
	assert.Equal(s.T(), r4.TaskIntentOrder, jbe.Intent)
	assert.Equal(s.T(), r4.TaskStatusCompleted, jbe.Status)
}

func (s *ResponseUtilsWriterTestSuite) TestGetFhirStatusCode() {
	rw := NewFhirResponseWriter()
	tests := []struct {
		name string

		status models.JobStatus
		code   r4.TaskStatus
	}{
		{"Job Failed returns fhir task status Failed", models.JobStatusFailed, r4.TaskStatusFailed},
		{"Job FailedExpired returns fhir task status Failed", models.JobStatusFailedExpired, r4.TaskStatusFailed},
		{"Job Pending returns fhir task status Accepted", models.JobStatusPending, r4.TaskStatusAccepted},
		{"Job In Progress returns fhir task status In Progress", models.JobStatusInProgress, r4.TaskStatusInProgress},
		{"Job Completed returns fhir task status Completed", models.JobStatusCompleted, r4.TaskStatusCompleted},
		{"Job Archived returns fhir task status Completed", models.JobStatusArchived, r4.TaskStatusCompleted},
		{"Job Expired returns fhir task status Completed", models.JobStatusExpired, r4.TaskStatusCompleted},
		{"Job Cancelled returns fhir task status Cancelled", models.JobStatusCancelled, r4.TaskStatusCancelled},
		{"Job CancelledExpired returns fhir task status Cancelled", models.JobStatusCancelledExpired, r4.TaskStatusCancelled},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			code := rw.GetFhirStatusCode(tt.status)
			assert.Equal(s.T(), tt.code, code)
		})
	}
}

func MakeTestStructuredLoggerEntry(logFields logrus.Fields) *log.StructuredLoggerEntry {
	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logFields)}
	return newLogEntry
}
