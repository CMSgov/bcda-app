package logging_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/logging"
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
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {})
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("Test")
	})
	return r
}

func contextToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ad := auth.AuthData{
			ACOID:   "dbbd1ce1-ae24-435c-807d-ed45953077d3",
			CMSID:   "A9995",
			TokenID: "665341c9-7d0c-4844-b66f-5910d9d0822f",
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
		s.Fail("Request error", err)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(req)
	if err != nil {
		s.Fail("Request error", err)
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
		assert.Equal("dbbd1ce1-ae24-435c-807d-ed45953077d3", logFields["aco_id"], "ACO in log entry should match the token.")
		assert.Equal("A9995", logFields["cms_id"], "CMS ID in log entry should match the token.")
		assert.Equal("665341c9-7d0c-4844-b66f-5910d9d0822f", logFields["token_id"], "Token ID in log entry should match the token.")
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
		s.Fail("Request error", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		s.Fail("Request error", err)
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
		s.Fail("Request error", err.Error())
	}

	_, err = client.Do(req)
	if err != nil {
		s.Fail("Request error", err.Error())
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
		assert.Equal("dbbd1ce1-ae24-435c-807d-ed45953077d3", logFields["aco_id"], "ACO in log entry should match the token.")
		assert.Equal("A9995", logFields["cms_id"], "CMS ID in log entry should match the token.")
		assert.Equal("665341c9-7d0c-4844-b66f-5910d9d0822f", logFields["token_id"], "Token ID in log entry should match the token.")
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
