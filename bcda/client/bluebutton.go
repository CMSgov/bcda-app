package client

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/utils"

	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/pborman/uuid"
	"golang.org/x/crypto/pbkdf2"
)

var logger *logrus.Logger

const blueButtonBasePath = "/v1/fhir"

type APIClient interface {
	GetExplanationOfBenefit(patientID, jobID, cmsID string) (string, error)
	GetPatient(patientID, jobID, cmsID string) (string, error)
	GetCoverage(beneficiaryID, jobID, cmsID string) (string, error)
	GetPatientByIdentifierHash(hashedIdentifier string) (string, error)
}

type BlueButtonClient struct {
	httpClient http.Client
}

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.SetReportCaller(true)
	filePath := os.Getenv("BCDA_BB_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

	if err == nil {
		logger.SetOutput(file)
	} else {
		logger.Info("Failed to open Blue Button log file; using default stderr")
	}
}

func NewBlueButtonClient() (*BlueButtonClient, error) {
	certFile := os.Getenv("BB_CLIENT_CERT_FILE")
	keyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not load Blue Button keypair")
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	if strings.ToLower(os.Getenv("BB_CHECK_CERT")) != "false" {
		caFile := os.Getenv("BB_CLIENT_CA_FILE")
		caCert, err := ioutil.ReadFile(filepath.Clean(caFile))
		if err != nil {
			return nil, errors.Wrap(err, "could not read CA file")
		}
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, errors.New("could not append CA certificate(s)")
		}
		tlsConfig.RootCAs = caCertPool
	} else {
		tlsConfig.InsecureSkipVerify = true
		logger.Warn("Blue Button certificate check disabled")
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	var timeout int
	if timeout, err = strconv.Atoi(os.Getenv("BB_TIMEOUT_MS")); err != nil {
		logger.Info("Could not get Blue Button timeout from environment variable; using default value of 500.")
		timeout = 500
	}
	client := &http.Client{Transport: transport, Timeout: time.Duration(timeout) * time.Millisecond}

	return &BlueButtonClient{*client}, nil
}

type BeneDataFunc func(string, string, string) (string, error)

func (bbc *BlueButtonClient) GetPatient(patientID, jobID, cmsID string) (string, error) {
	params := GetDefaultParams()
	params.Set("_id", patientID)
	return bbc.getData(blueButtonBasePath+"/Patient/", params, jobID, cmsID)
}

func (bbc *BlueButtonClient) GetPatientByIdentifierHash(hashedIdentifier string) (string, error) {
	params := GetDefaultParams()

	identifier := "hicnHash"
	if utils.FromEnv("PATIENT_IDENTIFIER_MODE","HICN_MODE") == "MBI_MODE" {
		identifier = "mbiHash"
	}

	// FHIR spec requires a FULLY qualified namespace so this is in fact the argument, not a URL
	params.Set("identifier", fmt.Sprintf("http://bluebutton.cms.hhs.gov/identifier#%s|%v", identifier, hashedIdentifier))
	return bbc.getData(blueButtonBasePath+"/Patient/", params, "", "")
}

func (bbc *BlueButtonClient) GetCoverage(beneficiaryID, jobID, cmsID string) (string, error) {
	params := GetDefaultParams()
	params.Set("beneficiary", beneficiaryID)
	return bbc.getData(blueButtonBasePath+"/Coverage/", params, jobID, cmsID)
}

func (bbc *BlueButtonClient) GetExplanationOfBenefit(patientID, jobID, cmsID string) (string, error) {
	params := GetDefaultParams()
	params.Set("patient", patientID)
	params.Set("excludeSAMHSA", "true")
	return bbc.getData(blueButtonBasePath+"/ExplanationOfBenefit/", params, jobID, cmsID)
}

func (bbc *BlueButtonClient) GetMetadata() (string, error) {
	params := GetDefaultParams()
	return bbc.getData(blueButtonBasePath+"/metadata/", params, "", "")
}

func (bbc *BlueButtonClient) getData(path string, params url.Values, jobID, cmsID string) (string, error) {
	m := monitoring.GetMonitor()
	txn := m.Start(path, nil, nil)
	defer m.End(txn)

	bbServer := os.Getenv("BB_SERVER_LOCATION")

	req, err := http.NewRequest("GET", bbServer+path, nil)
	if err != nil {
		return "", err
	}

	req.URL.RawQuery = params.Encode()

	queryID := uuid.NewRandom()
	AddRequestHeaders(req, queryID, jobID, cmsID)

	tryCount := 0
	maxTries := utils.GetEnvInt("BB_REQUEST_MAX_TRIES", 3)
	retryInterval := utils.GetEnvInt("BB_REQUEST_RETRY_INTERVAL_MS", 1000)

	for tryCount < maxTries {
		tryCount++
		if tryCount > 1 {
			logger.Infof("Blue Button request %s try #%d in %d ms...", queryID, tryCount, retryInterval)
			time.Sleep(time.Duration(retryInterval) * time.Millisecond)
		}

		data, err := bbc.tryRequest(req)
		if err != nil {
			logger.Error(err)
			continue
		}
		return data, nil
	}

	return "", fmt.Errorf("Blue Button request %s failed %d time(s)", queryID, tryCount)
}

func AddRequestHeaders(req *http.Request, reqID uuid.UUID, jobID, cmsID string) {
	// Info for BB backend: https://jira.cms.gov/browse/BLUEBUTTON-483
	req.Header.Add("BlueButton-OriginalQueryTimestamp", time.Now().String())
	req.Header.Add("BlueButton-OriginalQueryId", reqID.String())
	req.Header.Add("BlueButton-OriginalQueryCounter", "1")
	req.Header.Add("keep-alive", "")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("X-Forwarded-Host", "")
	req.Header.Add("BlueButton-OriginalUrl", req.URL.String())
	req.Header.Add("BlueButton-OriginalQuery", req.URL.RawQuery)
	req.Header.Add("BCDA-JOBID", jobID)
	req.Header.Add("BCDA-CMSID", cmsID)

        // Do not set BB-specific headers with blank values
        // Leaving them here, commented out, in case we want to set them to real values in future
        //req.Header.Add("BlueButton-BeneficiaryId", "")
        //req.Header.Add("BlueButton-OriginatingIpAddress", "")
        //req.Header.Add("BlueButton-BackendCall", "")

}

func (bbc *BlueButtonClient) tryRequest(req *http.Request) (string, error) {
	go logRequest(req)
	resp, err := bbc.httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
		logResponse(req, resp)
	}
	if err != nil {
		return "", errors.Wrapf(err, "error from request %s", req.Header.Get("BlueButton-OriginalQueryId"))
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("error from request %s: %s", req.Header.Get("BlueButton-OriginalQueryId"), resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "error reading response from request %s", req.Header.Get("BlueButton-OriginalQueryId"))
	}

	return string(data), nil
}

func logRequest(req *http.Request) {
	logger.WithFields(logrus.Fields{
		"bb_query_id": req.Header.Get("BlueButton-OriginalQueryId"),
		"bb_query_ts": req.Header.Get("BlueButton-OriginalQueryTimestamp"),
		"bb_uri":      req.Header.Get("BlueButton-OriginalUrl"),
		"job_id":      req.Header.Get("BCDA-JOBID"),
		"cms_id":      req.Header.Get("BCDA-CMSID"),
	}).Infoln("request")
}

func logResponse(req *http.Request, resp *http.Response) {
	logger.WithFields(logrus.Fields{
		"resp_code":   resp.StatusCode,
		"bb_query_id": req.Header.Get("BlueButton-OriginalQueryId"),
		"bb_query_ts": req.Header.Get("BlueButton-OriginalQueryTimestamp"),
		"bb_uri":      req.Header.Get("BlueButton-OriginalUrl"),
		"job_id":      req.Header.Get("BCDA-JOBID"),
		"cms_id":      req.Header.Get("BCDA-CMSID"),
	}).Infoln("response")
}

func GetDefaultParams() (params url.Values) {
	params = url.Values{}
	params.Set("_format", "application/fhir+json")
	return params
}

func HashIdentifier(toHash string) (hashedValue string) {
	blueButtonPepper := os.Getenv("BB_HASH_PEPPER")
	blueButtonIter := utils.GetEnvInt("BB_HASH_ITER", 1000)

	pepper, err := hex.DecodeString(blueButtonPepper)
	// not sure how this can happen
	if err != nil {
		return ""
	}
	return hex.EncodeToString(pbkdf2.Key([]byte(toHash), pepper, blueButtonIter, 32, sha256.New))
}
