package v2_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	v2 "github.com/CMSgov/bcda-app/bcda/api/v2"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

const (
	acoUnderTest = constants.LargeACOUUID // Should use a different ACO compared to v1 tests
)

type APITestSuite struct {
	suite.Suite
	db *gorm.DB

	cleanup func()
}

func (s *APITestSuite) SetupSuite() {
	origDate := os.Getenv("CCLF_REF_DATE")
	os.Setenv("CCLF_REF_DATE", time.Now().Format("060102 15:01:01"))
	os.Setenv("BB_REQUEST_RETRY_INTERVAL_MS", "10")
	origBBCert := os.Getenv("BB_CLIENT_CERT_FILE")
	os.Setenv("BB_CLIENT_CERT_FILE", "../../../shared_files/decrypted/bfd-dev-test-cert.pem")
	origBBKey := os.Getenv("BB_CLIENT_KEY_FILE")
	os.Setenv("BB_CLIENT_KEY_FILE", "../../../shared_files/decrypted/bfd-dev-test-key.pem")

	s.cleanup = func() {
		os.Setenv("CCLF_REF_DATE", origDate)
		os.Setenv("BB_CLIENT_CERT_FILE", origBBCert)
		os.Setenv("BB_CLIENT_KEY_FILE", origBBKey)
	}

	s.db = database.GetGORMDbConnection()
}

func (s *APITestSuite) TearDownSuite() {
	s.cleanup()
	database.Close(s.db)
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func (s *APITestSuite) TestMetadataResponse() {
	ts := httptest.NewServer(http.HandlerFunc(v2.Metadata))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), "application/json", res.Header.Get("Content-Type"))
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	resp, err := ioutil.ReadAll(res.Body)
	assert.NoError(s.T(), err)

	cs, err := fhir.UnmarshalCapabilityStatement(resp)
	assert.NoError(s.T(), err)

	// Expecting an R4 response so we'll evaluate some fields to reflect that
	assert.Equal(s.T(), fhir.FHIRVersion4_0_1, cs.FhirVersion)
	assert.Equal(s.T(), 1, len(cs.Rest))
	assert.Equal(s.T(), 2, len(cs.Rest[0].Resource))
	assert.Len(s.T(), cs.Instantiates, 2)
	assert.Contains(s.T(), cs.Instantiates[0], "/v2/fhir/metadata/")
	resourceData := []struct {
		rt           fhir.ResourceType
		opName       string
		opDefinition string
	}{
		{fhir.ResourceTypePatient, "patient-export", "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export"},
		{fhir.ResourceTypeGroup, "group-export", "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export"},
	}

	for _, rd := range resourceData {
		var rr *fhir.CapabilityStatementRestResource
		for _, r := range cs.Rest[0].Resource {
			if r.Type == rd.rt {
				rr = &r
				break
			}
		}
		assert.NotNil(s.T(), rr)
		assert.Equal(s.T(), 1, len(rr.Operation))
		assert.Equal(s.T(), rd.opName, rr.Operation[0].Name)
		assert.Equal(s.T(), rd.opDefinition, rr.Operation[0].Definition)
	}

	// Need to validate our security.extensions manually since the extension
	// field does not have support for polymorphic types
	// See: https://github.com/samply/golang-fhir-models/issues/1
	var obj map[string]interface{}
	err = json.Unmarshal(resp, &obj)
	assert.NoError(s.T(), err)

	targetExtension := obj["rest"].([]interface{})[0].(map[string]interface{})["security"].(map[string]interface{})["extension"].([]interface{})[0].(map[string]interface{})
	subExtension := targetExtension["extension"].([]interface{})[0].(map[string]interface{})

	assert.Equal(s.T(), "http://fhir-registry.smarthealthit.org/StructureDefinition/oauth-uris", targetExtension["url"])
	assert.Equal(s.T(), "token", subExtension["url"])
	assert.Equal(s.T(), ts.URL, subExtension["valueUri"])

}

func (s *APITestSuite) TestResourceTypes() {
	tests := []struct {
		name          string
		resourceTypes []string
		statusCode    int
	}{
		{"Supported type - Patient", []string{"Patient"}, http.StatusAccepted},
		{"Supported type - Coverage", []string{"Coverage"}, http.StatusAccepted},
		{"Supported type - Patient,Coverage", []string{"Patient", "Coverage"}, http.StatusAccepted},
		{"Unsupported type - EOB", []string{"ExplanationOfBenefit"}, http.StatusBadRequest},
		{"Unsupported type - default", nil, http.StatusBadRequest},
	}

	for idx, handler := range []http.HandlerFunc{v2.BulkGroupRequest, v2.BulkPatientRequest} {
		for _, tt := range tests {
			s.T().Run(fmt.Sprintf("%s-%d", tt.name, idx), func(t *testing.T) {
				rr := httptest.NewRecorder()

				ep := "Group"
				if idx == 1 {
					ep = "Patient"
				}

				u, err := url.Parse(fmt.Sprintf("/api/v2/%s/$export", ep))
				assert.NoError(t, err)

				q := u.Query()
				if len(tt.resourceTypes) > 0 {
					q.Add("_type", strings.Join(tt.resourceTypes, ","))

				}
				u.RawQuery = q.Encode()
				req := httptest.NewRequest("GET", u.String(), nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("groupId", "all")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

				ad := s.getAuthData()
				req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

				handler(rr, req)
				assert.Equal(t, tt.statusCode, rr.Code)
				fmt.Println(rr.Body.String())
			})
		}
	}
}

func (s *APITestSuite) getAuthData() (data auth.AuthData) {
	var aco models.ACO
	s.db.First(&aco, "uuid = ?", acoUnderTest)
	return auth.AuthData{ACOID: acoUnderTest, CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}
}
