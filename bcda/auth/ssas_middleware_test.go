package auth_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/models"
)

var (
	originalProvider      string
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
		ad := r.Context().Value("ad").(auth.AuthData)
		render.JSON(w, r, ad)
	})

	return router
}

func (s *SSASMiddlewareTestSuite) SetupSuite() {
	models.InitializeGormModels()
	s.server = httptest.NewServer(s.createRouter())

	originalSSASURL = os.Getenv("SSAS_URL")
	originalPublicSSASURL = os.Getenv("SSAS_PUBLIC_URL")
	originalSSASUseTLS = os.Getenv("SSAS_USE_TLS")

	originalProvider = auth.GetProviderName()
	auth.SetProvider("ssas")
	fmt.Println("testing with", auth.GetProviderName())
}

func (s *SSASMiddlewareTestSuite) TearDownSuite() {
	os.Setenv("SSAS_URL", originalSSASURL)
	os.Setenv("SSAS_PUBLIC_URL", originalPublicSSASURL)
	os.Setenv("SSAS_USE_TLS", originalSSASUseTLS)

	fmt.Println("restoring to", originalProvider)
	auth.SetProvider(originalProvider)
}

func (s *SSASMiddlewareTestSuite) TestSSASToken() {
	req, err := http.NewRequest("GET", s.server.URL, nil)
	require.NotNil(s.T(), err)

	s.ad = auth.AuthData{}

	s.token, s.tokenString, err = auth.MockSSASToken()
	assert.Nil(s.T(), err, "token signing error")
	auth.MockSSASServer(s.tokenString)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.tokenString))
	client := s.server.Client()
	resp, err := client.Do(req)
	require.Nil(s.T(), err)
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
