package public

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	system       ssas.System
}

func (s *PublicRouterTestSuite) SetupSuite() {
	os.Setenv("DEBUG", "true")
	s.publicRouter = routes()
	ssas.InitializeSystemModels()
	s.db = ssas.GetGORMDbConnection()
	s.rr = httptest.NewRecorder()
	groupBytes := []byte(`{"group_id":"T1234","users":["fake_okta_id","abcdefg"]}`)
	gd := ssas.GroupData{}
	err := json.Unmarshal(groupBytes, &gd)
	assert.Nil(s.T(), err)
	s.group, err = ssas.CreateGroup(gd)
	if err != nil {
		s.FailNow("unable to create group: " + err.Error())
	}
	s.system = ssas.System{GroupID: s.group.GroupID, ClientID: "abcd1234"}
	if err := s.db.Create(&s.system).Error; err != nil {
		s.FailNow("unable to create system: " + err.Error())
	}
}

func (s *PublicRouterTestSuite) TearDownSuite() {
	err := ssas.CleanDatabase(s.group)
	assert.Nil(s.T(), err)
	ssas.Close(s.db)
}

func (s *PublicRouterTestSuite) reqPublicRoute(verb string, route string, body io.Reader, token string) *http.Response {
	req := httptest.NewRequest(strings.ToUpper(verb), route, body)
	req.Header.Add("x-group-id", s.group.GroupID)
	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	s.publicRouter.ServeHTTP(rr, req)
	return rr.Result()
}

func (s *PublicRouterTestSuite) TestTokenRoute() {
	res := s.reqPublicRoute("POST", "/token", nil, "")
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestRegisterRoute() {
	groupIDs := []string{"T1234", "T0001"}
	_, ts, _ := MintRegistrationToken("test_okta_id", groupIDs)
	rb := strings.NewReader(`{"client_id":"evil_twin","client_name":"my evil twin","scope":"bcda-api","jwks":{"keys":[{"e":"AAEAAQ","n":"ok6rvXu95337IxsDXrKzlIqw_I_zPDG8JyEw2CTOtNMoDi1QzpXQVMGj2snNEmvNYaCTmFf51I-EDgeFLLexr40jzBXlg72quV4aw4yiNuxkigW0gMA92OmaT2jMRIdDZM8mVokoxyPfLub2YnXHFq0XuUUgkX_TlutVhgGbyPN0M12teYZtMYo2AUzIRggONhHvnibHP0CPWDjCwSfp3On1Recn4DPxbn3DuGslF2myalmCtkujNcrhHLhwYPP-yZFb8e0XSNTcQvXaQxAqmnWH6NXcOtaeWMQe43PNTAyNinhndgI8ozG3Hz-1NzHssDH_yk6UYFSszhDbWAzyqw","kty":"RSA"}]}}`)
	res := s.reqPublicRoute("POST", "/register", rb, ts)
	assert.Equal(s.T(), http.StatusCreated, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestRegisterRouteNoToken() {
	rb := strings.NewReader(`{"client_id":"evil_twin","client_name":"my evil twin","scope":"bcda-api","jwks":{"keys":[{"e":"AAEAAQ","n":"ok6rvXu95337IxsDXrKzlIqw_I_zPDG8JyEw2CTOtNMoDi1QzpXQVMGj2snNEmvNYaCTmFf51I-EDgeFLLexr40jzBXlg72quV4aw4yiNuxkigW0gMA92OmaT2jMRIdDZM8mVokoxyPfLub2YnXHFq0XuUUgkX_TlutVhgGbyPN0M12teYZtMYo2AUzIRggONhHvnibHP0CPWDjCwSfp3On1Recn4DPxbn3DuGslF2myalmCtkujNcrhHLhwYPP-yZFb8e0XSNTcQvXaQxAqmnWH6NXcOtaeWMQe43PNTAyNinhndgI8ozG3Hz-1NzHssDH_yk6UYFSszhDbWAzyqw","kty":"RSA"}]}}`)
	res := s.reqPublicRoute("POST", "/register", rb, "")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestResetRoute() {
	groupIDs := []string{"T1234", "T0001"}
	_, ts, _ := MintRegistrationToken("test_okta_id", groupIDs)
	rb := strings.NewReader(fmt.Sprintf(`{"client_id":"%s"}`, s.system.ClientID))
	res := s.reqPublicRoute("POST", "/reset", rb, ts)
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestAuthnChallengeRoute() {
	_, ts, _ := MintMFAToken("fake_okta_id")
	rb := strings.NewReader(`{"login_id":"success@test.com","factor_type":"SMS"}`)
	res := s.reqPublicRoute("POST", "/authn/challenge", rb, ts)
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(res.Body)
	if err != nil {
		s.FailNow(err.Error())
	}
	body := buf.String()
	assert.Contains(s.T(), body, "request_sent")
}

func (s *PublicRouterTestSuite) TestAuthnChallengeRouteNoToken() {
	rb := strings.NewReader(`{"login_id":"success@test.com","factor_type":"SMS"}`)
	res := s.reqPublicRoute("POST", "/authn/challenge", rb, "")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestResetRouteNoToken() {
	rb := strings.NewReader(fmt.Sprintf(`{"client_id":"%s"}`, s.system.ClientID))
	res := s.reqPublicRoute("POST", "/reset", rb, "")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestAuthnRoute() {
	rb := strings.NewReader(`{"login_id":"success@test.com","password":"abcdefg"}`)
	res := s.reqPublicRoute("POST", "/authn", rb, "")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestAuthnVerifyRoute() {
	_, ts, _ := MintMFAToken("fake_okta_id")
	rb := strings.NewReader(`{"login_id":"success@test.com","factor_type":"SMS","passcode":"123456"}`)
	res := s.reqPublicRoute("POST", "/authn/verify", rb, ts)
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *PublicRouterTestSuite) TestAuthnVerifyRouteNoToken() {
	rb := strings.NewReader(`{"login_id":"success@test.com","factor_type":"SMS","passcode":"123456"}`)
	res := s.reqPublicRoute("POST", "/authn/verify", rb, "")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func TestPublicRouterTestSuite(t *testing.T) {
	suite.Run(t, new(PublicRouterTestSuite))
}
