package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
	apiServer  *httptest.Server
	dataServer *httptest.Server
	rr         *httptest.ResponseRecorder
}

func (suite *RouterTestSuite) SetupTest() {
	os.Setenv("DEBUG", "true")
	suite.apiServer = httptest.NewServer(NewAPIRouter())
	suite.dataServer = httptest.NewServer(NewDataRouter())
	suite.rr = httptest.NewRecorder()
}

func (suite *RouterTestSuite) TearDownTest() {
	suite.apiServer.Close()
	suite.dataServer.Close()
}

func (suite *RouterTestSuite) GetStringBody(url string) (string, error) {
	client := suite.apiServer.Client()
	res, err := client.Get(url)
	if err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", bytes), nil
}

func (suite *RouterTestSuite) TestDefaultRoute() {
	r, err := suite.GetStringBody(suite.apiServer.URL)
	assert.Nil(suite.T(), err, fmt.Sprintf("error when getting default route: %s", err))
	assert.NotContains(suite.T(), r, "404 page not found", "Default route returned wrong body")
	assert.Contains(suite.T(), r, "Beneficiary Claims Data API")
}

func (suite *RouterTestSuite) TestDataRoute() {
	_, err := suite.GetStringBody(suite.dataServer.URL + "/data")
	assert.Nil(suite.T(), err, fmt.Sprintf("error when getting data route: %s", err))
	//assert.Equal(suite.T(), "Hello world!", r, "Default route returned wrong body")
}

func (suite *RouterTestSuite) TestAuthTokenRoute() {
	_, err := suite.GetStringBody(suite.apiServer.URL + "/auth/token")
	assert.Nil(suite.T(), err, fmt.Sprintf("error getting auth token route: %s", err))
}

func (suite *RouterTestSuite) TestFileServerRoute() {
	_, err := suite.GetStringBody(suite.apiServer.URL + "/api/v1/swagger")
	assert.Nil(suite.T(), err, fmt.Sprintf("error when getting swagger route: %s", err))
	r := chi.NewRouter()
	// Set up a bad route.  DON'T do this in real life
	assert.Panics(suite.T(), func() {
		FileServer(r, "/api/v1/swagger{}", http.Dir("./swaggerui"))
	})
}

func (suite *RouterTestSuite) TestHTTPServerRedirect() {
	router := NewHTTPRouter()

	// Redirect GET http requests to https
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()

	assert.Nil(suite.T(), err, "redirect GET http to https")
	assert.Equal(suite.T(), 301, res.StatusCode, "http to https redirect return correct status code")
	assert.Equal(suite.T(), "close", res.Header.Get("Connection"), "http to https redirect sets 'connection: close' header")
	assert.Contains(suite.T(), res.Header.Get("Location"), "https://", "location response header contains 'https://'")

	// Only respond to GET requests
	req, err = http.NewRequest("POST", "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res = w.Result()

	assert.Nil(suite.T(), err, "redirect POST http to https")
	assert.Equal(suite.T(), 405, res.StatusCode, "http to https redirect rejects POST requests")
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
