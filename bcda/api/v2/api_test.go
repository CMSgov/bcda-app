package v2

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/go-chi/chi"
	"github.com/google/fhir/go/jsonformat"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	fhirresources "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/bundle_and_contained_resource_go_proto"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	acoUnderTest = constants.LargeACOUUID // Should use a different ACO compared to v1 tests
)

type APITestSuite struct {
	suite.Suite
	db *sql.DB
}

func (s *APITestSuite) SetupSuite() {
	origDate := conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", time.Now().Format("060102 15:01:01"))
	conf.SetEnv(s.T(), "BB_REQUEST_RETRY_INTERVAL_MS", "10")
	origBBCert := conf.GetEnv("BB_CLIENT_CERT_FILE")
	conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", "../../../shared_files/decrypted/bfd-dev-test-cert.pem")
	origBBKey := conf.GetEnv("BB_CLIENT_KEY_FILE")
	conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", "../../../shared_files/decrypted/bfd-dev-test-key.pem")

	s.T().Cleanup(func() {
		conf.SetEnv(s.T(), "CCLF_REF_DATE", origDate)
		conf.SetEnv(s.T(), "BB_CLIENT_CERT_FILE", origBBCert)
		conf.SetEnv(s.T(), "BB_CLIENT_KEY_FILE", origBBKey)
	})

	s.db = database.Connection

	// Use a mock to ensure that this test does not generate artifacts in the queue for other tests
	enqueuer := &queueing.MockEnqueuer{}
	enqueuer.On("AddJob", mock.Anything, mock.Anything).Return(nil)
	h.Enq = enqueuer
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func (s *APITestSuite) TestMetadataResponse() {
	ts := httptest.NewServer(http.HandlerFunc(Metadata))
	defer ts.Close()

	unmarshaller, err := jsonformat.NewUnmarshaller("UTC", jsonformat.R4)
	assert.NoError(s.T(), err)

	res, err := http.Get(ts.URL)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), "application/json", res.Header.Get("Content-Type"))
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	resp, err := ioutil.ReadAll(res.Body)
	assert.NoError(s.T(), err)

	resource, err := unmarshaller.Unmarshal(resp)
	assert.NoError(s.T(), err)
	cs := resource.(*fhirresources.ContainedResource).GetCapabilityStatement()

	// Expecting an R4 response so we'll evaluate some fields to reflect that
	assert.Equal(s.T(), fhircodes.FHIRVersionCode_V_4_0_1, cs.FhirVersion.Value)
	assert.Equal(s.T(), 1, len(cs.Rest))
	assert.Equal(s.T(), 2, len(cs.Rest[0].Resource))
	assert.Len(s.T(), cs.Instantiates, 2)
	assert.Contains(s.T(), cs.Instantiates[0].Value, "/v2/fhir/metadata")
	resourceData := []struct {
		rt           fhircodes.ResourceTypeCode_Value
		opName       string
		opDefinition string
	}{
		{fhircodes.ResourceTypeCode_PATIENT, "patient-export", "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export"},
		{fhircodes.ResourceTypeCode_GROUP, "group-export", "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export"},
	}

	for _, rd := range resourceData {
		for _, r := range cs.Rest[0].Resource {
			if r.Type.Value == rd.rt {
				assert.NotNil(s.T(), r)
				assert.Equal(s.T(), 1, len(r.Operation))
				assert.Equal(s.T(), rd.opName, r.Operation[0].Name.Value)
				assert.Equal(s.T(), rd.opDefinition, r.Operation[0].Definition.Value)
				break
			}
		}
	}

	extensions := cs.Rest[0].Security.Extension
	assert.Len(s.T(), extensions, 1)
	assert.Equal(s.T(), "http://fhir-registry.smarthealthit.org/StructureDefinition/oauth-uris", extensions[0].Url.Value)

	subExtensions := extensions[0].Extension
	assert.Len(s.T(), subExtensions, 1)
	assert.Equal(s.T(), "token", subExtensions[0].Url.Value)
	assert.Equal(s.T(), ts.URL+"/auth/token", subExtensions[0].GetValue().GetUri().Value)

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

	for idx, handler := range []http.HandlerFunc{BulkGroupRequest, BulkPatientRequest} {
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
				if rr.Code == http.StatusAccepted {
					assert.NotEmpty(t, rr.Header().Get("Content-Location"))
				}
			})
		}
	}
}

func (s *APITestSuite) getAuthData() (data auth.AuthData) {
	aco := postgrestest.GetACOByUUID(s.T(), s.db, uuid.Parse(acoUnderTest))
	return auth.AuthData{ACOID: acoUnderTest, CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}
}
