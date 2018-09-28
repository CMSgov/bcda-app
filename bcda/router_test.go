package main

import (
	"fmt"
	"github.com/go-chi/chi"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
	server *httptest.Server
	rr     *httptest.ResponseRecorder
}

func (suite *RouterTestSuite) SetupTest() {
	os.Setenv("DEBUG", "true")
	suite.server = httptest.NewServer(NewRouter())
	suite.rr = httptest.NewRecorder()
}

func (suite *RouterTestSuite) TearDownTest() {
	suite.server.Close()
}

func (suite *RouterTestSuite) GetStringBody(url string) (string, error) {
	client := suite.server.Client()
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
	r, err := suite.GetStringBody(suite.server.URL)
	assert.Nil(suite.T(), err, fmt.Sprintf("Error when getting default route: %s", err))
	assert.Equal(suite.T(), "Hello world!", r, "Default route returned wrong body")
}

func (suite *RouterTestSuite) TestDataRoute() {
	_, err := suite.GetStringBody(suite.server.URL + "/data")
	assert.Nil(suite.T(), err, fmt.Sprintf("Error when getting data route: %s", err))
	//assert.Equal(suite.T(), "Hello world!", r, "Default route returned wrong body")
}
func (suite *RouterTestSuite) TestFileServerRoute() {
	_, err := suite.GetStringBody(suite.server.URL + "/api/v1/swagger")
	assert.Nil(suite.T(), err, fmt.Sprintf("Error when getting swagger route: %s", err))
	r := chi.NewRouter()
	// Set up a bad route.  DON'T do this in real life
	assert.Panics(suite.T(), func() {
		FileServer(r, "/api/v1/swagger{}", http.Dir("./swaggerui"))
	})

}
func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
