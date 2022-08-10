package middleware

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MiddlewareTestSuite struct {
	suite.Suite
	server *httptest.Server
}

func (s *MiddlewareTestSuite) SetupTest() {
	router := chi.NewRouter()
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(constants.TestRouter))
		if err != nil {
			log.Fatal(err)
		}
	})

	s.server = httptest.NewServer(router)
}

func (s *MiddlewareTestSuite) TestConnectionCloseHeader() {
	router := chi.NewRouter()
	router.Use(ConnectionClose)
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(constants.TestRouter))
		if err != nil {
			log.Fatal(err)
		}
	})

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	result := w.Result()

	assert.Equal(s.T(), "close", result.Header.Get("Connection"), "sets 'Connection: close' header")
}

func (s *MiddlewareTestSuite) TestSecurityHeader() {
	router := chi.NewRouter()
	router.Use(SecurityHeader)
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(constants.TestRouter))
		if err != nil {
			log.Fatal(err)
		}
	})

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Trick the request into thinking its being made over https
	ctx := mockTLSServerContext()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	result := w.Result()

	assert.NotEmpty(s.T(), result.Header.Get("Strict-Transport-Security"), "sets STS header")
	assert.NotEmpty(s.T(), result.Header.Get(constants.CacheControl), "sets cache control settings")
	assert.NotEmpty(s.T(), result.Header.Get("X-Content-Type-Options"), "sets x-content-type-options")
	assert.Equal(s.T(), result.Header.Get("Pragma"), "no-cache", "pragma header should be no-cache")
	assert.Equal(s.T(), result.Header.Get("X-Content-Type-Options"), "nosniff", "x-content-type header should be no-sniff")
	assert.Contains(s.T(), result.Header.Get(constants.CacheControl), "must-revalidate", "ensures must-revalidate control added")
	assert.Contains(s.T(), result.Header.Get(constants.CacheControl), "no-cache", "ensures no-cache control added")
	assert.Contains(s.T(), result.Header.Get(constants.CacheControl), "no-store", "ensures no-store control added")
	assert.Contains(s.T(), result.Header.Get(constants.CacheControl), "max-age=0", "ensures max-age=0 control added")

}

func (s *MiddlewareTestSuite) TearDownTest() {
	s.server.Close()
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

func mockTLSServerContext() context.Context {
	crt, err := ioutil.ReadFile("../../../shared_files/localhost.crt")
	if err != nil {
		panic(err)
	}
	key, err := ioutil.ReadFile("../../../shared_files/localhost.key")
	if err != nil {
		panic(err)
	}

	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	baseCtx := context.Background()
	ctx := context.WithValue(baseCtx, http.ServerContextKey, srv)

	return ctx
}

func (s *MiddlewareTestSuite) TestACOEnabled() {
	tests := []struct {
		name          string
		cmsid         string
		ACOconfig     service.ACOConfig
		expected_code int
	}{
		{"ACOIsEnabled", "TEST01234", service.ACOConfig{Pattern: `TEST\d{4}`, Disabled: false}, http.StatusOK},
		{"ACOIsDisabled", "TEST01234", service.ACOConfig{Pattern: `TEST\d{4}`, Disabled: true}, http.StatusUnauthorized},
		{"ACODNE", "Not_An_ACO", service.ACOConfig{Pattern: `TEST\d{4}`, Disabled: false}, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		cfg := &service.Config{AlrJobSize: 1000, RunoutConfig: service.RunoutConfig{CutoffDurationDays: 180, ClaimThruDate: "2020-12-31"}, ACOConfigs: []service.ACOConfig{tt.ACOconfig}}
		assert.NoError(s.T(), cfg.ComputeFields())

		rr := httptest.NewRecorder()
		ACOMiddleware := ACOEnabled(cfg)
		ACOMiddleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {})).
			ServeHTTP(rr, testRequest(RequestParameters{}, tt.cmsid))
		assert.Equal(s.T(), tt.expected_code, rr.Code)
	}
}

func testRequest(rp RequestParameters, cmsid string) *http.Request {
	ctx := context.WithValue(context.Background(), auth.AuthDataContextKey, auth.AuthData{CMSID: cmsid})
	ctx = NewRequestParametersContext(ctx, rp)
	return httptest.NewRequest("GET", "/api/v1/Patient", nil).WithContext(ctx)
}
