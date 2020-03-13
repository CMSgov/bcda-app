package web

import (
	"context"
	"crypto/tls"
	"io/ioutil"
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
	req.Header.Add("Prefer", "respond-async")

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

func (s *MiddlewareTestSuite) TestConnectionCloseHeader() {
	router := chi.NewRouter()
	router.Use(ConnectionClose)
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Test router"))
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
		_, err := w.Write([]byte("Test router"))
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

	assert.NotEmpty(s.T(), result.Header.Get("Strict-Transport-Security"), "sets Security header")
	assert.Contains(s.T(), result.Header.Get("Cache-Control"), "must-revalidate", "ensures must-revalidate control added")
	assert.Equal(s.T(), result.Header.Get("Pragma"), "no-cache", "pragma header should be no-cache")
}

func (s *MiddlewareTestSuite) TearDownTest() {
	s.server.Close()
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

func mockTLSServerContext() context.Context {
	crt, err := ioutil.ReadFile("../../shared_files/localhost.crt")
	if err != nil {
		panic(err)
	}
	key, err := ioutil.ReadFile("../../shared_files/localhost.key")
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
