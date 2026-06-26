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
	"github.com/CMSgov/bcda-app/bcda/models/fhir/stu3"
	"github.com/CMSgov/bcda-app/log"
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
	rw.Exception(ctx, s.rr, http.StatusAccepted, RequestErr, "TestResponseWriterExcepton")

	var respOO stu3.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), stu3.IssueSeverityError, respOO.Issue[0].Severity)
	assert.Equal(s.T(), stu3.IssueTypeCodeException, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Diagnostics)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))
}

func (s *ResponseUtilsWriterTestSuite) TestResponseWriterNotFound() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	rw.NotFound(ctx, s.rr, http.StatusAccepted, RequestErr, "TestResponseWriterNotFound")

	var respOO stu3.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), stu3.IssueSeverityError, respOO.Issue[0].Severity)
	assert.Equal(s.T(), stu3.IssueTypeCodeNotFound, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestResponseWriterNotFound", respOO.Issue[0].Diagnostics)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))
}

func (s *ResponseUtilsWriterTestSuite) TestResponseWriterOpOutcome() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	rw.OpOutcome(ctx, s.rr, http.StatusBadRequest, RequestErr, "TestResponseWriterExcepton")

	var respOO stu3.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
	assert.Equal(s.T(), stu3.IssueSeverityError, respOO.Issue[0].Severity)
	assert.Equal(s.T(), stu3.IssueTypeCodeStructure, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Diagnostics)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))
}

func (s *ResponseUtilsWriterTestSuite) TestCreateOpOutcome() {
	rw := NewFhirResponseWriter()
	oo := rw.CreateOpOutcome(stu3.IssueSeverityError, stu3.IssueTypeCodeException, RequestErr, "TestCreateOpOutcome")
	assert.Equal(s.T(), stu3.IssueSeverityError, oo.Issue[0].Severity)
	assert.Equal(s.T(), stu3.IssueTypeCodeException, oo.Issue[0].Code)
	assert.Equal(s.T(), "TestCreateOpOutcome", oo.Issue[0].Diagnostics)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteError() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	oo := rw.CreateOpOutcome(stu3.IssueSeverityError, stu3.IssueTypeCodeException, RequestErr, "TestCreateOpOutcome")
	rw.WriteError(ctx, oo, s.rr, http.StatusAccepted)

	var respOO stu3.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), stu3.IssueSeverityError, respOO.Issue[0].Severity)
	assert.Equal(s.T(), oo.Issue[0].Severity, respOO.Issue[0].Severity)
	assert.Equal(s.T(), stu3.IssueTypeCodeException, respOO.Issue[0].Code)
	assert.Equal(s.T(), oo.Issue[0].Code, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestCreateOpOutcome", respOO.Issue[0].Diagnostics)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateCapabilityStatement() {
	rw := NewFhirResponseWriter()
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	cs := rw.CreateCapabilityStatement(time.Now(), relversion, baseurl)
	assert.Equal(s.T(), relversion, cs.Software.Version)
	assert.Equal(s.T(), "Beneficiary Claims Data API", cs.Software.Name)
	assert.Equal(s.T(), baseurl, cs.Implementation.Url)
	assert.Equal(s.T(), "3.0.1", cs.FhirVersion)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteCapabilityStatement() {
	rw := NewFhirResponseWriter()
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	cs := rw.CreateCapabilityStatement(time.Now(), relversion, baseurl)
	rw.WriteCapabilityStatement(context.Background(), cs, s.rr)

	var respCS stu3.CapabilityStatement
	err := json.Unmarshal(s.rr.Body.Bytes(), &respCS)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), relversion, respCS.Software.Version)
	assert.Equal(s.T(), cs.Software.Version, respCS.Software.Version)
	assert.Equal(s.T(), "Beneficiary Claims Data API", respCS.Software.Name)
	assert.Equal(s.T(), cs.Software.Name, respCS.Software.Name)
	assert.Equal(s.T(), baseurl, respCS.Implementation.Url)
	assert.Equal(s.T(), cs.Implementation.Url, respCS.Implementation.Url)
	assert.Equal(s.T(), "3.0.1", respCS.FhirVersion)
	assert.Equal(s.T(), cs.FhirVersion, respCS.FhirVersion)
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

	var bundle stu3.Bundle
	err := json.Unmarshal(s.rr.Body.Bytes(), &bundle)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), uint32(len(bundle.Entry)), bundle.Total)
	assert.Equal(s.T(), "searchset", bundle.Type)

	var task stu3.Task
	taskBytes, err := json.Marshal(bundle.Entry[0].Resource)
	assert.NoError(s.T(), err)
	assert.NoError(s.T(), json.Unmarshal(taskBytes, &task))

	assert.Equal(s.T(), jobs[0].CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), task.ExecutionPeriod.Start)
	assert.Equal(s.T(), jobs[0].UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"), task.ExecutionPeriod.End)
	assert.Equal(s.T(), "https://www.api.com/api/v1/jobs", task.Identifier[0].System)
	assert.Equal(s.T(), "official", task.Identifier[0].Use)
	assert.Equal(s.T(), fmt.Sprint(jobs[0].ID), task.Identifier[0].Value)
	assert.Equal(s.T(), "BULK FHIR Export", task.Input[0].Type.Text)
	assert.Equal(s.T(), "GET "+jobs[0].RequestURL, task.Input[0].ValueString)
	assert.Equal(s.T(), stu3.TaskIntentOrder, task.Intent)
	assert.Equal(s.T(), stu3.TaskStatusCompleted, task.Status)
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
	jbe := rw.CreateJobsBundleEntry(&job, constants.TestAPIUrl).Resource.(*stu3.Task)

	assert.Equal(s.T(), job.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), jbe.ExecutionPeriod.Start)
	assert.Equal(s.T(), job.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"), jbe.ExecutionPeriod.End)
	assert.Equal(s.T(), "https://www.api.com/api/v1/jobs", jbe.Identifier[0].System)
	assert.Equal(s.T(), "official", jbe.Identifier[0].Use)
	assert.Equal(s.T(), fmt.Sprint(job.ID), jbe.Identifier[0].Value)
	assert.Equal(s.T(), "BULK FHIR Export", jbe.Input[0].Type.Text)
	assert.Equal(s.T(), "GET "+job.RequestURL, jbe.Input[0].ValueString)
	assert.Equal(s.T(), stu3.TaskIntentOrder, jbe.Intent)
	assert.Equal(s.T(), stu3.TaskStatusCompleted, jbe.Status)
}

func (s *ResponseUtilsWriterTestSuite) TestGetFhirStatusCode() {
	rw := NewFhirResponseWriter()
	tests := []struct {
		name string

		status models.JobStatus
		code   stu3.TaskStatus
	}{
		{"Job Failed returns fhir task status Failed", models.JobStatusFailed, stu3.TaskStatusFailed},
		{"Job FailedExpired returns fhir task status Failed", models.JobStatusFailedExpired, stu3.TaskStatusFailed},
		{"Job Pending returns fhir task status Accepted", models.JobStatusPending, stu3.TaskStatusAccepted},
		{"Job In Progress returns fhir task status In Progress", models.JobStatusInProgress, stu3.TaskStatusInProgress},
		{"Job Completed returns fhir task status Completed", models.JobStatusCompleted, stu3.TaskStatusCompleted},
		{"Job Archived returns fhir task status Completed", models.JobStatusArchived, stu3.TaskStatusCompleted},
		{"Job Expired returns fhir task status Completed", models.JobStatusExpired, stu3.TaskStatusCompleted},
		{"Job Cancelled returns fhir task status Cancelled", models.JobStatusCancelled, stu3.TaskStatusCancelled},
		{"Job CancelledExpired returns fhir task status Cancelled", models.JobStatusCancelledExpired, stu3.TaskStatusCancelled},
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
