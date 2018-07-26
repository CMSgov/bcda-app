package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
	server *httptest.Server
}

func (suite *RouterTestSuite) SetupTest() {
	suite.server = httptest.NewServer(NewRouter())
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

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
