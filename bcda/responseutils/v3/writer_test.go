package responseutils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirmodelCR "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhirmodelCS "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/capability_statement_go_proto"
	fhirvaluesets "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/valuesets_go_proto"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ResponseUtilsWriterTestSuite struct {
	suite.Suite
	rr           *httptest.ResponseRecorder
	unmarshaller *jsonformat.Unmarshaller
}

func (s *ResponseUtilsWriterTestSuite) SetupTest() {
	var err error
	s.rr = httptest.NewRecorder()
	s.unmarshaller, err = jsonformat.NewUnmarshaller("UTC", fhirversion.R4)
	assert.NoError(s.T(), err)
}

func TestResponseUtilsWriterTestSuite(t *testing.T) {
	suite.Run(t, new(ResponseUtilsWriterTestSuite))
}
func (s *ResponseUtilsWriterTestSuite) TestResponseWriterException() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	rw.Exception(ctx, s.rr, http.StatusAccepted, responseutils.RequestErr, "TestResponseWriterExcepton")

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodelCR.ContainedResource)
	respOO := cr.GetOperationOutcome()

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Diagnostics.Value)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))
	// Verify Details.Text exists
	assert.NotNil(s.T(), respOO.Issue[0].Details)
	assert.NotNil(s.T(), respOO.Issue[0].Details.Text)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Details.Text.Value)
	// Verify Details.Coding does not exist (v3 should not include coding)
	assert.Nil(s.T(), respOO.Issue[0].Details.Coding)
	assert.Empty(s.T(), respOO.Issue[0].Details.Coding)
}

func (s *ResponseUtilsWriterTestSuite) TestResponseWriterNotFound() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	rw.NotFound(ctx, s.rr, http.StatusAccepted, responseutils.RequestErr, "TestResponseWriterNotFound")

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodelCR.ContainedResource)
	respOO := cr.GetOperationOutcome()

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_NOT_FOUND, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestResponseWriterNotFound", respOO.Issue[0].Diagnostics.Value)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))
	// Verify Details.Text exists
	assert.NotNil(s.T(), respOO.Issue[0].Details)
	assert.NotNil(s.T(), respOO.Issue[0].Details.Text)
	assert.Equal(s.T(), "TestResponseWriterNotFound", respOO.Issue[0].Details.Text.Value)
	// Verify Details.Coding does not exist (v3 should not include coding)
	assert.Nil(s.T(), respOO.Issue[0].Details.Coding)
	assert.Empty(s.T(), respOO.Issue[0].Details.Coding)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateOpOutcome() {
	rw := NewFhirResponseWriter()
	oo := rw.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr, "TestCreateOpOutcome")
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, oo.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_EXCEPTION, oo.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestCreateOpOutcome", oo.Issue[0].Diagnostics.Value)
	// Verify Details.Text exists
	assert.NotNil(s.T(), oo.Issue[0].Details)
	assert.NotNil(s.T(), oo.Issue[0].Details.Text)
	assert.Equal(s.T(), "TestCreateOpOutcome", oo.Issue[0].Details.Text.Value)
	// Verify Details.Coding does not exist (v3 should not include coding)
	assert.Nil(s.T(), oo.Issue[0].Details.Coding)
	assert.Empty(s.T(), oo.Issue[0].Details.Coding)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateOpOutcomeNoCoding() {
	// This test specifically verifies that v3 OperationOutcome does not include Coding
	rw := NewFhirResponseWriter()
	oo := rw.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr, "TestDiagnostics")

	// Verify Details exists
	assert.NotNil(s.T(), oo.Issue[0].Details)

	// Verify Details.Text exists and contains the diagnostic message
	assert.NotNil(s.T(), oo.Issue[0].Details.Text)
	assert.Equal(s.T(), "TestDiagnostics", oo.Issue[0].Details.Text.Value)

	// Verify Details.Coding is nil or empty (should not be present)
	assert.Nil(s.T(), oo.Issue[0].Details.Coding)
	assert.Empty(s.T(), oo.Issue[0].Details.Coding)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteError() {
	rw := NewFhirResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	oo := rw.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr, "TestCreateOpOutcome")
	rw.WriteError(ctx, oo, s.rr, http.StatusAccepted)

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodelCR.ContainedResource)
	respOO := cr.GetOperationOutcome()

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), oo.Issue[0].Severity.Value, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), oo.Issue[0].Code.Value, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestCreateOpOutcome", respOO.Issue[0].Diagnostics.Value)
	// Verify Details.Text exists
	assert.NotNil(s.T(), respOO.Issue[0].Details)
	assert.NotNil(s.T(), respOO.Issue[0].Details.Text)
	assert.Equal(s.T(), "TestCreateOpOutcome", respOO.Issue[0].Details.Text.Value)
	// Verify Details.Coding does not exist (v3 should not include coding)
	assert.Nil(s.T(), respOO.Issue[0].Details.Coding)
	assert.Empty(s.T(), respOO.Issue[0].Details.Coding)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateCapabilityStatement() {
	rw := NewFhirResponseWriter()
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	cs := rw.CreateCapabilityStatement(time.Now(), relversion, baseurl)
	assert.Equal(s.T(), relversion, cs.Software.Version.Value)
	assert.Equal(s.T(), "Beneficiary Claims Data API", cs.Software.Name.Value)
	assert.Equal(s.T(), baseurl, cs.Implementation.Url.Value)
	assert.Equal(s.T(), fhircodes.FHIRVersionCode_V_3_0_1, cs.FhirVersion.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteCapabilityStatement() {
	rw := NewFhirResponseWriter()
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	cs := rw.CreateCapabilityStatement(time.Now(), relversion, baseurl)
	rw.WriteCapabilityStatement(context.Background(), cs, s.rr)
	var respCS *fhirmodelCS.CapabilityStatement

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	cr := res.(*fhirmodelCR.ContainedResource)
	respCS = cr.GetCapabilityStatement()

	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), relversion, respCS.Software.Version.Value)
	assert.Equal(s.T(), cs.Software.Version.Value, respCS.Software.Version.Value)
	assert.Equal(s.T(), "Beneficiary Claims Data API", respCS.Software.Name.Value)
	assert.Equal(s.T(), cs.Software.Name.Value, respCS.Software.Name.Value)
	assert.Equal(s.T(), baseurl, respCS.Implementation.Url.Value)
	assert.Equal(s.T(), cs.Implementation.Url.Value, respCS.Implementation.Url.Value)
	assert.Equal(s.T(), fhircodes.FHIRVersionCode_V_3_0_1, respCS.FhirVersion.Value)
	assert.Equal(s.T(), cs.FhirVersion.Value, respCS.FhirVersion.Value)
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

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodelCR.ContainedResource)
	bundle := cr.GetBundle()

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	u, err := safecast.ToUint32(len(bundle.Entry))
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), u, bundle.Total.Value)
	assert.Equal(s.T(), fhircodes.BundleTypeCode_SEARCHSET, bundle.Type.Value)

	task := bundle.Entry[0].GetResource().GetTask()

	assert.Equal(s.T(), jobs[0].CreatedAt.UTC().UnixNano()/int64(time.Microsecond), task.ExecutionPeriod.Start.ValueUs)
	assert.Equal(s.T(), jobs[0].UpdatedAt.UTC().UnixNano()/int64(time.Microsecond), task.ExecutionPeriod.End.ValueUs)
	assert.Equal(s.T(), "https://www.api.com/api/v3/jobs", task.Identifier[0].System.Value)
	assert.Equal(s.T(), fhircodes.IdentifierUseCode_OFFICIAL, task.Identifier[0].Use.Value)
	assert.Equal(s.T(), fmt.Sprint(jobs[0].ID), task.Identifier[0].Value.Value)
	assert.Equal(s.T(), "BULK FHIR Export", task.Input[0].Type.Text.Value)
	assert.Equal(s.T(), "GET "+jobs[0].RequestURL, task.Input[0].Value.GetStringValue().Value)
	assert.Equal(s.T(), fhirvaluesets.TaskIntentValueSet_ORDER, task.Intent.Value)
	assert.Equal(s.T(), fhircodes.TaskStatusCode_COMPLETED, task.Status.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateJobsBundle() {
	rw := NewFhirResponseWriter()
	jb := rw.CreateJobsBundle(nil, constants.TestAPIUrl)

	assert.Equal(s.T(), uint32(0), jb.Total.Value)
	assert.Equal(s.T(), fhircodes.BundleTypeCode_SEARCHSET, jb.Type.Value)
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
	jbe := rw.CreateJobsBundleEntry(&job, constants.TestAPIUrl).Resource.GetTask()

	assert.Equal(s.T(), job.CreatedAt.UTC().UnixNano()/int64(time.Microsecond), jbe.ExecutionPeriod.Start.ValueUs)
	assert.Equal(s.T(), job.UpdatedAt.UTC().UnixNano()/int64(time.Microsecond), jbe.ExecutionPeriod.End.ValueUs)
	assert.Equal(s.T(), "https://www.api.com/api/v3/jobs", jbe.Identifier[0].System.Value)
	assert.Equal(s.T(), fhircodes.IdentifierUseCode_OFFICIAL, jbe.Identifier[0].Use.Value)
	assert.Equal(s.T(), fmt.Sprint(job.ID), jbe.Identifier[0].Value.Value)
	assert.Equal(s.T(), "BULK FHIR Export", jbe.Input[0].Type.Text.Value)
	assert.Equal(s.T(), "GET "+job.RequestURL, jbe.Input[0].Value.GetStringValue().Value)
	assert.Equal(s.T(), fhirvaluesets.TaskIntentValueSet_ORDER, jbe.Intent.Value)
	assert.Equal(s.T(), fhircodes.TaskStatusCode_COMPLETED, jbe.Status.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestGetFhirStatusCode() {
	rw := NewFhirResponseWriter()
	tests := []struct {
		name string

		status models.JobStatus
		code   fhircodes.TaskStatusCode_Value
	}{
		{"Job Failed returns fhir task status Failed", models.JobStatusFailed, fhircodes.TaskStatusCode_FAILED},
		{"Job FailedExpired returns fhir task status Failed", models.JobStatusFailedExpired, fhircodes.TaskStatusCode_FAILED},
		{"Job Pending returns fhir task status Accepted", models.JobStatusPending, fhircodes.TaskStatusCode_ACCEPTED},
		{"Job In Progress returns fhir task status In Progress", models.JobStatusInProgress, fhircodes.TaskStatusCode_IN_PROGRESS},
		{"Job Completed returns fhir task status Completed", models.JobStatusCompleted, fhircodes.TaskStatusCode_COMPLETED},
		{"Job Archived returns fhir task status Completed", models.JobStatusArchived, fhircodes.TaskStatusCode_COMPLETED},
		{"Job Expired returns fhir task status Completed", models.JobStatusExpired, fhircodes.TaskStatusCode_COMPLETED},
		{"Job Cancelled returns fhir task status Cancelled", models.JobStatusCancelled, fhircodes.TaskStatusCode_CANCELLED},
		{"Job CancelledExpired returns fhir task status Cancelled", models.JobStatusCancelledExpired, fhircodes.TaskStatusCode_CANCELLED},
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
