package responseutils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirmodelCR "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhirmodelCS "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/capability_statement_go_proto"
	fhirvaluesets "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/valuesets_go_proto"
	"github.com/pborman/uuid"
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
	s.unmarshaller, err = jsonformat.NewUnmarshaller("UTC", jsonformat.R4)
	assert.NoError(s.T(), err)
}

func TestResponseUtilsWriterTestSuite(t *testing.T) {
	suite.Run(t, new(ResponseUtilsWriterTestSuite))
}
func (s *ResponseUtilsWriterTestSuite) TestResponseWriterException() {
	rw := NewResponseWriter()
	rw.Exception(s.rr, http.StatusAccepted, responseutils.RequestErr, "TestResponseWriterExcepton")

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodelCR.ContainedResource)
	respOO := cr.GetOperationOutcome()

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Details.Coding[0].Display.Value)
	assert.Equal(s.T(), "TestResponseWriterExcepton", respOO.Issue[0].Details.Text.Value)
	assert.Equal(s.T(), responseutils.RequestErr, respOO.Issue[0].Details.Coding[0].Code.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestResponseWriterNotFound() {
	rw := NewResponseWriter()

	rw.NotFound(s.rr, http.StatusAccepted, responseutils.RequestErr, "TestResponseWriterNotFound")

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodelCR.ContainedResource)
	respOO := cr.GetOperationOutcome()

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_NOT_FOUND, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestResponseWriterNotFound", respOO.Issue[0].Details.Coding[0].Display.Value)
	assert.Equal(s.T(), "TestResponseWriterNotFound", respOO.Issue[0].Details.Text.Value)
	assert.Equal(s.T(), responseutils.RequestErr, respOO.Issue[0].Details.Coding[0].Code.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateOpOutcome() {
	oo := CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr, "TestCreateOpOutcome")
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, oo.Issue[0].Severity.Value)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_EXCEPTION, oo.Issue[0].Code.Value)
	assert.Equal(s.T(), "TestCreateOpOutcome", oo.Issue[0].Details.Coding[0].Display.Value)
	assert.Equal(s.T(), "TestCreateOpOutcome", oo.Issue[0].Details.Text.Value)
	assert.Equal(s.T(), responseutils.RequestErr, oo.Issue[0].Details.Coding[0].Code.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteError() {
	oo := CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_EXCEPTION, responseutils.RequestErr, "TestCreateOpOutcome")
	WriteError(oo, s.rr, http.StatusAccepted)

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodelCR.ContainedResource)
	respOO := cr.GetOperationOutcome()

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), fhircodes.IssueSeverityCode_ERROR, respOO.Issue[0].Severity.Value)
	assert.Equal(s.T(), oo.Issue[0].Severity, respOO.Issue[0].Severity)
	assert.Equal(s.T(), fhircodes.IssueTypeCode_EXCEPTION, respOO.Issue[0].Code.Value)
	assert.Equal(s.T(), oo.Issue[0].Code, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestCreateOpOutcome", respOO.Issue[0].Details.Coding[0].Display.Value)
	assert.Equal(s.T(), oo.Issue[0].Details.Coding[0].Display, respOO.Issue[0].Details.Coding[0].Display)
	assert.Equal(s.T(), "TestCreateOpOutcome", respOO.Issue[0].Details.Text.Value)
	assert.Equal(s.T(), oo.Issue[0].Details.Text, respOO.Issue[0].Details.Text)
	assert.Equal(s.T(), responseutils.RequestErr, respOO.Issue[0].Details.Coding[0].Code.Value)
	assert.Equal(s.T(), oo.Issue[0].Details.Coding[0].Code, respOO.Issue[0].Details.Coding[0].Code)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateCapabilityStatement() {
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	var cs *fhirmodelCS.CapabilityStatement = CreateCapabilityStatement(time.Now(), relversion, baseurl)
	assert.Equal(s.T(), relversion, cs.Software.Version.Value)
	assert.Equal(s.T(), "Beneficiary Claims Data API", cs.Software.Name.Value)
	assert.Equal(s.T(), baseurl, cs.Implementation.Url.Value)
	assert.Equal(s.T(), fhircodes.FHIRVersionCode_V_3_0_1, cs.FhirVersion.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteCapabilityStatement() {
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	cs := CreateCapabilityStatement(time.Now(), relversion, baseurl)
	WriteCapabilityStatement(cs, s.rr)
	var respCS *fhirmodelCS.CapabilityStatement

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	cr := res.(*fhirmodelCR.ContainedResource)
	respCS = cr.GetCapabilityStatement()

	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), relversion, respCS.Software.Version.Value)
	assert.Equal(s.T(), cs.Software.Version, respCS.Software.Version)
	assert.Equal(s.T(), "Beneficiary Claims Data API", respCS.Software.Name.Value)
	assert.Equal(s.T(), cs.Software.Name, respCS.Software.Name)
	assert.Equal(s.T(), baseurl, respCS.Implementation.Url.Value)
	assert.Equal(s.T(), cs.Implementation.Url, respCS.Implementation.Url)
	assert.Equal(s.T(), fhircodes.FHIRVersionCode_V_3_0_1, respCS.FhirVersion.Value)
	assert.Equal(s.T(), cs.FhirVersion, respCS.FhirVersion)
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
	jb := CreateJobsBundle(jobs, "https://www.api.com")
	WriteBundleResponse(jb, s.rr)

	res, err := s.unmarshaller.Unmarshal(s.rr.Body.Bytes())
	assert.NoError(s.T(), err)
	cr := res.(*fhirmodelCR.ContainedResource)
	bundle := cr.GetBundle()

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), uint32(len(bundle.Entry)), bundle.Total.Value)
	assert.Equal(s.T(), fhircodes.BundleTypeCode_SEARCHSET, bundle.Type.Value)

	task := bundle.Entry[0].GetResource().GetTask()

	assert.Equal(s.T(), jobs[0].CreatedAt.UTC().UnixNano()/int64(time.Microsecond), task.ExecutionPeriod.Start.ValueUs)
	assert.Equal(s.T(), jobs[0].UpdatedAt.UTC().UnixNano()/int64(time.Microsecond), task.ExecutionPeriod.End.ValueUs)
	assert.Equal(s.T(), "https://www.api.com/api/v1/jobs", task.Identifier[0].System.Value)
	assert.Equal(s.T(), fhircodes.IdentifierUseCode_OFFICIAL, task.Identifier[0].Use.Value)
	assert.Equal(s.T(), fmt.Sprint(jobs[0].ID), task.Identifier[0].Value.Value)
	assert.Equal(s.T(), "BULK FHIR Export", task.Input[0].Type.Text.Value)
	assert.Equal(s.T(), "GET "+jobs[0].RequestURL, task.Input[0].Value.GetStringValue().Value)
	assert.Equal(s.T(), fhirvaluesets.TaskIntentValueSet_ORDER, task.Intent.Value)
	assert.Equal(s.T(), fhircodes.TaskStatusCode_COMPLETED, task.Status.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateJobsBundle() {
	var jb *fhirmodelCR.Bundle = CreateJobsBundle(nil, "https://www.api.com")

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
	jbe := CreateJobsBundleEntry(&job, "https://www.api.com").Resource.GetTask()

	assert.Equal(s.T(), job.CreatedAt.UTC().UnixNano()/int64(time.Microsecond), jbe.ExecutionPeriod.Start.ValueUs)
	assert.Equal(s.T(), job.UpdatedAt.UTC().UnixNano()/int64(time.Microsecond), jbe.ExecutionPeriod.End.ValueUs)
	assert.Equal(s.T(), "https://www.api.com/api/v1/jobs", jbe.Identifier[0].System.Value)
	assert.Equal(s.T(), fhircodes.IdentifierUseCode_OFFICIAL, jbe.Identifier[0].Use.Value)
	assert.Equal(s.T(), fmt.Sprint(job.ID), jbe.Identifier[0].Value.Value)
	assert.Equal(s.T(), "BULK FHIR Export", jbe.Input[0].Type.Text.Value)
	assert.Equal(s.T(), "GET "+job.RequestURL, jbe.Input[0].Value.GetStringValue().Value)
	assert.Equal(s.T(), fhirvaluesets.TaskIntentValueSet_ORDER, jbe.Intent.Value)
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
		{"Job Pending returns fhir task status In Progress", models.JobStatusPending, fhircodes.TaskStatusCode_IN_PROGRESS},
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
