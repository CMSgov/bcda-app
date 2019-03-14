package logging_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/logging"
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
			UserID:  "82503a18-bf3b-436d-ba7b-bae09b7ffdff",
			TokenID: "665341c9-7d0c-4844-b66f-5910d9d0822f",
		}

		ctx := context.WithValue(req.Context(), "ad", ad)
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func (s *LoggingMiddlewareTestSuite) TestLogRequest() {
	reqLogPathOrig := os.Getenv("BCDA_REQUEST_LOG")
	os.Setenv("BCDA_REQUEST_LOG", "bcda-req-test.log")

	server := httptest.NewTLSServer(s.CreateRouter())
	client := server.Client()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		s.Fail("Request error", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		s.Fail("Request error", err)
	}

	assert := assert.New(s.T())
	assert.Equal(200, resp.StatusCode)

	server.Close()

	logFile, err := os.OpenFile(os.Getenv("BCDA_REQUEST_LOG"), os.O_RDONLY, os.ModePerm)
	if err != nil {
		s.Fail("File read error")
	}
	defer logFile.Close()

	sc := bufio.NewScanner(logFile)
	var logFields log.Fields
	for i := 0; i < 2; i++ {
		assert.True(sc.Scan())

		err = json.Unmarshal(sc.Bytes(), &logFields)
		if err != nil {
			s.Fail("JSON unmarshal error", err)
		}

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
		assert.Equal("82503a18-bf3b-436d-ba7b-bae09b7ffdff", logFields["user_id"], "Sub in log entry should match the token.")
		assert.Equal("665341c9-7d0c-4844-b66f-5910d9d0822f", logFields["token_id"], "Token ID in log entry should match the token.")
	}

	assert.False(sc.Scan(), "There should be no additional log entries.")

	os.Remove("bcda-req-test.log")
	os.Setenv("BCDA_REQUEST_LOG", reqLogPathOrig)
}

func (s *LoggingMiddlewareTestSuite) TestNoLogFile() {
	reqLogPathOrig := os.Getenv("BCDA_REQUEST_LOG")
	os.Setenv("BCDA_REQUEST_LOG", "")
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
	os.Setenv("BCDA_REQUEST_LOG", reqLogPathOrig)
}

func (s *LoggingMiddlewareTestSuite) TestPanic() {
	reqLogPathOrig := os.Getenv("BCDA_REQUEST_LOG")
	os.Setenv("BCDA_REQUEST_LOG", "bcda-req-test.log")

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

	logFile, err := os.OpenFile(os.Getenv("BCDA_REQUEST_LOG"), os.O_RDONLY, os.ModePerm)
	if err != nil {
		s.Fail("File read error")
	}
	defer logFile.Close()

	assert := assert.New(s.T())
	sc := bufio.NewScanner(logFile)
	var logFields log.Fields
	for i := 0; i < 2; i++ {
		assert.True(sc.Scan())

		err = json.Unmarshal(sc.Bytes(), &logFields)
		if err != nil {
			s.Fail("JSON unmarshal error", err)
		}

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
		assert.Equal("82503a18-bf3b-436d-ba7b-bae09b7ffdff", logFields["user_id"], "Sub in log entry should match the token.")
		assert.Equal("665341c9-7d0c-4844-b66f-5910d9d0822f", logFields["token_id"], "Token ID in log entry should match the token.")
		if i == 1 {
			assert.Equal("Test", logFields["panic"])
			assert.NotEmpty(logFields["stack"])
		}
	}

	assert.False(sc.Scan(), "There should be no additional log entries.")

	server.Close()
	os.Remove("bcda-req-test.log")
	os.Setenv("BCDA_REQUEST_LOG", reqLogPathOrig)
}

func TestLoggingMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(LoggingMiddlewareTestSuite))
}
