package health

import (
	"context"
	"encoding/json"
	"io"
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
	hc HealthChecker
}

func (s *HealthCheckerTestSuite) SetupSuite() {
	origSSASURL = conf.GetEnv("SSAS_URL")
	origPublicURL = conf.GetEnv("SSAS_PUBLIC_URL")
	origSSASUseTLS = conf.GetEnv("SSAS_USE_TLS")
	origSSASClientID = conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	origSSASSecret = conf.GetEnv("BCDA_SSAS_SECRET")
}

func (s *HealthCheckerTestSuite) SetupTest() {
	s.hc = NewHealthChecker(nil)
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

// makeTestCombinedServer creates a test server that handles both /token and /introspect endpoints
// For tests that only need one endpoint, pass appropriate defaults for the other endpoint
func makeTestCombinedServer(tokenStatusCode int, tokenResponse string, introspectStatusCode int, introspectActive bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" && r.Method == "POST" {
			_, _, ok := r.BasicAuth()
			if !ok && tokenStatusCode == http.StatusOK {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(tokenStatusCode)
			_, _ = io.WriteString(w, tokenResponse)
		} else if r.URL.Path == "/introspect" && r.Method == "POST" {
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
	combinedServer := makeTestCombinedServer(
		http.StatusOK,
		`{"access_token":"test-token","expires_in":"3600","token_type":"bearer"}`,
		http.StatusOK,
		true,
	)
	defer combinedServer.Close()

	conf.SetEnv(s.T(), "SSAS_URL", combinedServer.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", combinedServer.URL)
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

	combinedServer := makeTestCombinedServer(
		http.StatusOK,
		`{"access_token":"test-token","expires_in":"3600","token_type":"bearer"}`,
		http.StatusOK,
		true,
	)
	defer combinedServer.Close()

	conf.SetEnv(s.T(), "SSAS_URL", combinedServer.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", combinedServer.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.True(s.T(), ok)
	assert.Equal(s.T(), "ok", result)
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_TokenRequestFailure() {
	server := makeTestCombinedServer(
		http.StatusUnauthorized,
		`{"error":"Unauthorized"}`,
		http.StatusOK,
		false,
	)
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.False(s.T(), ok)
	assert.Equal(s.T(), "Failed to get BCDA admin token", result)
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_InvalidTokenResponse() {
	server := makeTestCombinedServer(
		http.StatusOK,
		`invalid json`,
		http.StatusOK,
		false,
	)
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.False(s.T(), ok)
	assert.Equal(s.T(), "Failed to get BCDA admin token", result)
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_EmptyToken() {
	server := makeTestCombinedServer(
		http.StatusOK,
		`{"access_token":"","expires_in":"3600","token_type":"bearer"}`,
		http.StatusOK,
		false,
	)
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", "test-client-id")
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", "test-secret")

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.False(s.T(), ok)
	assert.Equal(s.T(), "Empty access token in response", result)
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_IntrospectFailure() {
	combinedServer := makeTestCombinedServer(
		http.StatusOK,
		`{"access_token":"test-token","expires_in":"3600","token_type":"bearer"}`,
		http.StatusInternalServerError,
		false,
	)
	defer combinedServer.Close()

	conf.SetEnv(s.T(), "SSAS_URL", combinedServer.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", combinedServer.URL)
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

func (s *HealthCheckerTestSuite) TestGetTokenWithRetry_Success() {
	server := makeTestCombinedServer(
		http.StatusOK,
		`{"access_token":"test-token","expires_in":"3600","token_type":"bearer"}`,
		http.StatusOK,
		false,
	)
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := ssasClient.NewSSASClient()
	s.Require().NoError(err)

	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.CtxTransactionKey, "test")
	req, err := http.NewRequestWithContext(ctx, "GET", "/", nil)
	s.Require().NoError(err)

	tokenInfo, err := s.hc.getTokenWithRetry(client, "test-client-id", "test-secret", req)

	assert.NoError(s.T(), err)
	assert.Contains(s.T(), tokenInfo, "test-token")
}

func (s *HealthCheckerTestSuite) TestGetTokenWithRetry_NonRetryableError() {
	server := makeTestCombinedServer(
		http.StatusUnauthorized,
		`{"error":"Unauthorized"}`,
		http.StatusOK,
		false,
	)
	defer server.Close()

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	client, err := ssasClient.NewSSASClient()
	s.Require().NoError(err)

	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.CtxTransactionKey, "test")
	req, err := http.NewRequestWithContext(ctx, "GET", "/", nil)
	s.Require().NoError(err)

	tokenInfo, err := s.hc.getTokenWithRetry(client, "test-client-id", "test-secret", req)

	assert.Error(s.T(), err)
	assert.Empty(s.T(), tokenInfo)
	// Should not retry on 401, so should fail immediately
}

func (s *HealthCheckerTestSuite) TestIntrospectWithRetry_Success() {
	server := makeTestCombinedServer(
		http.StatusOK,
		`{}`,
		http.StatusOK,
		true,
	)
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

	result, err := s.hc.introspectWithRetry(client, ctx, "test-token")

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)

	var introspectResp struct {
		Active bool `json:"active"`
	}
	err = json.Unmarshal(result, &introspectResp)
	assert.NoError(s.T(), err)
	assert.True(s.T(), introspectResp.Active)
}

func (s *HealthCheckerTestSuite) TestIntrospectWithRetry_NonRetryableError() {
	server := makeTestCombinedServer(
		http.StatusOK,
		`{}`,
		http.StatusBadRequest,
		false,
	)
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

	result, err := s.hc.introspectWithRetry(client, ctx, "test-token")

	assert.Error(s.T(), err)
	assert.Nil(s.T(), result)
	// Should not retry on 4xx, so should fail immediately
}

func (s *HealthCheckerTestSuite) TestIsSsasIntrospectOK_ConcurrentAccess() {
	// Test that concurrent access to cache is safe
	combinedServer := makeTestCombinedServer(
		http.StatusOK,
		`{"access_token":"test-token","expires_in":"3600","token_type":"bearer"}`,
		http.StatusOK,
		true,
	)
	defer combinedServer.Close()

	conf.SetEnv(s.T(), "SSAS_URL", combinedServer.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", combinedServer.URL)
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
	s.hc.introspectCache.result = "Failed to get BCDA admin token"
	s.hc.introspectCache.ok = false
	s.hc.introspectCache.timestamp = time.Now()
	s.hc.introspectCache.mu.Unlock()

	result, ok := s.hc.IsSsasIntrospectOK()

	assert.False(s.T(), ok)
	assert.Equal(s.T(), "Failed to get BCDA admin token", result)
}
