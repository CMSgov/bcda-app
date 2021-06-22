package responseutils

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirmodelCR "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	fhirmodelCS "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/capability_statement_go_proto"
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
