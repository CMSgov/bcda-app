package responseutils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/fhir/go/jsonformat"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
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

func (s *ResponseUtilsWriterTestSuite) TestCreateOpOutcome() {
	oo := CreateOpOutcome(Error, Exception, RequestErr, "TestCreateOpOutcome")
	assert.Equal(s.T(), Error, oo.Issue[0].Severity.Value.String())
	assert.Equal(s.T(), Exception, oo.Issue[0].Code.Value.String())
	assert.Equal(s.T(), "TestCreateOpOutcome", oo.Issue[0].Details.Coding[0].Display.Value)
	assert.Equal(s.T(), "TestCreateOpOutcome", oo.Issue[0].Details.Text.Value)
	assert.Equal(s.T(), RequestErr, oo.Issue[0].Details.Coding[0].Code.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteError() {
	oo := CreateOpOutcome(Error, Exception, RequestErr, "TestCreateOpOutcome")
	WriteError(oo, s.rr, http.StatusAccepted)
	var respOO fhirmodels.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), Error, respOO.Issue[0].Severity.Value.String())
	assert.Equal(s.T(), oo.Issue[0].Severity, respOO.Issue[0].Severity)
	assert.Equal(s.T(), Exception, respOO.Issue[0].Code.Value.String())
	assert.Equal(s.T(), oo.Issue[0].Code, respOO.Issue[0].Code)
	assert.Equal(s.T(), "TestCreateOpOutcome", respOO.Issue[0].Details.Coding[0].Display.Value)
	assert.Equal(s.T(), oo.Issue[0].Details.Coding[0].Display, respOO.Issue[0].Details.Coding[0].Display)
	assert.Equal(s.T(), "TestCreateOpOutcome", respOO.Issue[0].Details.Text.Value)
	assert.Equal(s.T(), oo.Issue[0].Details.Text, respOO.Issue[0].Details.Text)
	assert.Equal(s.T(), RequestErr, respOO.Issue[0].Details.Coding[0].Code.Value)
	assert.Equal(s.T(), oo.Issue[0].Details.Coding[0].Code, respOO.Issue[0].Details.Coding[0].Code)
}

func (s *ResponseUtilsWriterTestSuite) TestCreateCapabilityStatement() {
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	var cs *fhirmodels.CapabilityStatement = CreateCapabilityStatement(time.Now(), relversion, baseurl)
	assert.Equal(s.T(), relversion, cs.Software.Version.Value)
	assert.Equal(s.T(), "Beneficiary Claims Data API", cs.Software.Name.Value)
	assert.Equal(s.T(), baseurl, cs.Implementation.Url.Value)
	assert.Equal(s.T(), "3.0.1", cs.FhirVersion.Value)
}

func (s *ResponseUtilsWriterTestSuite) TestWriteCapabilityStatement() {
	relversion := "r1"
	baseurl := "bcda.cms.gov"
	cs := CreateCapabilityStatement(time.Now(), relversion, baseurl)
	WriteCapabilityStatement(cs, s.rr)
	var respCS *fhirmodels.CapabilityStatement

	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", jsonformat.STU3)
	assert.NoError(s.T(), err)
	res, err := unmarshaller.Unmarshal(s.rr.Body.Bytes())
	cr := res.(*fhirmodels.ContainedResource)
	respCS = cr.GetCapabilityStatement()

	assert.NoError(s.T(), err)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), relversion, respCS.Software.Version.Value)
	assert.Equal(s.T(), cs.Software.Version, respCS.Software.Version)
	assert.Equal(s.T(), "Beneficiary Claims Data API", respCS.Software.Name.Value)
	assert.Equal(s.T(), cs.Software.Name, respCS.Software.Name)
	assert.Equal(s.T(), baseurl, respCS.Implementation.Url.Value)
	assert.Equal(s.T(), cs.Implementation.Url, respCS.Implementation.Url)
	assert.Equal(s.T(), "3.0.1", respCS.FhirVersion.Value)
	assert.Equal(s.T(), cs.FhirVersion, respCS.FhirVersion)
}
