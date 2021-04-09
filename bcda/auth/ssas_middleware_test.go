package auth_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/conf"
)

var (
	originalSSASURL       string
	originalPublicSSASURL string
	originalSSASUseTLS    string
)

type SSASMiddlewareTestSuite struct {
	suite.Suite
	server      *httptest.Server
	token       *jwt.Token
	tokenString string
	ad          auth.AuthData
}

func (s *SSASMiddlewareTestSuite) createRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(auth.ParseToken)
	router.With(auth.RequireTokenAuth).Get("/", func(w http.ResponseWriter, r *http.Request) {
		ad := r.Context().Value(auth.AuthDataContextKey).(auth.AuthData)
		render.JSON(w, r, ad)
	})

	return router
}

func (s *SSASMiddlewareTestSuite) SetupSuite() {
	s.server = httptest.NewServer(s.createRouter())

	originalSSASURL = conf.GetEnv("SSAS_URL")
	originalPublicSSASURL = conf.GetEnv("SSAS_PUBLIC_URL")
	originalSSASUseTLS = conf.GetEnv("SSAS_USE_TLS")
}

func (s *SSASMiddlewareTestSuite) TearDownSuite() {
	s.server.Close()
	conf.SetEnv(s.T(), "SSAS_URL", originalSSASURL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", originalPublicSSASURL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", originalSSASUseTLS)
}

func (s *SSASMiddlewareTestSuite) TestSSASToken() {
	req, err := http.NewRequest("GET", s.server.URL, nil)
	require.NotNil(s.T(), req, "req not created; ", err)

	s.ad = auth.AuthData{}

	s.token, s.tokenString, err = auth.MockSSASToken()
	assert.NotNil(s.T(), s.tokenString, "token creation error; ", err)
	auth.MockSSASServer(s.tokenString)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.tokenString))
	client := s.server.Client()
	resp, err := client.Do(req)
	require.Nil(s.T(), err, "request failed; ", err)
	assert.Equal(s.T(), "200 OK", resp.Status)

	b, _ := ioutil.ReadAll(resp.Body)
	assert.NotZero(s.T(), len(b), "no content in response body")
	var ad auth.AuthData
	_ = json.Unmarshal(b, &ad)
	assert.NotEmpty(s.T(), ad)
	assert.Equal(s.T(), "A9995", ad.CMSID)
	assert.Equal(s.T(), "dbbd1ce1-ae24-435c-807d-ed45953077d3", ad.ACOID)
}

func TestSSASMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(SSASMiddlewareTestSuite))
}
