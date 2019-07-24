package public

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/ssas"
)

type PublicRouterTestSuite struct {
	suite.Suite
	publicRouter http.Handler
	rr           *httptest.ResponseRecorder
	db           *gorm.DB
	group        ssas.Group
}

func (s *PublicRouterTestSuite) SetupSuite() {
	os.Setenv("DEBUG", "true")
	s.publicRouter = Routes()
	ssas.InitializeGroupModels()
	ssas.InitializeSystemModels()
	s.db = ssas.GetGORMDbConnection()
	s.rr = httptest.NewRecorder()
	s.group.GroupID = "T1234"
	err := s.db.Create(&s.group).Error
	if err != nil {
		s.FailNow(err.Error())
	}
}

func (s *PublicRouterTestSuite) TearDownTestSuite() {
	s.db.Unscoped().Delete(&s.group)
	ssas.Close(s.db)
}

func (s *PublicRouterTestSuite) reqPublicRoute(verb string, route string, body io.Reader) *http.Response {
	req := httptest.NewRequest(strings.ToUpper(verb), route, body)
	req.Header.Add("x-fake-token", s.group.GroupID)
	rr := httptest.NewRecorder()
	s.publicRouter.ServeHTTP(rr, req)
	return rr.Result()
}

func (s *PublicRouterTestSuite) TestTokenRoute() {
	res := s.reqPublicRoute("GET", "/token", nil)
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestRegisterRoute() {
	rb := strings.NewReader(`{"client_id":"evil_twin","client_name":"my evil twin","scope":"bcda-api","jwks":{"keys":[{"e":"AAEAAQ","n":"ok6rvXu95337IxsDXrKzlIqw_I_zPDG8JyEw2CTOtNMoDi1QzpXQVMGj2snNEmvNYaCTmFf51I-EDgeFLLexr40jzBXlg72quV4aw4yiNuxkigW0gMA92OmaT2jMRIdDZM8mVokoxyPfLub2YnXHFq0XuUUgkX_TlutVhgGbyPN0M12teYZtMYo2AUzIRggONhHvnibHP0CPWDjCwSfp3On1Recn4DPxbn3DuGslF2myalmCtkujNcrhHLhwYPP-yZFb8e0XSNTcQvXaQxAqmnWH6NXcOtaeWMQe43PNTAyNinhndgI8ozG3Hz-1NzHssDH_yk6UYFSszhDbWAzyqw","kty":"RSA"}]}}`)
	res := s.reqPublicRoute("POST", "/register", rb)
	assert.Equal(s.T(), http.StatusCreated, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestAuthnRoute() {
	rb := strings.NewReader(`{"cms_id":"success@test.com","password":"abcdefg"}`)
	res := s.reqPublicRoute("POST", "/authn", rb)
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestAuthnRequestRoute() {
	rb := strings.NewReader(`{"cms_id":"success@test.com","factor_type":"SMS"}`)
	res := s.reqPublicRoute("POST", "/authn/request", rb)
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestAuthnVerifyRoute() {
	rb := strings.NewReader(`{"cms_id":"success@test.com","factor_type":"SMS","passcode":"123456"}`)
	res := s.reqPublicRoute("POST", "/authn/verify", rb)
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func TestAuthRouterTestSuite(t *testing.T) {
	suite.Run(t, new(PublicRouterTestSuite))
}
