package admin

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/jinzhu/gorm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
	router    http.Handler
	basicAuth string
	group     ssas.Group
}

func (s *RouterTestSuite) SetupSuite() {
	db := ssas.GetGORMDbConnection()
	defer ssas.Close(db)

	group := ssas.Group{GroupID: "admin"}
	if err := db.Save(&group).Error; err != nil {
		fmt.Println(err)
	}
	s.group = group

	system := makeSystem(db, "admin", "31e029ef-0e97-47f8-873c-0e8b7e7f99bf",
		"BCDA API Admin", "bcda-admin",
		"nbZ5oAnTlzyzeep46bL4qDGGuidXuYxs3xknVWBKjTI=:9s/Tnqvs8M7GN6VjGkLhCgjmS59r6TaVguos8dKV9lGqC1gVG8ywZVEpDMkdwOaj8GoNe4TU3jS+OZsK3kTfEQ==",
	)
	creds, err := system.ResetSecret("31e029ef-0e97-47f8-873c-0e8b7e7f99bf")
	if err != nil {
		s.FailNow(err.Error())
	}
	secret := creds.ClientSecret
	basicAuth := system.ClientID + ":" + secret
	s.basicAuth = base64.StdEncoding.EncodeToString([]byte(basicAuth))
}

func (s *RouterTestSuite) SetupTest() {
	s.router = routes()
}

func (s *RouterTestSuite) TearDownSuite() {
	db := ssas.GetGORMDbConnection()
	defer ssas.Close(db)

	ssas.CleanDatabase(s.group)
}

func (s *RouterTestSuite) TestRevokeToken() {
	req := httptest.NewRequest("DELETE", "/token/abc-123", nil)
	req.Header.Add("Authorization", "Basic "+s.basicAuth)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestPostGroup() {
	req := httptest.NewRequest("POST", "/group", nil)
	req.Header.Add("Authorization", "Basic "+s.basicAuth)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestGetGroup() {
	req := httptest.NewRequest("GET", "/group", nil)
	req.Header.Add("Authorization", "Basic "+s.basicAuth)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestPutGroup() {
	req := httptest.NewRequest("PUT", "/group/1", nil)
	req.Header.Add("Authorization", "Basic "+s.basicAuth)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestDeleteGroup() {
	req := httptest.NewRequest("DELETE", "/group/101", nil)
	req.Header.Add("Authorization", "Basic "+s.basicAuth)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestPostSystem() {
	req := httptest.NewRequest("POST", "/system", nil)
	req.Header.Add("Authorization", "Basic "+s.basicAuth)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestDeactivateSystemCredentials() {
	db := ssas.GetGORMDbConnection()
	defer db.Close()
	group := ssas.Group{GroupID: "delete-system-credentials-test-group"}
	db.Create(&group)
	system := ssas.System{GroupID: group.GroupID, ClientID: "delete-system-credentials-test-system"}
	db.Create(&system)
	systemID := strconv.FormatUint(uint64(system.ID), 10)

	req := httptest.NewRequest("DELETE", "/system/"+systemID+"/credentials", nil)
	req.Header.Add("Authorization", "Basic "+s.basicAuth)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	err := ssas.CleanDatabase(group)
	assert.Nil(s.T(), err)
}

func (s *RouterTestSuite) TestPutSystemCredentials() {
	db := ssas.GetGORMDbConnection()
	defer db.Close()
	group := ssas.Group{GroupID: "put-system-credentials-test-group"}
	db.Create(&group)
	system := ssas.System{GroupID: group.GroupID, ClientID: "put-system-credentials-test-system"}
	db.Create(&system)
	systemID := strconv.FormatUint(uint64(system.ID), 10)

	req := httptest.NewRequest("PUT", "/system/"+systemID+"/credentials", nil)
	req.Header.Add("Authorization", "Basic "+s.basicAuth)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	err := ssas.CleanDatabase(group)
	assert.Nil(s.T(), err)
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}

func makeSystem(db *gorm.DB, groupID, clientID, clientName, scope, hash string) ssas.System {
	pem := `-----BEGIN PUBLIC KEY-----
	MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArhxobShmNifzW3xznB+L
	I8+hgaePpSGIFCtFz2IXGU6EMLdeufhADaGPLft9xjwdN1ts276iXQiaChKPA2CK
	/CBpuKcnU3LhU8JEi7u/db7J4lJlh6evjdKVKlMuhPcljnIKAiGcWln3zwYrFCeL
	cN0aTOt4xnQpm8OqHawJ18y0WhsWT+hf1DeBDWvdfRuAPlfuVtl3KkrNYn1yqCgQ
	lT6v/WyzptJhSR1jxdR7XLOhDGTZUzlHXh2bM7sav2n1+sLsuCkzTJqWZ8K7k7cI
	XK354CNpCdyRYUAUvr4rORIAUmcIFjaR3J4y/Dh2JIyDToOHg7vjpCtNnNoS+ON2
	HwIDAQAB
	-----END PUBLIC KEY-----`
	system := ssas.System{GroupID: groupID, ClientID: clientID, ClientName: clientName, APIScope: scope}
	if err := db.Save(&system).Error; err != nil {
		ssas.Logger.Warn(err)
	}
	encryptionKey := ssas.EncryptionKey{
		Body:     pem,
		SystemID: system.ID,
	}
	if err := db.Save(&encryptionKey).Error; err != nil {
		ssas.Logger.Warn(err)
	}
	secret := ssas.Secret{
		Hash:     hash,
		SystemID: system.ID,
	}
	if err := db.Save(&secret).Error; err != nil {
		ssas.Logger.Warn(err)
	}

	return system
}
