package middleware

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/service"
	logAPI "github.com/CMSgov/bcda-app/log"
	"github.com/go-chi/chi/v5"
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
	crt, err := os.ReadFile("../../../shared_files/localhost.crt")
	if err != nil {
		panic(err)
	}
	key, err := os.ReadFile("../../../shared_files/localhost.key")
	if err != nil {
		panic(err)
	}

	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Default middleware test route handler
		}),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
		ReadHeaderTimeout: 10 * time.Second,
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
		cfg := &service.Config{RunoutConfig: service.RunoutConfig{CutoffDurationDays: 180, ClaimThruDate: "2020-12-31"}, ACOConfigs: []service.ACOConfig{tt.ACOconfig}}
		assert.NoError(s.T(), cfg.ComputeFields())

		rr := httptest.NewRecorder()
		ACOMiddleware := ACOEnabled(cfg)

		ACOMiddleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// ACO middleware test route, blank return for overrides
		})).ServeHTTP(rr, testRequest(RequestParameters{}, tt.cmsid))
		assert.Equal(s.T(), tt.expected_code, rr.Code)
	}
}

func (s *MiddlewareTestSuite) TestACOEnabled_NoContextKey() {
	ctx := SetRequestParamsCtx(context.Background(), RequestParameters{})
	ctx = logAPI.NewStructuredLoggerEntry(log.New(), ctx)
	cfg := &service.Config{RunoutConfig: service.RunoutConfig{CutoffDurationDays: 180, ClaimThruDate: "2020-12-31"}, ACOConfigs: []service.ACOConfig{{Pattern: `TEST\d{4}`, Disabled: false}}}
	assert.NoError(s.T(), cfg.ComputeFields())

	rr := httptest.NewRecorder()
	ACOMiddleware := ACOEnabled(cfg)

	ACOMiddleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// ACO middleware test route, blank return for overrides
	})).ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/Patient", nil).WithContext(ctx))
	assert.Equal(s.T(), http.StatusInternalServerError, rr.Code)
}

func (s *MiddlewareTestSuite) TestACOEnabled_InvalidVersionsInPath() {
	tests := []struct {
		name         string
		path         string
		expected_err string
	}{
		{"Not Enough Parts", "/Patient", "not enough parts"},
		{"Invalid Version", "/api/v9/Patient", "unexpected API version"},
	}

	for _, tt := range tests {
		ctx := context.WithValue(context.Background(), auth.AuthDataContextKey, auth.AuthData{CMSID: "A1234"})
		ctx = SetRequestParamsCtx(ctx, RequestParameters{})
		ctx = logAPI.NewStructuredLoggerEntry(log.New(), ctx)

		cfg := &service.Config{RunoutConfig: service.RunoutConfig{CutoffDurationDays: 180, ClaimThruDate: "2020-12-31"}, ACOConfigs: []service.ACOConfig{{Pattern: `TEST\d{4}`, Disabled: false}}}
		assert.NoError(s.T(), cfg.ComputeFields())

		rr := httptest.NewRecorder()
		ACOMiddleware := ACOEnabled(cfg)

		ACOMiddleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// ACO middleware test route, blank return for overrides
		})).ServeHTTP(rr, httptest.NewRequest("GET", tt.path, nil).WithContext(ctx))

		assert.Equal(s.T(), http.StatusBadRequest, rr.Code)
		assert.Contains(s.T(), rr.Body.String(), tt.expected_err)
	}
}

func (s *MiddlewareTestSuite) TestV3AccessControl() {
	originalDeploymentTarget := os.Getenv("DEPLOYMENT_TARGET")
	defer func() {
		os.Setenv("DEPLOYMENT_TARGET", originalDeploymentTarget)
	}()
	os.Setenv("DEPLOYMENT_TARGET", "prod")

	tests := []struct {
		name          string
		cmsid         string
		enabledACOs   []string
		expected_code int
	}{
		{"V3AccessEnabled", "A1234", []string{"A1234", "A5678"}, http.StatusOK},
		{"V3AccessDisabled", "A9999", []string{"A1234", "A5678"}, http.StatusForbidden},
		{"EmptyEnabledList", "A1234", []string{}, http.StatusForbidden},
		{"NilEnabledList", "A1234", nil, http.StatusForbidden},
		{"CaseSensitive", "a1234", []string{"A1234", "A5678"}, http.StatusForbidden},
	}

	for _, tt := range tests {
		cfg := &service.Config{
			RunoutConfig:  service.RunoutConfig{CutoffDurationDays: 180, ClaimThruDate: "2020-12-31"},
			V3EnabledACOs: tt.enabledACOs,
		}

		rr := httptest.NewRecorder()
		V3Middleware := V3AccessControl(cfg)

		V3Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		})).ServeHTTP(rr, testRequest(RequestParameters{}, tt.cmsid))
		assert.Equal(s.T(), tt.expected_code, rr.Code)
	}
}

func (s *MiddlewareTestSuite) TestV1V2DenyControl() {
	tests := []struct {
		name         string
		acoID        string
		regexes      []string
		expectedCode int
	}{
		{"V1V2 deny empty", "A1234", []string{}, http.StatusOK},
		{"V1V2 deny list nil", "A1234", nil, http.StatusOK},
		{"V1V2 deny list mismatch", "A1234", []string{"D\\d{4}"}, http.StatusOK},
		{"V1V2 deny list mismatch specific", "A1234", []string{"A1235", "A1236"}, http.StatusOK},
		{"V1V2 deny list mismatch case sensitive", "a1234", []string{"A1234", "A5678"}, http.StatusOK},
		{"V1V2 deny list match", "A1234", []string{"A\\d{4}"}, http.StatusForbidden},
		{"V1V2 deny list match specific", "A1234", []string{"A1234"}, http.StatusForbidden},
		{"V1V2 deny all", "A9999", []string{".*"}, http.StatusForbidden},
		{"V1V2 deny many regexes", "A9999", []string{"A1234", "A9998", "A999\\d"}, http.StatusForbidden},
	}

	for _, tt := range tests {
		cfg := &service.Config{
			RunoutConfig:  service.RunoutConfig{CutoffDurationDays: 180, ClaimThruDate: "2020-12-31"},
			V1V2DenyRegexes: tt.regexes,
		}

		rr := httptest.NewRecorder()

		V1V2DenyControl(cfg)(http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {}),
		).ServeHTTP(rr, testRequest(RequestParameters{}, tt.acoID))

		assert.Equal(s.T(), tt.expectedCode, rr.Code)
	}
}

func (s *MiddlewareTestSuite) TestV1V2DenyControl_PanicOnBadRegex() {
	cfg := &service.Config{
		RunoutConfig:  service.RunoutConfig{CutoffDurationDays: 180, ClaimThruDate: "2020-12-31"},
		V1V2DenyRegexes: []string{"?!.*"}, // invalid regex (invalid target for quantifier)
	}
	assert.Panics(s.T(), func() { isACOV1V2DeniedAccess(cfg, "A1234") })
}

func testRequest(rp RequestParameters, cmsid string) *http.Request {
	ctx := context.WithValue(context.Background(), auth.AuthDataContextKey, auth.AuthData{CMSID: cmsid, ACOID: cmsid})
	ctx = SetRequestParamsCtx(ctx, rp)
	ctx = logAPI.NewStructuredLoggerEntry(log.New(), ctx)
	return httptest.NewRequest("GET", "/api/v1/Patient", nil).WithContext(ctx)
}
