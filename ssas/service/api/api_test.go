package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/go-chi/chi"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
)

type APITestSuite struct {
	suite.Suite
	rr      *httptest.ResponseRecorder
	db      *gorm.DB
}

func (s *APITestSuite) SetupTest() {
	ssas.InitializeGroupModels()
	ssas.InitializeSystemModels()
	s.db = database.GetGORMDbConnection()
	s.rr = httptest.NewRecorder()
}

func (s *APITestSuite) TearDownTest() {
	database.Close(s.db)
}

func (s *APITestSuite) TestAuthRegisterEmpty() {
	regBody := strings.NewReader("")

	req, err := http.NewRequest("GET", "/auth/register", regBody)
	assert.Nil(s.T(), err)

	req = addRegDataContext(req, "T12123")
	http.HandlerFunc(RegisterSystem).ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func (s *APITestSuite) TestAuthRegisterBadJSON() {
	regBody := strings.NewReader("asdflkjghjkl")

	req, err := http.NewRequest("GET", "/auth/register", regBody)
	assert.Nil(s.T(), err)

	req = addRegDataContext(req, "T12123")
	http.HandlerFunc(RegisterSystem).ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func (s *APITestSuite) TestAuthRegisterSuccess() {
	groupID := "T12123"
	group := ssas.Group{GroupID: groupID}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	regBody := strings.NewReader(fmt.Sprintf(`{"client_id":"my_client_id","client_name":"my_client_name","scope":"%s","jwks":{"keys":[{"e":"AAEAAQ","n":"ok6rvXu95337IxsDXrKzlIqw_I_zPDG8JyEw2CTOtNMoDi1QzpXQVMGj2snNEmvNYaCTmFf51I-EDgeFLLexr40jzBXlg72quV4aw4yiNuxkigW0gMA92OmaT2jMRIdDZM8mVokoxyPfLub2YnXHFq0XuUUgkX_TlutVhgGbyPN0M12teYZtMYo2AUzIRggONhHvnibHP0CPWDjCwSfp3On1Recn4DPxbn3DuGslF2myalmCtkujNcrhHLhwYPP-yZFb8e0XSNTcQvXaQxAqmnWH6NXcOtaeWMQe43PNTAyNinhndgI8ozG3Hz-1NzHssDH_yk6UYFSszhDbWAzyqw","kty":"RSA"}]}}`,
		ssas.DEFAULT_SCOPE))

	req, err := http.NewRequest("GET", "/auth/register", regBody)
	assert.Nil(s.T(), err)

	req = addRegDataContext(req, "T12123")
	http.HandlerFunc(RegisterSystem).ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusCreated, s.rr.Code)
	fmt.Println("Response body:", s.rr.Body)

	j := map[string]string{}
	err = json.Unmarshal(s.rr.Body.Bytes(), &j)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "my_client_name", j["client_name"])

	err = ssas.CleanDatabase(group)
	assert.Nil(s.T(), err)
}

func TestAuthAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func addRegDataContext(req *http.Request, groupID string) *http.Request {
	rctx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rd := auth.AuthRegData{GroupID: groupID}
	req = req.WithContext(context.WithValue(req.Context(), "rd", rd))
	return req
}