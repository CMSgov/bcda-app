package logging_test

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/logging"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/suite"
)

type LoggingMiddlewareTestSuite struct {
	suite.Suite
	server *httptest.Server
}

func (s *LoggingMiddlewareTestSuite) SetupTest() {
	os.Setenv("BCDA_REQUEST_LOG", "bcda-req-test.log")
	s.server = httptest.NewServer(s.CreateRouter())
}

func (s *LoggingMiddlewareTestSuite) CreateRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(logging.NewStructuredLogger())
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {})
	return r
}

func (s *LoggingMiddlewareTestSuite) TestLogRequest() {
	client := s.server.Client()

	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		s.Fail("Request error", err)
	}
	assert.Equal(s.T(), 200, resp.StatusCode)

	logFile, err := os.OpenFile(os.Getenv("BCDA_REQUEST_LOG"), os.O_RDONLY, os.ModePerm)
	if err != nil {
		s.Fail("File read error", err)
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
		assert.Equal(s.T(), "http", logFields["http_scheme"])
		assert.Equal(s.T(), "HTTP/1.1", logFields["http_proto"])
		assert.Equal(s.T(), "GET", logFields["http_method"])
		assert.NotEmpty(s.T(), logFields["remote_addr"])
		assert.NotEmpty(s.T(), logFields["user_agent"])
		assert.Equal(s.T(), s.server.URL+"/", logFields["uri"])
	}
}

func TestLoggingMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(LoggingMiddlewareTestSuite))
}

func (s *LoggingMiddlewareTestSuite) TearDownTest() {
	s.server.Close()
	os.Remove("bcda-req-test.log")
}
