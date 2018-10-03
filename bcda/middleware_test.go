package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
	router.With(ValidateBulkRequestHeaders).Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Test router"))
		if err != nil {
			log.Fatal(err)
		}
	})

	s.server = httptest.NewServer(router)
}

func (s *MiddlewareTestSuite) TestValidateBulkRequestHeaders() {
	client := s.server.Client()

	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Accept", "application/fhir+json")
	req.Header.Add("Prefer", "respond-async")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(s.T(), 200, resp.StatusCode)
}

func (s *MiddlewareTestSuite) TestValidateBulkRequestHeadersInvalidAccept() {
	client := s.server.Client()

	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Accept", "")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(s.T(), 400, resp.StatusCode)

	req.Header.Set("Accept", "test")

	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(s.T(), 400, resp.StatusCode)
}

func (s *MiddlewareTestSuite) TestValidateBulkRequestHeadersInvalidPrefer() {
	client := s.server.Client()

	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Accept", "application/fhir+json")
	req.Header.Add("Prefer", "")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(s.T(), 400, resp.StatusCode)

	req.Header.Set("Prefer", "test")

	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(s.T(), 400, resp.StatusCode)
}

func (s *MiddlewareTestSuite) TearDownTest() {
	s.server.Close()
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
