package logging_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/logging"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
)

type LoggingMiddlewareTestSuite struct {
	suite.Suite
}

func (s *LoggingMiddlewareTestSuite) CreateRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, contextToken, logging.NewStructuredLogger(), middleware.Recoverer)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// Base server route for logging tests to be checked, blank return for overrides
	})
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("Test")
	})
	return r
}

func contextToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ad := auth.AuthData{
			ACOID:   constants.TestACOID,
			CMSID:   "A9995",
			TokenID: constants.TestTokenID,
		}

		ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func (s *LoggingMiddlewareTestSuite) TestLogRequest() {
	hook := test.NewLocal(testUtils.GetLogger(log.Request))
	server := httptest.NewTLSServer(s.CreateRouter())
	client := server.Client()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		s.Fail(constants.TestReqErr, err)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(req)
	if err != nil {
		s.Fail(constants.TestReqErr, err)
	}

	assert := assert.New(s.T())
	assert.Equal(200, resp.StatusCode)

	server.Close()

	assert.Len(hook.AllEntries(), 2)
	var logFields logrus.Fields
	for _, entry := range hook.AllEntries() {
		logFields = entry.Data

		assert.NotEmpty(logFields["ts"], "Log entry should have a value for field `ts`.")
		// TODO: Solution for go-chi logging middleware relying on Request.TLS
		// assert.Equal(s.T(), "https", logFields["http_scheme"])
		assert.Equal("HTTP/1.1", logFields["http_proto"], "Log entry should contain the HTTP protocol.")
		assert.Equal("GET", logFields["http_method"], "Log entry should contain the HTTP method.")
		assert.NotEmpty(logFields["remote_addr"], "Log entry should contain the remote address.")
		assert.NotEmpty(logFields["user_agent"], "Log entry should contain the user agent.")
		// TODO: Solution for go-chi logging middleware relying on Request.TLS
		// assert.Equal(s.T(), server.URL+"/", logFields["uri"])
		assert.Equal(constants.TestACOID, logFields["aco_id"], "ACO in log entry should match the token.")
		assert.Equal("A9995", logFields["cms_id"], "CMS ID in log entry should match the token.")
		assert.Equal(constants.TestTokenID, logFields["token_id"], "Token ID in log entry should match the token.")
		assert.Equal("gzip", logFields["accept_encoding"])
	}
}

func (s *LoggingMiddlewareTestSuite) TestNoLogFile() {
	reqLogPathOrig := conf.GetEnv("BCDA_REQUEST_LOG")
	conf.SetEnv(s.T(), "BCDA_REQUEST_LOG", "")
	server := httptest.NewServer(s.CreateRouter())
	client := server.Client()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		s.Fail(constants.TestReqErr, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		s.Fail(constants.TestReqErr, err)
	}
	assert.Equal(s.T(), 200, resp.StatusCode)

	server.Close()
	conf.SetEnv(s.T(), "BCDA_REQUEST_LOG", reqLogPathOrig)
}

func (s *LoggingMiddlewareTestSuite) TestPanic() {
	hook := test.NewLocal(testUtils.GetLogger(log.Request))

	server := httptest.NewTLSServer(s.CreateRouter())
	client := server.Client()

	req, err := http.NewRequest("GET", server.URL+"/panic", nil)
	if err != nil {
		s.Fail(constants.TestReqErr, err.Error())
	}

	_, err = client.Do(req)
	if err != nil {
		s.Fail(constants.TestReqErr, err.Error())
	}

	server.Close()

	assert := assert.New(s.T())

	assert.Len(hook.AllEntries(), 2)
	var logFields logrus.Fields
	for _, entry := range hook.AllEntries() {
		logFields = entry.Data

		assert.NotEmpty(logFields["ts"], "Log entry should have a value for field `ts`.")
		// TODO: Solution for go-chi logging middleware relying on Request.TLS
		// assert.Equal(s.T(), "https", logFields["http_scheme"])
		assert.Equal("HTTP/1.1", logFields["http_proto"], "Log entry should contain the HTTP protocol.")
		assert.Equal("GET", logFields["http_method"], "Log entry should contain the HTTP method.")
		assert.NotEmpty(logFields["remote_addr"], "Log entry should contain the remote address.")
		assert.NotEmpty(logFields["user_agent"], "Log entry should contain the user agent.")
		// TODO: Solution for go-chi logging middleware relying on Request.TLS
		// assert.Equal(s.T(), server.URL+"/panic", logFields["uri"])
		assert.Equal(constants.TestACOID, logFields["aco_id"], "ACO in log entry should match the token.")
		assert.Equal("A9995", logFields["cms_id"], "CMS ID in log entry should match the token.")
		assert.Equal(constants.TestTokenID, logFields["token_id"], "Token ID in log entry should match the token.")
	}

	panicFields := hook.LastEntry().Data
	assert.Equal("Test", panicFields["panic"])
	assert.NotEmpty(panicFields["stack"])
}

func (s *LoggingMiddlewareTestSuite) TestRedact() {
	uri := "https://www.example.com/api/endpoint?Authorization=Bearer%20abcdef.12345"
	redacted := logging.Redact(uri)
	assert.Equal(s.T(), "https://www.example.com/api/endpoint?Authorization=Bearer%20<redacted>", redacted)

	uri = "https://www.example.com/api/endpoint?Authorization=Bearer%20abcdef.12345&Authorization=Bearer%2019dgks8gfasdf&foo=bar"
	redacted = logging.Redact(uri)
	assert.Equal(s.T(), "https://www.example.com/api/endpoint?Authorization=Bearer%20<redacted>&Authorization=Bearer%20<redacted>&foo=bar", redacted)

	uri = "https://www.example.com/api/endpoint?foo=bar"
	redacted = logging.Redact(uri)
	assert.Equal(s.T(), uri, redacted)
}

func TestLoggingMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(LoggingMiddlewareTestSuite))
}

type mockLogger struct {
	Logger logrus.FieldLogger
	entry  *logging.StructuredLoggerEntry
}

func (l *mockLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	return l.entry
}

func TestResourceTypeLogging(t *testing.T) {
	testCases := []struct {
		jobID        string
		ResourceType interface{}
	}{
		{
			jobID:        "1234",
			ResourceType: "Coverage",
		},
		{
			jobID:        "bad",
			ResourceType: nil,
		},
		{
			jobID:        "4321",
			ResourceType: nil,
		},
	}

	for _, test := range testCases {
		req := httptest.NewRequest("GET", fmt.Sprintf("/data/%s/blob.ndjson", test.jobID), nil)
		repository := &models.MockRepository{}
		if test.ResourceType != nil {
			j := &models.JobKey{ID: 1, JobID: 1234, FileName: constants.TestBlobFileName, ResourceType: test.ResourceType.(string)}
			repository.On("GetJobKey", testUtils.CtxMatcher, uint(1234), constants.TestBlobFileName).Return(j, nil)
		} else {
			repository.On("GetJobKey", testUtils.CtxMatcher, mock.MatchedBy(func(i interface{}) bool { return true }), constants.TestBlobFileName).Return(nil, errors.New("expected error"))
		}

		entry := &logging.StructuredLoggerEntry{Logger: log.Request}

		logger := logging.ResourceTypeLogger{
			Repository: repository,
		}

		r := chi.NewRouter()
		r.With(
			middleware.RequestLogger(&mockLogger{Logger: log.Request, entry: entry}),
			logger.LogJobResourceType).Get("/data/{jobID}/{fileName}", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Test route handler method for retrieving resources
		}))

		rw := httptest.NewRecorder()
		r.ServeHTTP(rw, req)
		testEntry := entry.Logger.WithField("test", nil)
		if respT := testEntry.Data["resource_type"]; respT != test.ResourceType {
			t.Error("Failed to find resource_type in logs", respT, testEntry)
		}
	}
}

func TestMiddlewareLogCtx(t *testing.T) {

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := r.Context().Value(logging.CommonLogCtxKey).(*logging.StructuredLoggerEntry)
		if val == nil {
			t.Error("no log context")
		}

	})

	handlerToTest := contextToken(middleware.RequestID(logging.NewCommonLogFields(nextHandler)))
	req := httptest.NewRequest("GET", "http://testing", nil)
	handlerToTest.ServeHTTP(httptest.NewRecorder(), req)

}

func TestLogEntrySetField(t *testing.T) {
	ctx := context.Background()
	ctx = logging.LogEntrySetField(ctx, "request_id", "123456")
	ctx = logging.LogEntrySetField(ctx, "cms_id", "A0000")
	ctxEntryAppend := ctx.Value(logging.CommonLogCtxKey).(*logging.StructuredLoggerEntry)
	entry := ctxEntryAppend.Logger.WithField("test", "entry")

	if cmsId, ok := entry.Data["cms_id"]; ok {
		if cmsId != "A0000" {
			t.Errorf("unexpected value for cms_id")
		}
	} else {
		t.Errorf("key cms_id does not exist")
	}
	if reqId, ok := entry.Data["request_id"]; ok {
		if reqId != "123456" {
			t.Errorf("unexpected value for request_id")
		}
	} else {
		t.Errorf("key request_id does not exist")
	}
}
