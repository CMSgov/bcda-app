package api

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type APITestSuite struct {
	suite.Suite
	rr      *httptest.ResponseRecorder
	db      *gorm.DB
	backend *auth.AlphaBackend
	reset   func()
}

func (s *APITestSuite) SetupSuite() {
	private := testUtils.SetAndRestoreEnvKey("JWT_PRIVATE_KEY_FILE", "../../../shared_files/api_unit_test_auth_private.pem")
	public := testUtils.SetAndRestoreEnvKey("JWT_PUBLIC_KEY_FILE", "../../../shared_files/api_unit_test_auth_public.pem")
	s.reset = func() {
		private()
		public()
	}
	s.backend = auth.InitAlphaBackend()
}

func (s *APITestSuite) TearDownSuite() {
	s.reset()
}

func (s *APITestSuite) SetupTest() {
	models.InitializeGormModels()
	auth.InitializeGormModels()
	s.db = database.GetGORMDbConnection()
	s.rr = httptest.NewRecorder()
}

func (s *APITestSuite) TearDownTest() {
	database.Close(s.db)
}

func (s *APITestSuite) TestAuthToken() {
	var aco models.ACO
	err := s.db.Where("uuid = ?", constants.DEVACOUUID).First(&aco).Error
	assert.Nil(s.T(), err)
	aco.AlphaSecret = ""
	s.db.Save(&aco)

	// Missing authorization header
	req := httptest.NewRequest("POST", "/auth/token", nil)
	handler := http.HandlerFunc(GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)

	// Malformed authorization header
	s.rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/auth/token", nil)
	req.Header.Add("Authorization", "Basic not_an_encoded_client_and_secret")
	req.Header.Add("Accept", "application/json")
	handler = http.HandlerFunc(GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)

	// Invalid credentials
	s.rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/auth/token", nil)
	req.SetBasicAuth("not_a_client", "not_a_secret")
	req.Header.Add("Accept", "application/json")
	handler = http.HandlerFunc(GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusUnauthorized, s.rr.Code)

	// Success!?
	s.rr = httptest.NewRecorder()
	t := TokenResponse{}
	creds, err := auth.GetProvider().RegisterClient(constants.DEVACOUUID)
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), creds.ClientID)
	assert.NotEmpty(s.T(), creds.ClientSecret)

	req = httptest.NewRequest("POST", "/auth/token", nil)
	req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	req.Header.Add("Accept", "application/json")
	handler = http.HandlerFunc(GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.NoError(s.T(), json.NewDecoder(s.rr.Body).Decode(&t))
	assert.NotEmpty(s.T(), t)
	assert.NotEmpty(s.T(), t.AccessToken)
}

func (s *APITestSuite) TestAuthRegisterEmpty() {
	regBody := strings.NewReader("")

	req, err := http.NewRequest("GET", "/auth/register", regBody)
	assert.Nil(s.T(), err)

	handler := ParseRegToken(http.HandlerFunc(RegisterSystem))
	req = addRegDataContext(req, "T12123")
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func (s *APITestSuite) TestAuthRegisterBadJSON() {
	regBody := strings.NewReader("asdflkjghjkl")

	req, err := http.NewRequest("GET", "/auth/register", regBody)
	assert.Nil(s.T(), err)

	handler := ParseRegToken(http.HandlerFunc(RegisterSystem))
	req = addRegDataContext(req, "T12123")
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func TestAuthAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func addRegDataContext(req *http.Request, groupID string) *http.Request {
	rctx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rd := auth.AuthRegData{GroupID: groupID}
	req = req.WithContext(context.WithValue(req.Context(), "rd", rd))
	return req
}
