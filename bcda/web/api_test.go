package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bgentry/que-go"
	fhirmodels "github.com/eug48/fhir/models"
	"github.com/go-chi/chi"
	"github.com/jackc/pgx"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	cclfUtils "github.com/CMSgov/bcda-app/bcda/cclf/testutils"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type APITestSuite struct {
	suite.Suite
	rr    *httptest.ResponseRecorder
	db    *gorm.DB
	reset func()
}

func (s *APITestSuite) SetupSuite() {
	s.reset = testUtils.SetUnitTestKeysForAuth() // needed until token endpoint moves to auth
}

func (s *APITestSuite) TearDownSuite() {
	s.reset()
}

func (s *APITestSuite) SetupTest() {
	models.InitializeGormModels()
	auth.InitializeGormModels() // needed until token endpoint moves to auth
	s.db = database.GetGORMDbConnection()
	s.rr = httptest.NewRecorder()
}

func (s *APITestSuite) TearDownTest() {
	database.Close(s.db)
}

func (s *APITestSuite) TestBulkEOBRequest() {
	bulkEOBRequestHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestHelper("Group", s)
}

func (s *APITestSuite) TestBulkEOBRequestNoBeneficiariesInACO() {
	bulkEOBRequestNoBeneficiariesInACOHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestNoBeneficiariesInACOHelper("Group", s)
}

func (s *APITestSuite) TestBulkEOBRequestMissingToken() {
	bulkEOBRequestMissingTokenHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestMissingTokenHelper("Group", s)
}

func (s *APITestSuite) TestBulkEOBRequestNoQueue() {
	bulkEOBRequestNoQueueHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestNoQueueHelper("Group", s)
}

func (s *APITestSuite) TestBulkPatientRequest() {
	bulkPatientRequestHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkPatientRequestHelper("Group", s)
}

func (s *APITestSuite) TestBulkCoverageRequest() {
	bulkCoverageRequestHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Group", s)
}

func (s *APITestSuite) TestBulkRequestInvalidType() {
	bulkRequestInvalidTypeHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkRequestInvalidTypeHelper("Group", s)
}

func (s *APITestSuite) TestBulkConcurrentRequest() {
	bulkConcurrentRequestHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkConcurrentRequestHelper("Group", s)
}

func (s *APITestSuite) TestBulkConcurrentRequestTime() {
	bulkConcurrentRequestTimeHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkConcurrentRequestTimeHelper("Group", s)
}

func (s *APITestSuite) TestValidateRequest() {
	validateRequestHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	validateRequestHelper("Group", s)
}

func bulkEOBRequestHelper(endpoint string, s *APITestSuite) {
	err := cclfUtils.ImportCCLFPackage("dev", "test")
	assert.Nil(s.T(), err)
	acoID := constants.DevACOUUID
	err = s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)
	user, err := models.CreateUser("api.go Test User", "testbulkeobrequest@example.com", uuid.Parse(acoID))
	if err != nil {
		s.T().Error(err)
	}

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit")
	ad := makeContextValues(acoID, user.UUID.String())
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		s.T().Error(err)
	}

	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   pgxcfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		s.T().Error(err)
	}
	defer pgxpool.Close()

	qc = que.NewClient(pgxpool)

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)

	s.db.Unscoped().Where("request_url = ?", requestUrl).Delete(models.Job{})
	s.db.Unscoped().Where("user_id = ?", user.UUID).Delete(models.Job{})
	s.db.Unscoped().Where("uuid = ?", user.UUID).Delete(models.User{})
}

func bulkEOBRequestNoBeneficiariesInACOHelper(endpoint string, s *APITestSuite) {
	userID := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
	acoID := "A40404F7-1EF2-485A-9B71-40FE7ACDCBC2"

	_, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit")
	ad := makeContextValues(acoID, userID)
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		s.T().Error(err)
	}

	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   pgxcfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		s.T().Error(err)
	}
	defer pgxpool.Close()

	qc = que.NewClient(pgxpool)

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusInternalServerError, s.rr.Code)
}

func bulkEOBRequestMissingTokenHelper(endpoint string, s *APITestSuite) {
	_, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit")

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusUnauthorized, s.rr.Code)

	var respOO fhirmodels.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), responseutils.TokenErr, respOO.Issue[0].Details.Coding[0].Display)
}

func bulkEOBRequestNoQueueHelper(endpoint string, s *APITestSuite) {
	qc = nil

	acoID := constants.SmallACOUUID
	user, err := models.CreateUser("api.go Test User", "testbulkrequestnoqueue@example.com", uuid.Parse(acoID))
	if err != nil {
		s.T().Error(err)
	}
	defer s.db.Unscoped().Where("uuid = ?", user.UUID).Delete(models.User{})

	_, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit")

	ad := makeContextValues(acoID, user.UUID.String())
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusInternalServerError, s.rr.Code)

	var respOO fhirmodels.OperationOutcome
	err = json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), responseutils.Processing, respOO.Issue[0].Details.Coding[0].Display)
}

func bulkPatientRequestHelper(endpoint string, s *APITestSuite) {
	err := cclfUtils.ImportCCLFPackage("dev", "test")
	assert.Nil(s.T(), err)
	acoID := constants.DevACOUUID
	user, err := models.CreateUser("api.go Test User", "testbulkpatientrequest@example.com", uuid.Parse(acoID))
	if err != nil {
		s.T().Error(err)
	}

	defer func() {
		s.db.Unscoped().Where("user_id = ?", user.UUID).Delete(models.Job{})
		s.db.Unscoped().Where("uuid = ?", user.UUID).Delete(models.User{})
	}()

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "Patient")

	ad := makeContextValues(acoID, user.UUID.String())
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		s.T().Error(err)
	}

	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   pgxcfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		s.T().Error(err)
	}
	defer pgxpool.Close()

	qc = que.NewClient(pgxpool)

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	s.db.Unscoped().Where("request_url = ?", requestUrl).Delete(models.Job{})
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
}

func bulkCoverageRequestHelper(endpoint string, s *APITestSuite) {
	acoID := constants.DevACOUUID
	user, err := models.CreateUser("api.go Test User", "testbulkcoveragerequest@example.com", uuid.Parse(acoID))
	if err != nil {
		s.T().Error(err)
	}

	defer func() {
		s.db.Unscoped().Where("user_id = ?", user.UUID).Delete(models.Job{})
		s.db.Unscoped().Where("uuid = ?", user.UUID).Delete(models.User{})
	}()

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "Coverage")

	ad := makeContextValues(acoID, user.UUID.String())
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		s.T().Error(err)
	}

	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   pgxcfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		s.T().Error(err)
	}
	defer pgxpool.Close()

	qc = que.NewClient(pgxpool)

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	s.db.Unscoped().Where("request_url = ?", requestUrl).Delete(models.Job{})
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
}

func bulkRequestInvalidTypeHelper(endpoint string, s *APITestSuite) {
	_, handlerFunc, req := bulkRequestHelper(endpoint+"/test/Foo/", "Foo")
	handlerFunc(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func bulkConcurrentRequestHelper(endpoint string, s *APITestSuite) {
	err := os.Setenv("DEPLOYMENT_TARGET", "prod")
	assert.Nil(s.T(), err)
	acoID := constants.DevACOUUID
	userID := "82503a18-bf3b-436d-ba7b-bae09b7ffd2f"
	err = s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit")

	j := models.Job{
		ACOID:      uuid.Parse(acoID),
		UserID:     uuid.Parse(userID),
		RequestURL: requestUrl,
		Status:     "In Progress",
		JobCount:   1,
	}
	s.db.Save(&j)

	ad := makeContextValues(acoID, userID)
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))
	pool := makeConnPool(s)
	defer pool.Close()

	// serve job
	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusTooManyRequests, s.rr.Code)

	// change status to Pending and serve job
	var job models.Job
	err = s.db.Find(&job, "id = ?", j.ID).Error
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), s.db.Model(&job).Update("status", "Pending").Error)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusTooManyRequests, s.rr.Code)

	// change status to Completed and serve job
	err = s.db.Find(&job, "id = ?", j.ID).Error
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), s.db.Model(&job).Update("status", "Completed").Error)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	var lastRequestJob models.Job
	s.db.Last(&lastRequestJob)
	s.db.Unscoped().Delete(&lastRequestJob)

	// change status to Failed and serve job
	err = s.db.Find(&job, "id = ?", j.ID).Error
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), s.db.Model(&job).Update("status", "Failed").Error)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	lastRequestJob = models.Job{}
	s.db.Last(&lastRequestJob)
	s.db.Unscoped().Delete(&lastRequestJob)

	// change status to Archived
	err = s.db.Find(&job, "id = ?", j.ID).Error
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), s.db.Model(&job).Update("status", "Archived").Error)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	lastRequestJob = models.Job{}
	s.db.Last(&lastRequestJob)
	s.db.Unscoped().Delete(&lastRequestJob)

	// change status to Expired
	err = s.db.Find(&job, "id = ?", j.ID).Error
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), s.db.Model(&job).Update("status", "Expired").Error)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	lastRequestJob = models.Job{}
	s.db.Last(&lastRequestJob)
	s.db.Unscoped().Delete(&lastRequestJob)

	// different aco same endpoint
	err = s.db.Find(&job, "id = ?", j.ID).Error
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), s.db.Model(&job).Updates(models.Job{ACOID: uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"), Status: "In Progress"}).Error)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	lastRequestJob = models.Job{}
	s.db.Last(&lastRequestJob)
	s.db.Unscoped().Delete(&lastRequestJob)

	// same aco different endpoint
	handler = http.HandlerFunc(handlerFunc)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)

	_, handlerFunc, req = bulkRequestHelper(endpoint, "Patient")
	ad = makeContextValues(acoID, userID)
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))
	handler = http.HandlerFunc(handlerFunc)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)

	// do another patient call behind this one
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusTooManyRequests, s.rr.Code)

	lastRequestJob = models.Job{}
	s.db.Last(&lastRequestJob)
	s.db.Unscoped().Delete(&lastRequestJob)

	os.Unsetenv("DEPLOYMENT_TARGET")
}

func bulkConcurrentRequestTimeHelper(endpoint string, s *APITestSuite) {
	err := os.Setenv("DEPLOYMENT_TARGET", "prod")
	assert.Nil(s.T(), err)
	acoID := constants.DevACOUUID
	userID := "82503a18-bf3b-436d-ba7b-bae09b7ffd2f"
	err = s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit")

	j := models.Job{
		ACOID:      uuid.Parse(acoID),
		UserID:     uuid.Parse(userID),
		RequestURL: requestUrl,
		Status:     "In Progress",
		JobCount:   1,
	}
	s.db.Save(&j)

	ad := makeContextValues(acoID, userID)
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))
	pool := makeConnPool(s)
	defer pool.Close()

	// serve job
	handler := http.HandlerFunc(handlerFunc)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusTooManyRequests, s.rr.Code)

	// change created_at timestamp
	var job models.Job
	err = s.db.Find(&job, "id = ?", j.ID).Error
	assert.Nil(s.T(), err)
	assert.Nil(s.T(), s.db.Model(&job).Update("created_at", job.CreatedAt.Add(-GetJobTimeout())).Error)
	s.rr = httptest.NewRecorder()
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	os.Unsetenv("DEPLOYMENT_TARGET")
}

func validateRequestHelper(endpoint string, s *APITestSuite) {
	_, _, req := bulkRequestHelper(endpoint, "")

	resourceTypes, err := validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 3, len(resourceTypes))
	for _, t := range resourceTypes {
		if t != "ExplanationOfBenefit" && t != "Patient" && t != "Coverage" {
			assert.Fail(s.T(), "Invalid Resource type found")
		}
	}

	_, _, req = bulkRequestHelper(endpoint, "ExplanationOfBenefit,Patient")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 2, len(resourceTypes))
	for _, t := range resourceTypes {
		if t != "ExplanationOfBenefit" && t != "Patient" {
			assert.Fail(s.T(), "Invalid Resource type found")
		}
	}

	_, _, req = bulkRequestHelper(endpoint, "Coverage,Patient")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 2, len(resourceTypes))
	for _, t := range resourceTypes {
		if t != "Coverage" && t != "Patient" {
			assert.Fail(s.T(), "Invalid Resource type found")
		}
	}

	_, _, req = bulkRequestHelper(endpoint, "ExplanationOfBenefit")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(resourceTypes))
	assert.Contains(s.T(), resourceTypes, "ExplanationOfBenefit")

	_, _, req = bulkRequestHelper(endpoint, "Patient")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(resourceTypes))
	assert.Contains(s.T(), resourceTypes, "Patient")

	_, _, req = bulkRequestHelper(endpoint, "Coverage")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(resourceTypes))
	assert.Contains(s.T(), resourceTypes, "Coverage")

	_, _, req = bulkRequestHelper(endpoint, "Practitioner")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), resourceTypes)
	assert.Equal(s.T(), responseutils.Error, err.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, err.Issue[0].Code)
	assert.Equal(s.T(), responseutils.RequestErr, err.Issue[0].Details.Coding[0].Display)

	_, _, req = bulkRequestHelper(endpoint, "Patient,Patient")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), resourceTypes)
	assert.Equal(s.T(), responseutils.Error, err.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, err.Issue[0].Code)
	assert.Equal(s.T(), responseutils.RequestErr, err.Issue[0].Details.Coding[0].Display)
}

func bulkRequestHelper(endpoint, resourceType string) (string, func(http.ResponseWriter, *http.Request), *http.Request) {
	var requestUrl string
	var handlerFunc http.HandlerFunc
	var req *http.Request

	if endpoint == "Patient" {
		if resourceType != "" {
			requestUrl = fmt.Sprintf("/api/v1/%s/$export?_type=%s", endpoint, resourceType)
		} else {
			requestUrl = fmt.Sprintf("/api/v1/%s/$export", endpoint)
		}
		req = httptest.NewRequest("GET", requestUrl, nil)
		handlerFunc = bulkPatientRequest
		return requestUrl, handlerFunc, req

	} else {
		endpoint = "Group/all"
		if resourceType != "" {
			requestUrl = fmt.Sprintf("/api/v1/%s/$export?_type=%s", endpoint, resourceType)
		} else {
			requestUrl = fmt.Sprintf("/api/v1/%s/$export", endpoint)
		}
		req = httptest.NewRequest("GET", requestUrl, nil)
		handlerFunc = bulkGroupRequest
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("groupId", groupAll)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		return requestUrl, handlerFunc, req
	}
}

func (s *APITestSuite) TestJobStatusInvalidJobID() {
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%s", "test"), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", "test")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	var respOO fhirmodels.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), responseutils.DbErr, respOO.Issue[0].Details.Coding[0].Display)
}

func (s *APITestSuite) TestJobStatusJobDoesNotExist() {
	jobID := "1234"
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%s", jobID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	var respOO fhirmodels.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), responseutils.DbErr, respOO.Issue[0].Details.Coding[0].Display)
}

func (s *APITestSuite) TestJobStatusPending() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Pending",
		JobCount:   1,
	}
	s.db.Save(&j)

	req, err := http.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)
	assert.Nil(s.T(), err)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), "Pending", s.rr.Header().Get("X-Progress"))
	assert.Equal(s.T(), "", s.rr.Header().Get("Expires"))
	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestJobStatusInProgress() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "In Progress",
		JobCount:   1,
	}
	s.db.Save(&j)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), "In Progress (0%)", s.rr.Header().Get("X-Progress"))
	assert.Equal(s.T(), "", s.rr.Header().Get("Expires"))

	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestJobStatusFailed() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Failed",
	}

	s.db.Save(&j)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusInternalServerError, s.rr.Code)

	s.db.Unscoped().Delete(&j)
}

// https://stackoverflow.com/questions/34585957/postgresql-9-3-how-to-insert-upper-case-uuid-into-table
func (s *APITestSuite) TestJobStatusCompleted() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Completed",
	}
	s.db.Save(&j)

	var expectedUrls []string

	for i := 1; i <= 10; i++ {
		fileName := fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
		expectedurl := fmt.Sprintf("%s/%s/%s", "http://example.com/data", fmt.Sprint(j.ID), fileName)
		expectedUrls = append(expectedUrls, expectedurl)
		jobKey := models.JobKey{JobID: j.ID, EncryptedKey: []byte("FOO"), FileName: fileName, ResourceType: "ExplanationOfBenefit"}
		err := s.db.Save(&jobKey).Error
		assert.Nil(s.T(), err)

	}
	assert.Equal(s.T(), 10, len(expectedUrls))

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)
	req.TLS = &tls.ConnectionState{}

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), "application/json", s.rr.Header().Get("Content-Type"))
	// There seems to be some slight difference in precision here.  Match on first 20 chars sb fine.
	assert.Equal(s.T(), j.CreatedAt.Add(GetJobTimeout()).String()[:20], s.rr.Header().Get("Expires")[:20])

	var rb bulkResponseBody
	err := json.Unmarshal(s.rr.Body.Bytes(), &rb)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), j.RequestURL, rb.RequestURL)
	assert.Equal(s.T(), true, rb.RequiresAccessToken)
	assert.Equal(s.T(), "ExplanationOfBenefit", rb.Files[0].Type)
	assert.Equal(s.T(), len(expectedUrls), len(rb.Files))
	// Order of these values is impossible to know so this is the only way
	for _, fileItem := range rb.Files {
		inOutput := false
		for _, expectedUrl := range expectedUrls {
			if fileItem.URL == expectedUrl {
				inOutput = true
				break
			}
		}
		assert.True(s.T(), inOutput)

	}

	encryptionEnabled := utils.GetEnvBool("ENABLE_ENCRYPTION", true)
	if encryptionEnabled {
		assert.NotNil(s.T(), rb.KeyMap)
	} else {
		assert.Nil(s.T(), rb.KeyMap)
	}
	assert.Empty(s.T(), rb.Errors)

	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestJobStatusCompletedErrorFileExists() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Completed",
	}
	s.db.Save(&j)
	fileName := fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
	jobKey := models.JobKey{
		JobID:        j.ID,
		FileName:     fileName,
		EncryptedKey: []byte("Encrypted Key"),
		ResourceType: "ExplanationOfBenefit",
	}
	s.db.Save(&jobKey)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)
	req.TLS = &tls.ConnectionState{}

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	f := fmt.Sprintf("%s/%s", os.Getenv("FHIR_PAYLOAD_DIR"), fmt.Sprint(j.ID))
	if _, err := os.Stat(f); os.IsNotExist(err) {
		err = os.MkdirAll(f, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	errFileName := strings.Split(jobKey.FileName, ".")[0]
	errFilePath := fmt.Sprintf("%s/%s/%s-error.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), fmt.Sprint(j.ID), errFileName)
	_, err := os.Create(errFilePath)
	if err != nil {
		s.T().Error(err)
	}

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Equal(s.T(), "application/json", s.rr.Header().Get("Content-Type"))

	var rb bulkResponseBody
	err = json.Unmarshal(s.rr.Body.Bytes(), &rb)
	if err != nil {
		s.T().Error(err)
	}

	dataurl := fmt.Sprintf("%s/%s/%s", "http://example.com/data", fmt.Sprint(j.ID), fileName)
	errorurl := fmt.Sprintf("%s/%s/%s-error.ndjson", "http://example.com/data", fmt.Sprint(j.ID), errFileName)

	assert.Equal(s.T(), j.RequestURL, rb.RequestURL)
	assert.Equal(s.T(), true, rb.RequiresAccessToken)
	assert.Equal(s.T(), "ExplanationOfBenefit", rb.Files[0].Type)
	assert.Equal(s.T(), dataurl, rb.Files[0].URL)
	assert.Equal(s.T(), "OperationOutcome", rb.Errors[0].Type)
	assert.Equal(s.T(), errorurl, rb.Errors[0].URL)

	s.db.Unscoped().Delete(&j)
	os.Remove(errFilePath)
}

func (s *APITestSuite) TestJobStatusExpired() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Expired",
	}

	s.db.Save(&j)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusGone, s.rr.Code)
	// There seems to be some slight difference in precision here.  Match on first 20 chars sb fine.
	assert.Equal(s.T(), j.CreatedAt.Add(GetJobTimeout()).String()[:20], s.rr.Header().Get("Expires")[:20])
	s.db.Unscoped().Delete(&j)
}

// THis job is old, but has not yet been marked as expired.
func (s *APITestSuite) TestJobStatusNotExpired() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Completed",
	}

	// s.db.Save(&j)
	j.UpdatedAt = time.Now().Add(-GetJobTimeout()).Add(-GetJobTimeout())
	s.db.Save(&j)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusGone, s.rr.Code)
	// There seems to be some slight difference in precision here.  Match on first 20 chars sb fine.
	assert.Equal(s.T(), j.UpdatedAt.Add(GetJobTimeout()).String()[:20], s.rr.Header().Get("Expires")[:20])
	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestJobStatusArchived() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Archived",
	}

	s.db.Save(&j)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3", "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F")
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusGone, s.rr.Code)
	// There seems to be some slight difference in precision here.  Match on first 20 chars sb fine.
	assert.Equal(s.T(), j.CreatedAt.Add(GetJobTimeout()).String()[:20], s.rr.Header().Get("Expires")[:20])
	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestServeData() {
	os.Setenv("FHIR_PAYLOAD_DIR", "../../bcdaworker/data/test")
	req := httptest.NewRequest("GET", "/data/test.ndjson", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("fileName", "test.ndjson")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := http.HandlerFunc(serveData)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.Contains(s.T(), s.rr.Body.String(), `{"resourceType": "Bundle", "total": 33, "entry": [{"resource": {"status": "active", "diagnosis": [{"diagnosisCodeableConcept": {"coding": [{"system": "http://hl7.org/fhir/sid/icd-9-cm", "code": "2113"}]},`)
}

func (s *APITestSuite) TestAuthTokenMissingAuthHeader() {

	req := httptest.NewRequest("POST", "/auth/token", nil)
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func (s *APITestSuite) TestAuthTokenMalformedAuthHeader() {
	req := httptest.NewRequest("POST", "/auth/token", nil)
	req.Header.Add("Authorization", "Basic not_an_encoded_client_and_secret")
	req.Header.Add("Accept", "application/json")
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func (s *APITestSuite) TestAuthTokenBadCredentials() {
	req := httptest.NewRequest("POST", "/auth/token", nil)
	req.SetBasicAuth("not_a_client", "not_a_secret")
	req.Header.Add("Accept", "application/json")
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusUnauthorized, s.rr.Code)
}

func (s *APITestSuite) TestAuthTokenSuccess() {
	// Create and verify new credentials
	t := TokenResponse{}
	outputPattern := regexp.MustCompile(`.+\n(.+)\n(.+)`)
	tokenResp, _ := auth.CreateAlphaToken(60, "Dev")
	assert.Regexp(s.T(), outputPattern, tokenResp)
	matches := outputPattern.FindSubmatch([]byte(tokenResp))
	clientID := string(matches[1])
	clientSecret := string(matches[2])
	assert.NotEmpty(s.T(), clientID)
	assert.NotEmpty(s.T(), clientSecret)

	// Test success
	req := httptest.NewRequest("POST", "/auth/token", nil)
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Add("Accept", "application/json")
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.NoError(s.T(), json.NewDecoder(s.rr.Body).Decode(&t))
	assert.NotEmpty(s.T(), t)
	assert.NotEmpty(s.T(), t.AccessToken)
}

func (s *APITestSuite) TestMetadata() {
	req := httptest.NewRequest("GET", "/api/v1/metadata", nil)
	req.TLS = &tls.ConnectionState{}

	handler := http.HandlerFunc(metadata)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
}

func (s *APITestSuite) TestGetVersion() {
	req := httptest.NewRequest("GET", "/_version", nil)

	handler := http.HandlerFunc(getVersion)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusOK, s.rr.Code)

	respMap := make(map[string]string)
	err := json.Unmarshal(s.rr.Body.Bytes(), &respMap)
	if err != nil {
		s.T().Error(err.Error())
	}

	assert.Equal(s.T(), "latest", respMap["version"])
}

func (s *APITestSuite) TestJobStatusWithWrongACO() {
	j := models.Job{
		ACOID:      uuid.Parse("dbbd1ce1-ae24-435c-807d-ed45953077d3"),
		UserID:     uuid.Parse("82503a18-bf3b-436d-ba7b-bae09b7ffd2f"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Pending",
	}
	s.db.Save(&j)

	req, err := http.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)
	assert.Nil(s.T(), err)

	handler := auth.RequireTokenJobMatch(http.HandlerFunc(jobStatus))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues(constants.SmallACOUUID, j.UserID.String())
	req = req.WithContext(context.WithValue(req.Context(), "ad", ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusNotFound, s.rr.Code)

	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestHealthCheck() {
	req, err := http.NewRequest("GET", "/_health", nil)
	assert.Nil(s.T(), err)
	handler := http.HandlerFunc(healthCheck)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
}

func (s *APITestSuite) TestHealthCheckWithBadDatabaseURL() {
	// Mock database.LogFatal() to allow execution to continue despite bad URL
	origLogFatal := database.LogFatal
	defer func() { database.LogFatal = origLogFatal }()
	database.LogFatal = func(args ...interface{}) {
		fmt.Println("FATAL (NO-OP)")
	}
	dbURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", dbURL)
	os.Setenv("DATABASE_URL", "not-a-database")
	req, err := http.NewRequest("GET", "/_health", nil)
	assert.Nil(s.T(), err)
	handler := http.HandlerFunc(healthCheck)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadGateway, s.rr.Code)
}

func (s *APITestSuite) TestAuthInfoDefault() {

	// get original provider so we can reset at the end of the test
	originalProvider := auth.GetProviderName()

	// set provider to bogus value and make sure default (alpha) is retrieved
	auth.SetProvider("bogus")
	req := httptest.NewRequest("GET", "/_auth", nil)
	handler := http.HandlerFunc(getAuthInfo)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	respMap := make(map[string]string)
	err := json.Unmarshal(s.rr.Body.Bytes(), &respMap)
	if err != nil {
		s.T().Error(err.Error())
	}
	assert.Equal(s.T(), "alpha", respMap["auth_provider"])

	// set provider back to original value
	auth.SetProvider(originalProvider)
}

func (s *APITestSuite) TestAuthInfoAlpha() {

	// get original provider so we can reset at the end of the test
	originalProvider := auth.GetProviderName()

	// set provider to alpha and make sure alpha is retrieved
	auth.SetProvider("alpha")
	req := httptest.NewRequest("GET", "/_auth", nil)
	handler := http.HandlerFunc(getAuthInfo)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	respMap := make(map[string]string)
	err := json.Unmarshal(s.rr.Body.Bytes(), &respMap)
	if err != nil {
		s.T().Error(err.Error())
	}
	assert.Equal(s.T(), "alpha", respMap["auth_provider"])

	// set provider back to original value
	auth.SetProvider(originalProvider)
}

func (s *APITestSuite) TestAuthInfoOkta() {

	// get original provider so we can reset at the end of the test
	originalProvider := auth.GetProviderName()

	// set provider to okta and make sure okta is retrieved
	auth.SetProvider("okta")
	req := httptest.NewRequest("GET", "/_auth", nil)
	handler := http.HandlerFunc(getAuthInfo)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	respMap := make(map[string]string)
	err := json.Unmarshal(s.rr.Body.Bytes(), &respMap)
	if err != nil {
		s.T().Error(err.Error())
	}
	assert.Equal(s.T(), "okta", respMap["auth_provider"])

	// set provider back to original value
	auth.SetProvider(originalProvider)
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func makeContextValues(acoID string, userID string) (data auth.AuthData) {
	return auth.AuthData{ACOID: acoID, UserID: userID, TokenID: uuid.NewRandom().String()}
}

func makeConnPool(s *APITestSuite) *pgx.ConnPool {
	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		s.T().Error(err)
	}

	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   pgxcfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		s.T().Error(err)
	}
	qc = que.NewClient(pgxpool)

	return pgxpool
}
