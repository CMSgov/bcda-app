package admin

import (
	"encoding/json"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type APITestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *APITestSuite) SetupSuite() {
	s.db = database.GetGORMDbConnection()
}

func (s *APITestSuite) TearDownSuite() {
	database.Close(s.db)
}

func (s *APITestSuite) TestCreateSystem() {
	group := ssas.Group{GroupID: "T00000"}
	s.db.Save(&group)

	req := httptest.NewRequest("POST", "/auth/system", strings.NewReader(`{"group_id": "T00000", "client_id": "a1b2c3", "client_name": "Test Client", "client_uri": "", "scope": "bcda-api", "public_key": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArhxobShmNifzW3xznB+L\nI8+hgaePpSGIFCtFz2IXGU6EMLdeufhADaGPLft9xjwdN1ts276iXQiaChKPA2CK\n/CBpuKcnU3LhU8JEi7u/db7J4lJlh6evjdKVKlMuhPcljnIKAiGcWln3zwYrFCeL\ncN0aTOt4xnQpm8OqHawJ18y0WhsWT+hf1DeBDWvdfRuAPlfuVtl3KkrNYn1yqCgQ\nlT6v/WyzptJhSR1jxdR7XLOhDGTZUzlHXh2bM7sav2n1+sLsuCkzTJqWZ8K7k7cI\nXK354CNpCdyRYUAUvr4rORIAUmcIFjaR3J4y/Dh2JIyDToOHg7vjpCtNnNoS+ON2\nHwIDAQAB\n-----END PUBLIC KEY-----"}`))
	handler := http.HandlerFunc(createSystem)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(s.T(), http.StatusCreated, rr.Result().StatusCode)
	assert.Equal(s.T(), "application/json", rr.Result().Header.Get("Content-Type"))
	var result map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &result)
	assert.NotNil(s.T(), result["user_id"])
	assert.Equal(s.T(), "a1b2c3", result["client_id"])
	assert.NotEmpty(s.T(), result["client_secret"])
	assert.NotNil(s.T(), result["token"])
	assert.Equal(s.T(), "Test Client", result["client_name"])
}

func (s *APITestSuite) TestCreateSystem_InvalidRequest() {
	group := ssas.Group{GroupID: "T00000"}
	s.db.Save(&group)

	req := httptest.NewRequest("POST", "/auth/system", strings.NewReader("{ badJSON }"))
	handler := http.HandlerFunc(createSystem)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, rr.Result().StatusCode)
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}