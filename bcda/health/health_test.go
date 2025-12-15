package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	ssasClient "github.com/CMSgov/bcda-app/bcda/auth/client"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/middleware"
)

var (
	origSSASURL      string
	origPublicURL    string
	origSSASUseTLS   string
	origSSASClientID string
	origSSASSecret   string
)

type HealthCheckerTestSuite struct {
	suite.Suite
	hc healthChecker
}

func (s *HealthCheckerTestSuite) SetupSuite() {
	origSSASURL = conf.GetEnv("SSAS_URL")
	origPublicURL = conf.GetEnv("SSAS_PUBLIC_URL")
	origSSASUseTLS = conf.GetEnv("SSAS_USE_TLS")
	origSSASClientID = conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	origSSASSecret = conf.GetEnv("BCDA_SSAS_SECRET")
}

func (s *HealthCheckerTestSuite) SetupTest() {
	s.hc = healthChecker{
		db:              nil,
		introspectCache: &introspectCache{},
	}
}

func (s *HealthCheckerTestSuite) TearDownTest() {
	conf.SetEnv(s.T(), "SSAS_URL", origSSASURL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", origPublicURL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", origSSASUseTLS)
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", origSSASClientID)
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", origSSASSecret)
}

func TestHealthCheckerTestSuite(t *testing.T) {
	suite.Run(t, new(HealthCheckerTestSuite))
}

// makeTestIntrospectServer creates a test server that handles the /introspect endpoint
func makeTestIntrospectServer(introspectStatusCode int, introspectActive bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/introspect" && r.Method == "POST" {
			_, _, ok := r.BasicAuth()
			if !ok && introspectStatusCode == http.StatusOK {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(introspectStatusCode)
			body, _ := json.Marshal(struct {
				Active bool `json:"active"`
			}{Active: introspectActive})
			_, _ = w.Write(body)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_MissingCredentials() {
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "")

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.False(s.T(), ok)
	assert.Equal(s.T(), "BCDA admin credentials not configured", result)
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_Success() {
	server := makeTestIntrospectServer(http.StatusOK, false) // active=false is fine, we just need 200 OK
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.True(s.T(), ok)
	assert.Equal(s.T(), "ok", result)
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_CacheHit() {
	// Set up cache with valid data
	s.hc.introspectCache.mu.Lock()
	s.hc.introspectCache.result = "ok"
	s.hc.introspectCache.ok = true
	s.hc.introspectCache.timestamp = time.Now()
	s.hc.introspectCache.mu.Unlock()

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.True(s.T(), ok)
	assert.Equal(s.T(), "ok", result)
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_CacheExpired() {
	// Set up cache with expired data
	s.hc.introspectCache.mu.Lock()
	s.hc.introspectCache.result = "old result"
	s.hc.introspectCache.ok = false
	s.hc.introspectCache.timestamp = time.Now().Add(-6 * time.Minute) // Expired
	s.hc.introspectCache.mu.Unlock()

	server := makeTestIntrospectServer(http.StatusOK, false)
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.True(s.T(), ok)
	assert.Equal(s.T(), "ok", result)
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_IntrospectFailure() {
	server := makeTestIntrospectServer(http.StatusInternalServerError, false)
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.False(s.T(), ok)
	assert.Equal(s.T(), "SSAS introspect check failed", result)
}

func (s *HealthCheckerTestSuite) TestIsRetryableError_RequestTimeout() {
	err := &customErrors.RequestTimeoutError{Msg: "timeout"}
	assert.True(s.T(), isRetryableError(err))
}

func (s *HealthCheckerTestSuite) TestIsRetryableError_UnexpectedSSASError_5xx() {
	err := &customErrors.UnexpectedSSASError{SsasStatusCode: 500}
	assert.True(s.T(), isRetryableError(err))

	err = &customErrors.UnexpectedSSASError{SsasStatusCode: 503}
	assert.True(s.T(), isRetryableError(err))
}

func (s *HealthCheckerTestSuite) TestIsRetryableError_UnexpectedSSASError_4xx() {
	err := &customErrors.UnexpectedSSASError{SsasStatusCode: 400}
	assert.False(s.T(), isRetryableError(err))

	err = &customErrors.UnexpectedSSASError{SsasStatusCode: 401}
	assert.False(s.T(), isRetryableError(err))
}

func (s *HealthCheckerTestSuite) TestIsRetryableError_SSASErrorUnauthorized() {
	err := &customErrors.SSASErrorUnauthorized{}
	assert.False(s.T(), isRetryableError(err))
}

func (s *HealthCheckerTestSuite) TestIsRetryableError_SSASErrorBadRequest() {
	err := &customErrors.SSASErrorBadRequest{}
	assert.False(s.T(), isRetryableError(err))
}

func (s *HealthCheckerTestSuite) TestIsRetryableError_InternalParsingError() {
	err := &customErrors.InternalParsingError{}
	assert.False(s.T(), isRetryableError(err))
}

func (s *HealthCheckerTestSuite) TestIsRetryableError_ConfigError() {
	err := &customErrors.ConfigError{}
	assert.False(s.T(), isRetryableError(err))
}

func (s *HealthCheckerTestSuite) TestIntrospectWithRetry_Success() {
	server := makeTestIntrospectServer(http.StatusOK, false) // active=false is fine, we just need 200 OK
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	client, err := ssasClient.NewSSASClient()
	s.Require().NoError(err)

	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.CtxTransactionKey, "test")

	// Use "None" as the token (old approach)
	result, err := s.hc.introspectWithRetry(client, ctx, "None")

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)

	var introspectResp struct {
		Active bool `json:"active"`
	}
	err = json.Unmarshal(result, &introspectResp)
	assert.NoError(s.T(), err)
	// "None" will result in active=false, but that's fine - we just need 200 OK response
	assert.False(s.T(), introspectResp.Active)
}

func (s *HealthCheckerTestSuite) TestIntrospectWithRetry_NonRetryableError() {
	server := makeTestIntrospectServer(http.StatusBadRequest, false)
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	client, err := ssasClient.NewSSASClient()
	s.Require().NoError(err)

	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.CtxTransactionKey, "test")

	// Use "None" as the token (old approach)
	result, err := s.hc.introspectWithRetry(client, ctx, "None")

	assert.Error(s.T(), err)
	assert.Nil(s.T(), result)
	// Should not retry on 4xx, so should fail immediately
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_ConcurrentAccess() {
	// Test that concurrent access to cache is safe
	server := makeTestIntrospectServer(http.StatusOK, false)
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	// Run multiple goroutines accessing the cache
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			result, ok := s.hc.IsSsasIntrospectOK()
			assert.True(s.T(), ok)
			assert.Equal(s.T(), "ok", result)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_SSASClientCreationFailure() {
	conf.SetEnv(s.T(), "SSAS_URL", "")
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", "")
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.False(s.T(), ok)
	assert.Equal(s.T(), "Failed to create SSAS client", result)
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_CacheWithFailedResult() {
	// Set up cache with failed result
	s.hc.introspectCache.mu.Lock()
	s.hc.introspectCache.result = "SSAS introspect check failed"
	s.hc.introspectCache.ok = false
	s.hc.introspectCache.timestamp = time.Now()
	s.hc.introspectCache.mu.Unlock()

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.False(s.T(), ok)
	assert.Equal(s.T(), "SSAS introspect check failed", result)
}
