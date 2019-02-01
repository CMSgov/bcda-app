package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi"

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
	assert.Nil(suite.T(), err, fmt.Sprintf("Error when getting default route: %s", err))
	assert.NotContains(suite.T(), r, "404 page not found", "Default route returned wrong body")
	assert.Contains(suite.T(), r, "Beneficiary Claims Data API")
}

func (suite *RouterTestSuite) TestDataRoute() {
	_, err := suite.GetStringBody(suite.dataServer.URL + "/data")
	assert.Nil(suite.T(), err, fmt.Sprintf("Error when getting data route: %s", err))
	//assert.Equal(suite.T(), "Hello world!", r, "Default route returned wrong body")
}
func (suite *RouterTestSuite) TestFileServerRoute() {
	_, err := suite.GetStringBody(suite.apiServer.URL + "/api/v1/swagger")
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
