package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const sampleGroup = `{  
	"id":"A12345",
	"name":"ACO Corp Systems",
	"users":[  
		"00uiqolo7fEFSfif70h7",
		"l0vckYyfyow4TZ0zOKek",
		"HqtEi2khroEZkH4sdIzj"
	],
	"scopes":[  
		"user-admin",
		"system-admin"
	],
	"resources":[  
		{  
			"id":"xxx",
			"name":"BCDA API",
			"scopes":[  
				"bcda-api"
			]
		},
		{  
			"id":"eft",
			"name":"EFT CCLF",
			"scopes":[  
				"eft-app:download",
				"eft-data:read"
			]
		}
	],
	"system":
		{  
		"client_id":"4tuhiOIFIwriIOH3zn",
		"software_id":"4NRB1-0XZABZI9E6-5SM3R",
		"client_name":"ACO System A",
		"client_uri":"https://www.acocorpsite.com"
		}
}`

type APITestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *APITestSuite) SetupSuite() {
	s.db = ssas.GetGORMDbConnection()
}

func (s *APITestSuite) TearDownSuite() {
	ssas.Close(s.db)
}

func (s *APITestSuite) TestCreateGroup() {
	req := httptest.NewRequest("POST", "/group", strings.NewReader(sampleGroup))
	handler := http.HandlerFunc(createGroup)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(s.T(), http.StatusCreated, rr.Result().StatusCode)
	assert.Equal(s.T(), "application/json", rr.Result().Header.Get("Content-Type"))
}

func (s *APITestSuite) TestCreateSystem() {
	group := ssas.Group{GroupID: "test-group-id"}
	err := s.db.Save(&group).Error
	if err != nil {
		s.FailNow("Error creating test data", err.Error())
	}

	req := httptest.NewRequest("POST", "/auth/system", strings.NewReader(`{"client_name": "Test Client", "group_id": "test-group-id", "scope": "bcda-api", "public_key": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArhxobShmNifzW3xznB+L\nI8+hgaePpSGIFCtFz2IXGU6EMLdeufhADaGPLft9xjwdN1ts276iXQiaChKPA2CK\n/CBpuKcnU3LhU8JEi7u/db7J4lJlh6evjdKVKlMuhPcljnIKAiGcWln3zwYrFCeL\ncN0aTOt4xnQpm8OqHawJ18y0WhsWT+hf1DeBDWvdfRuAPlfuVtl3KkrNYn1yqCgQ\nlT6v/WyzptJhSR1jxdR7XLOhDGTZUzlHXh2bM7sav2n1+sLsuCkzTJqWZ8K7k7cI\nXK354CNpCdyRYUAUvr4rORIAUmcIFjaR3J4y/Dh2JIyDToOHg7vjpCtNnNoS+ON2\nHwIDAQAB\n-----END PUBLIC KEY-----", "tracking_id": "T00000"}`))
	handler := http.HandlerFunc(createSystem)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(s.T(), http.StatusCreated, rr.Result().StatusCode)
	assert.Equal(s.T(), "application/json", rr.Result().Header.Get("Content-Type"))
	var result map[string]interface{}
	_ = json.Unmarshal(rr.Body.Bytes(), &result)
	assert.NotNil(s.T(), result["user_id"])
	assert.NotEmpty(s.T(), result["client_id"])
	assert.NotEmpty(s.T(), result["client_secret"])
	assert.NotNil(s.T(), result["token"])
	assert.Equal(s.T(), "Test Client", result["client_name"])

	_ = ssas.CleanDatabase(group)
}

func (s *APITestSuite) TestCreateSystem_InvalidRequest() {
	req := httptest.NewRequest("POST", "/auth/system", strings.NewReader("{ badJSON }"))
	handler := http.HandlerFunc(createSystem)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, rr.Result().StatusCode)
}

func (s *APITestSuite) TestCreateSystem_MissingRequiredParam() {
	req := httptest.NewRequest("POST", "/auth/system", strings.NewReader(`{"group_id": "T00001", "client_name": "Test Client 1", "scope": "bcda-api"}`))
	handler := http.HandlerFunc(createSystem)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, rr.Result().StatusCode)
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}
