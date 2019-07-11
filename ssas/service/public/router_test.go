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
	rb := strings.NewReader(`{"client_id":"evil_twin","client_name":"my evil twin","scope":"adcb","jwks":{"keys":[{"e":"AAEAAQ","n":"ok6rvXu95337IxsDXrKzlIqw_I_zPDG8JyEw2CTOtNMoDi1QzpXQVMGj2snNEmvNYaCTmFf51I-EDgeFLLexr40jzBXlg72quV4aw4yiNuxkigW0gMA92OmaT2jMRIdDZM8mVokoxyPfLub2YnXHFq0XuUUgkX_TlutVhgGbyPN0M12teYZtMYo2AUzIRggONhHvnibHP0CPWDjCwSfp3On1Recn4DPxbn3DuGslF2myalmCtkujNcrhHLhwYPP-yZFb8e0XSNTcQvXaQxAqmnWH6NXcOtaeWMQe43PNTAyNinhndgI8ozG3Hz-1NzHssDH_yk6UYFSszhDbWAzyqw","kty":"RSA"}]}}`)
	res := s.reqPublicRoute("POST", "/auth/register", rb)
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func TestAuthRouterTestSuite(t *testing.T) {
	suite.Run(t, new(PublicRouterTestSuite))
}
