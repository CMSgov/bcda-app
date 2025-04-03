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
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	"github.com/google/fhir/go/fhirversion"
	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
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
	s.unmarshaller, err = jsonformat.NewUnmarshaller("UTC", fhirversion.STU3)
	assert.NoError(s.T(), err)
}

func TestResponseUtilsWriterTestSuite(t *testing.T) {
	suite.Run(t, new(ResponseUtilsWriterTestSuite))
}

func (s *ResponseUtilsWriterTestSuite) TestResponseWriterException() {
	rw := NewResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	rw.Exception(ctx, s.rr, http.StatusAccepted, RequestErr, "TestResponseWriterExcepton")

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodels.ContainedResource)
	respOO := cr.GetOperationOutcome()

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Diagnostics.Value)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))

}

func (s *ResponseUtilsWriterTestSuite) TestResponseWriterNotFound() {
	rw := NewResponseWriter()
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	rw.NotFound(ctx, s.rr, http.StatusAccepted, RequestErr, "TestResponseWriterNotFound")

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodels.ContainedResource)
	respOO := cr.GetOperationOutcome()

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_NOT_FOUND, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestResponseWriterNotFound", respOO.Issue[0].Diagnostics.Value)
	assert.Equal(s.T(), constants.FHIRJsonContentType, s.rr.Header().Get("Content-Type"))
}

func (s *ResponseUtilsWriterTestSuite) TestCreateOpOutcome() {
	oo := CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, RequestErr, "TestCreateOpOutcome")
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, oo.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_EXCEPTION, oo.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestCreateOpOutcome", oo.Issue[0].Diagnostics.Value)

}

func (s *ResponseUtilsWriterTestSuite) TestWriteError() {
	newLogEntry := MakeTestStructuredLoggerEntry(logrus.Fields{"foo": "bar"})
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	oo := CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, RequestErr, "TestCreateOpOutcome")
	WriteError(ctx, oo, s.rr, http.StatusAccepted)

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodels.ContainedResource)
	respOO := cr.GetOperationOutcome()

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), oo.Issue[0].Severity.Value, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), oo.Issue[0].Code.Value, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestCreateOpOutcome", respOO.Issue[0].Diagnostics.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateCapabilityStatement() {
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	cs := CreateCapabilityStatement(time.Now(), relversion, baseurl)
	assert.Equal(s.T(), relversion, cs.Software.Version.Value)
	assert.Equal(s.T(), "Beneficiary Claims Data API", cs.Software.Name.Value)
	assert.Equal(s.T(), baseurl, cs.Implementation.Url.Value)
	assert.Equal(s.T(), "3.0.1", cs.FhirVersion.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteCapabilityStatement() {
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	cs := CreateCapabilityStatement(time.Now(), relversion, baseurl)
	WriteCapabilityStatement(context.Background(), cs, s.rr)
	var respCS *fhirmodels.CapabilityStatement

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	cr := res.(*fhirmodels.ContainedResource)
	respCS = cr.GetCapabilityStatement()

	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), relversion, respCS.Software.Version.Value)
	assert.Equal(s.T(), cs.Software.Version.Value, respCS.Software.Version.Value)
	assert.Equal(s.T(), "Beneficiary Claims Data API", respCS.Software.Name.Value)
	assert.Equal(s.T(), cs.Software.Name.Value, respCS.Software.Name.Value)
	assert.Equal(s.T(), baseurl, respCS.Implementation.Url.Value)
	assert.Equal(s.T(), cs.Implementation.Url.Value, respCS.Implementation.Url.Value)
	assert.Equal(s.T(), "3.0.1", respCS.FhirVersion.Value)
	assert.Equal(s.T(), cs.FhirVersion.Value, respCS.FhirVersion.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteJobsBundle() {
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
	jb := CreateJobsBundle(jobs, constants.TestAPIUrl)
	WriteBundleResponse(jb, s.rr)

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodels.ContainedResource)
	bundle := cr.GetBundle()

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	total, _ := safecast.ToUint32(len(bundle.Entry))
	assert.Equal(s.T(), total, bundle.Total.Value)
	assert.Equal(s.T(), fhircodes.BundleTypeCode_SEARCHSET, bundle.Type.Value)

	task := bundle.Entry[0].GetResource().GetTask()

	assert.Equal(s.T(), jobs[0].CreatedAt.UTC().UnixNano()/int64(time.Microsecond), task.ExecutionPeriod.Start.ValueUs)
	assert.Equal(s.T(), jobs[0].UpdatedAt.UTC().UnixNano()/int64(time.Microsecond), task.ExecutionPeriod.End.ValueUs)
	assert.Equal(s.T(), "https://www.api.com/api/v1/jobs", task.Identifier[0].System.Value)
	assert.Equal(s.T(), fhirdatatypes.IdentifierUseCode_OFFICIAL, task.Identifier[0].Use.Value)
	assert.Equal(s.T(), fmt.Sprint(jobs[0].ID), task.Identifier[0].Value.Value)
	assert.Equal(s.T(), "BULK FHIR Export", task.Input[0].Type.Text.Value)
	assert.Equal(s.T(), "GET "+jobs[0].RequestURL, task.Input[0].Value.GetStringValue().Value)
	assert.Equal(s.T(), fhircodes.RequestIntentCode_ORDER, task.Intent.Value)
	assert.Equal(s.T(), fhircodes.TaskStatusCode_COMPLETED, task.Status.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateJobsBundle() {
	jb := CreateJobsBundle(nil, constants.TestAPIUrl)

	assert.Equal(s.T(), uint32(0), jb.Total.Value)
	assert.Equal(s.T(), fhircodes.BundleTypeCode_SEARCHSET, jb.Type.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateJobsBundleEntry() {
	job := models.Job{
		ID:         1,
		ACOID:      uuid.NewUUID(),
		RequestURL: "https://www.requesturl.com",
		Status:     models.JobStatusCompleted,
		CreatedAt:  time.Now().Add(-24 * time.Hour).Truncate(time.Second),
		UpdatedAt:  time.Now().Truncate(time.Second),
	}
	jbe := CreateJobsBundleEntry(&job, constants.TestAPIUrl).Resource.GetTask()

	assert.Equal(s.T(), job.CreatedAt.UTC().UnixNano()/int64(time.Microsecond), jbe.ExecutionPeriod.Start.ValueUs)
	assert.Equal(s.T(), job.UpdatedAt.UTC().UnixNano()/int64(time.Microsecond), jbe.ExecutionPeriod.End.ValueUs)
	assert.Equal(s.T(), "https://www.api.com/api/v1/jobs", jbe.Identifier[0].System.Value)
	assert.Equal(s.T(), fhirdatatypes.IdentifierUseCode_OFFICIAL, jbe.Identifier[0].Use.Value)
	assert.Equal(s.T(), fmt.Sprint(job.ID), jbe.Identifier[0].Value.Value)
	assert.Equal(s.T(), "BULK FHIR Export", jbe.Input[0].Type.Text.Value)
	assert.Equal(s.T(), "GET "+job.RequestURL, jbe.Input[0].Value.GetStringValue().Value)
	assert.Equal(s.T(), fhircodes.RequestIntentCode_ORDER, jbe.Intent.Value)
	assert.Equal(s.T(), fhircodes.TaskStatusCode_COMPLETED, jbe.Status.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestGetFhirStatusCode() {
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
			code := GetFhirStatusCode(tt.status)
			assert.Equal(s.T(), tt.code, code)
		})
	}
}

func MakeTestStructuredLoggerEntry(logFields logrus.Fields) *log.StructuredLoggerEntry {
	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logFields)}
	return newLogEntry
}
