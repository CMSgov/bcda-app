package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

type APITestSuite struct {
	suite.Suite
	rr    *httptest.ResponseRecorder
	db    *gorm.DB
	reset func()
}

var origDate string

func (s *APITestSuite) SetupSuite() {
	s.reset = testUtils.SetUnitTestKeysForAuth() // needed until token endpoint moves to auth
	origDate = os.Getenv("CCLF_REF_DATE")
	os.Setenv("CCLF_REF_DATE", time.Now().Format("060102 15:01:01"))
}

func (s *APITestSuite) TearDownSuite() {
	os.Setenv("CCLF_REF_DATE", origDate)
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
	since := "2020-02-13T08:00:00.000-05:00"
	bulkEOBRequestHelper("Patient", "", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestHelper("Group/all", "", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkEOBRequestInvalidSinceFormat() {
	since := "invalidDate"
	bulkEOBRequestInvalidSinceFormatHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestInvalidSinceFormatHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkEOBRequestNoBeneficiariesInACO() {
	bulkEOBRequestNoBeneficiariesInACOHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestNoBeneficiariesInACOHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkEOBRequestMissingToken() {
	bulkEOBRequestMissingTokenHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestMissingTokenHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkEOBRequestNoQueue() {
	bulkEOBRequestNoQueueHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkEOBRequestNoQueueHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkPatientRequest() {
	since := "2020-02-13T08:00:00.000-05:00"
	bulkPatientRequestHelper("Patient", "", s)
	s.TearDownTest()
	s.SetupTest()
	bulkPatientRequestHelper("Group/all", "", s)
	s.TearDownTest()
	s.SetupTest()
	bulkPatientRequestHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkPatientRequestHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkPatientRequestInvalidSinceFormat() {
	since := "invalidDate"
	bulkPatientRequestInvalidSinceFormatHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkPatientRequestInvalidSinceFormatHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkCoverageRequest() {
	since := "2020-02-13T08:00:00.000-05:00"
	bulkCoverageRequestHelper("Patient", "", s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Group/all", "", s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkCoverageRequestInvalidSinceFormat() {
	since := "invalidDate"
	bulkCoverageRequestInvalidSinceFormatHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestInvalidSinceFormatHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkCoverageRequestInvalidSinceFormatEscapeCharacterFormat() {
	since := "2020-03-01T00%3A%2000%3A00.000-00%3A00"
	bulkCoverageRequestInvalidSinceFormatHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestInvalidSinceFormatHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkCoverageRequestInvalidSinceFormatMissingTimeZone() {
	since := "2020-02-13T08:00:00.000"
	bulkCoverageRequestInvalidSinceFormatHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestInvalidSinceFormatHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkCoverageRequestInvalidSinceFormatInvalidTime() {
	since := "2020-02-13T33:00:00.000-05:00"
	bulkCoverageRequestInvalidSinceFormatHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestInvalidSinceFormatHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkCoverageRequestInvalidSinceFormatInvalidDate() {
	since := "2020-20-13T08:00:00.000-05:00"
	bulkCoverageRequestInvalidSinceFormatHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestInvalidSinceFormatHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkCoverageRequestInvalidSinceFormatOnlyDate() {
	since := "2020-03-01"
	bulkCoverageRequestInvalidSinceFormatHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestInvalidSinceFormatHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkCoverageRequestInvalidSinceFormatOnlyInvalidDate() {
	since := "2020-04-0"
	bulkCoverageRequestInvalidSinceFormatHelper("Patient", since, s)
	s.TearDownTest()
	s.SetupTest()
	bulkCoverageRequestInvalidSinceFormatHelper("Group/all", since, s)
}

func (s *APITestSuite) TestBulkRequestInvalidType() {
	bulkRequestInvalidTypeHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkRequestInvalidTypeHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkConcurrentRequest() {
	bulkConcurrentRequestHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkConcurrentRequestHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkConcurrentRequestTime() {
	bulkConcurrentRequestTimeHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkConcurrentRequestTimeHelper("Group/all", s)
}

func (s *APITestSuite) TestValidateRequest() {
	validateRequestHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	validateRequestHelper("Group/all", s)
}

func (s *APITestSuite) TestBulkPatientRequestBBClientFailure() {
	bulkPatientRequestBBClientFailureHelper("Patient", s)
	s.TearDownTest()
	s.SetupTest()
	bulkPatientRequestBBClientFailureHelper("Group/all", s)
}

func bulkEOBRequestHelper(endpoint, since string, s *APITestSuite) {
	acoID := constants.DevACOUUID
	err := s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit", since)
	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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
	s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{})
}

func bulkEOBRequestInvalidSinceFormatHelper(endpoint, since string, s *APITestSuite) {
	acoID := constants.DevACOUUID
	err := s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)

	_, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit", since)
	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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

	var respOO fhirmodels.OperationOutcome
	err = json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format.", respOO.Issue[0].Details.Coding[0].Display)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func bulkEOBRequestNoBeneficiariesInACOHelper(endpoint string, s *APITestSuite) {
	acoID := "A40404F7-1EF2-485A-9B71-40FE7ACDCBC2"

	_, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit", "")
	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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
	_, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit", "")

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
	assert.Equal(s.T(), responseutils.TokenErr, respOO.Issue[0].Details.Coding[0].Code)
}

func bulkEOBRequestNoQueueHelper(endpoint string, s *APITestSuite) {
	qc = nil

	acoID := constants.SmallACOUUID

	_, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit", "")

	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusInternalServerError, s.rr.Code)

	var respOO fhirmodels.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), responseutils.Processing, respOO.Issue[0].Details.Coding[0].Code)
}

func bulkPatientRequestHelper(endpoint, since string, s *APITestSuite) {
	acoID := constants.DevACOUUID

	defer func() {
		s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{})
	}()

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "Patient", since)

	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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

func bulkPatientRequestInvalidSinceFormatHelper(endpoint, since string, s *APITestSuite) {
	acoID := constants.DevACOUUID
	err := s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)

	_, handlerFunc, req := bulkRequestHelper(endpoint, "Patient", since)
	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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

	var respOO fhirmodels.OperationOutcome
	err = json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format.", respOO.Issue[0].Details.Coding[0].Display)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func bulkCoverageRequestHelper(endpoint, since string, s *APITestSuite) {
	acoID := constants.DevACOUUID

	defer func() {
		s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{})
	}()

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "Coverage", since)

	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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

func bulkCoverageRequestInvalidSinceFormatHelper(endpoint, since string, s *APITestSuite) {
	acoID := constants.DevACOUUID
	err := s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)

	_, handlerFunc, req := bulkRequestHelper(endpoint, "Coverage", since)
	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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

	var respOO fhirmodels.OperationOutcome
	err = json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), "Invalid date format supplied in _since parameter.  Date must be in FHIR Instant format.", respOO.Issue[0].Details.Coding[0].Display)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func bulkPatientRequestBBClientFailureHelper(endpoint string, s *APITestSuite) {
	orig := os.Getenv("BB_CLIENT_CERT_FILE")
	err := os.Setenv("BB_CLIENT_CERT_FILE", "blah")
	assert.Nil(s.T(), err)

	acoID := constants.DevACOUUID
	err = s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)

	_, handlerFunc, req := bulkRequestHelper(endpoint, "Patient", "")
	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
	handlerFunc(s.rr, req)

	assert.Equal(s.T(), http.StatusInternalServerError, s.rr.Code)
	os.Setenv("BB_CLIENT_CERT_FILE", orig)
}

func bulkRequestInvalidTypeHelper(endpoint string, s *APITestSuite) {
	_, handlerFunc, req := bulkRequestHelper(endpoint, "Foo", "")
	handlerFunc(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func bulkConcurrentRequestHelper(endpoint string, s *APITestSuite) {
	err := os.Setenv("DEPLOYMENT_TARGET", "prod")
	assert.Nil(s.T(), err)
	acoID := constants.DevACOUUID
	err = s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit", "")

	j := models.Job{
		ACOID:      uuid.Parse(acoID),
		RequestURL: requestUrl,
		Status:     "In Progress",
		JobCount:   1,
	}
	s.db.Save(&j)

	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
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

	_, handlerFunc, req = bulkRequestHelper(endpoint, "Patient", "")
	ad = makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
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
	err = s.db.Unscoped().Where("aco_id = ?", acoID).Delete(models.Job{}).Error
	assert.Nil(s.T(), err)

	requestUrl, handlerFunc, req := bulkRequestHelper(endpoint, "ExplanationOfBenefit", "")

	j := models.Job{
		ACOID:      uuid.Parse(acoID),
		RequestURL: requestUrl,
		Status:     "In Progress",
		JobCount:   1,
	}
	s.db.Save(&j)

	ad := makeContextValues(acoID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))
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
	_, _, req := bulkRequestHelper(endpoint, "", "")

	resourceTypes, err := validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 3, len(resourceTypes))
	for _, t := range resourceTypes {
		if t != "ExplanationOfBenefit" && t != "Patient" && t != "Coverage" {
			assert.Fail(s.T(), "Invalid Resource type found")
		}
	}

	_, _, req = bulkRequestHelper(endpoint, "ExplanationOfBenefit,Patient", "")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 2, len(resourceTypes))
	for _, t := range resourceTypes {
		if t != "ExplanationOfBenefit" && t != "Patient" {
			assert.Fail(s.T(), "Invalid Resource type found")
		}
	}

	_, _, req = bulkRequestHelper(endpoint, "Coverage,Patient", "")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 2, len(resourceTypes))
	for _, t := range resourceTypes {
		if t != "Coverage" && t != "Patient" {
			assert.Fail(s.T(), "Invalid Resource type found")
		}
	}

	_, _, req = bulkRequestHelper(endpoint, "ExplanationOfBenefit", "")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(resourceTypes))
	assert.Contains(s.T(), resourceTypes, "ExplanationOfBenefit")

	_, _, req = bulkRequestHelper(endpoint, "Patient", "")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(resourceTypes))
	assert.Contains(s.T(), resourceTypes, "Patient")

	_, _, req = bulkRequestHelper(endpoint, "Coverage", "")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(resourceTypes))
	assert.Contains(s.T(), resourceTypes, "Coverage")

	_, _, req = bulkRequestHelper(endpoint, "Practitioner", "")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), resourceTypes)
	assert.Equal(s.T(), responseutils.Error, err.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, err.Issue[0].Code)
	assert.Equal(s.T(), responseutils.RequestErr, err.Issue[0].Details.Coding[0].Code)
	assert.Equal(s.T(), "Invalid resource type", err.Issue[0].Details.Coding[0].Display)

	_, _, req = bulkRequestHelper(endpoint, "Patient,Patient", "")
	resourceTypes, err = validateRequest(req)
	assert.Nil(s.T(), resourceTypes)
	assert.Equal(s.T(), responseutils.Error, err.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, err.Issue[0].Code)
	assert.Equal(s.T(), responseutils.RequestErr, err.Issue[0].Details.Coding[0].Code)
	assert.Equal(s.T(), "Repeated resource type", err.Issue[0].Details.Coding[0].Display)
}

func bulkRequestHelper(endpoint, resourceType, since string) (string, func(http.ResponseWriter, *http.Request), *http.Request) {
	var handlerFunc http.HandlerFunc
	var req *http.Request
	var group string

	if endpoint == "Patient" {
		handlerFunc = bulkPatientRequest
	} else if endpoint == "Group/all" {
		handlerFunc = bulkGroupRequest
		group = groupAll
	}

	requestUrl, _ := url.Parse(fmt.Sprintf("/api/v1/%s/$export", endpoint))
	q := requestUrl.Query()
	if resourceType != "" {
		q.Set("_type", resourceType)
	}
	if since != "" {
		q.Set("_since", since)
	}

	requestUrl.RawQuery = q.Encode()
	req = httptest.NewRequest("GET", requestUrl.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("groupId", group)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return requestUrl.Path, handlerFunc, req
}

func (s *APITestSuite) TestJobStatusInvalidJobID() {
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%s", "test"), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", "test")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler.ServeHTTP(s.rr, req)

	var respOO fhirmodels.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), responseutils.DbErr, respOO.Issue[0].Details.Coding[0].Code)
}

func (s *APITestSuite) TestJobStatusJobDoesNotExist() {
	jobID := "1234"
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%s", jobID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler.ServeHTTP(s.rr, req)

	var respOO fhirmodels.OperationOutcome
	err := json.Unmarshal(s.rr.Body.Bytes(), &respOO)
	if err != nil {
		s.T().Error(err)
	}

	assert.Equal(s.T(), responseutils.Error, respOO.Issue[0].Severity)
	assert.Equal(s.T(), responseutils.Exception, respOO.Issue[0].Code)
	assert.Equal(s.T(), responseutils.DbErr, respOO.Issue[0].Details.Coding[0].Code)
}

func (s *APITestSuite) TestJobStatusPending() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
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
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), "Pending", s.rr.Header().Get("X-Progress"))
	assert.Equal(s.T(), "", s.rr.Header().Get("Expires"))
	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestJobStatusInProgress() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
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
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusAccepted, s.rr.Code)
	assert.Equal(s.T(), "In Progress (0%)", s.rr.Header().Get("X-Progress"))
	assert.Equal(s.T(), "", s.rr.Header().Get("Expires"))

	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestJobStatusFailed() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Failed",
	}

	s.db.Save(&j)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusInternalServerError, s.rr.Code)

	s.db.Unscoped().Delete(&j)
}

// https://stackoverflow.com/questions/34585957/postgresql-9-3-how-to-insert-upper-case-uuid-into-table
func (s *APITestSuite) TestJobStatusCompleted() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Completed",
	}
	s.db.Save(&j)

	var expectedUrls []string

	for i := 1; i <= 10; i++ {
		fileName := fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
		expectedurl := fmt.Sprintf("%s/%s/%s", "http://example.com/data", fmt.Sprint(j.ID), fileName)
		expectedUrls = append(expectedUrls, expectedurl)
		jobKey := models.JobKey{JobID: j.ID, FileName: fileName, ResourceType: "ExplanationOfBenefit"}
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
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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
	assert.Empty(s.T(), rb.Errors)
	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestJobStatusCompletedErrorFileExists() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Completed",
	}
	s.db.Save(&j)
	fileName := fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
	jobKey := models.JobKey{
		JobID:        j.ID,
		FileName:     fileName,
		ResourceType: "ExplanationOfBenefit",
	}
	s.db.Save(&jobKey)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)
	req.TLS = &tls.ConnectionState{}

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Expired",
	}

	s.db.Save(&j)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	handler.ServeHTTP(s.rr, req)

	assert.Equal(s.T(), http.StatusGone, s.rr.Code)
	// There seems to be some slight difference in precision here.  Match on first 20 chars sb fine.
	assert.Equal(s.T(), j.UpdatedAt.Add(GetJobTimeout()).String()[:20], s.rr.Header().Get("Expires")[:20])
	s.db.Unscoped().Delete(&j)
}

func (s *APITestSuite) TestJobStatusArchived() {
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Archived",
	}

	s.db.Save(&j)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)

	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", fmt.Sprint(j.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	ad := makeContextValues("DBBD1CE1-AE24-435C-807D-ED45953077D3")
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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
	ad := makeContextValues(constants.SmallACOUUID)
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

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

func makeContextValues(acoID string) (data auth.AuthData) {
	return auth.AuthData{ACOID: acoID, TokenID: uuid.NewRandom().String()}
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
