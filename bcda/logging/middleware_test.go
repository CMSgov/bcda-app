package logging_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/logging"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/suite"
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
		acoID := "dbbd1ce1-ae24-435c-807d-ed45953077d3"
		subID := "82503a18-bf3b-436d-ba7b-bae09b7ffdff"
		tokenID := "665341c9-7d0c-4844-b66f-5910d9d0822f"

		token := jwt.New(jwt.SigningMethodRS512)
		token.Claims = jwt.MapClaims{
			"sub": subID,
			"aco": acoID,
			"id":  tokenID,
		}
		ctx := context.WithValue(req.Context(), "token", token)
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
	assert.Equal(s.T(), 200, resp.StatusCode)

	server.Close()

	logFile, err := os.OpenFile(os.Getenv("BCDA_REQUEST_LOG"), os.O_RDONLY, os.ModePerm)
	if err != nil {
		s.Fail("File read error")
	}
	defer logFile.Close()

	sc := bufio.NewScanner(logFile)
	for sc.Scan() {
		var logFields log.Fields
		err = json.Unmarshal(sc.Bytes(), &logFields)
		if err != nil {
			s.Fail("JSON unmarshal error", err)
		}
		assert.NotEmpty(s.T(), logFields["ts"])
		assert.Equal(s.T(), "https", logFields["http_scheme"])
		assert.Equal(s.T(), "HTTP/1.1", logFields["http_proto"])
		assert.Equal(s.T(), "GET", logFields["http_method"])
		assert.NotEmpty(s.T(), logFields["remote_addr"])
		assert.NotEmpty(s.T(), logFields["user_agent"])
		assert.Equal(s.T(), server.URL+"/", logFields["uri"])
		assert.Equal(s.T(), "dbbd1ce1-ae24-435c-807d-ed45953077d3", logFields["aco"])
		assert.Equal(s.T(), "82503a18-bf3b-436d-ba7b-bae09b7ffdff", logFields["sub"])
		assert.Equal(s.T(), "665341c9-7d0c-4844-b66f-5910d9d0822f", logFields["token_id"])
	}

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

	sc := bufio.NewScanner(logFile)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		var logFields log.Fields
		err = json.Unmarshal(sc.Bytes(), &logFields)
		if err != nil {
			s.Fail("JSON unmarshal error", err)
		}

		assert.NotEmpty(s.T(), logFields["ts"])
		assert.Equal(s.T(), "https", logFields["http_scheme"])
		assert.Equal(s.T(), "HTTP/1.1", logFields["http_proto"])
		assert.Equal(s.T(), "GET", logFields["http_method"])
		assert.NotEmpty(s.T(), logFields["remote_addr"])
		assert.NotEmpty(s.T(), logFields["user_agent"])
		assert.Equal(s.T(), server.URL+"/panic", logFields["uri"])
		assert.Equal(s.T(), "dbbd1ce1-ae24-435c-807d-ed45953077d3", logFields["aco"])
		assert.Equal(s.T(), "82503a18-bf3b-436d-ba7b-bae09b7ffdff", logFields["sub"])
		assert.Equal(s.T(), "665341c9-7d0c-4844-b66f-5910d9d0822f", logFields["token_id"])
		if lineNum == 2 {
			assert.Equal(s.T(), "Test", logFields["panic"])
			assert.NotEmpty(s.T(), logFields["stack"])
		}
	}

	server.Close()
	os.Remove("bcda-req-test.log")
	os.Setenv("BCDA_REQUEST_LOG", reqLogPathOrig)
}

func TestLoggingMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(LoggingMiddlewareTestSuite))
}
