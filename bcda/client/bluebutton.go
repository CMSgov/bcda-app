package client

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/CMSgov/bcda-app/bcda/monitoring"

	"github.com/sirupsen/logrus"

	"github.com/pborman/uuid"
)

var logger *logrus.Logger

const blueButtonBasePath = "/v1/fhir"

type APIClient interface {
	GetExplanationOfBenefitData(patientID, jobID string) (string, error)
	GetPatientData(patientID, jobID string) (string, error)
	GetCoverageData(beneficiaryID, jobID string) (string, error)
}

type BlueButtonClient struct {
	httpClient http.Client
}

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
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
		/* #nosec */
		caCert, err := ioutil.ReadFile(caFile)
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

type BeneDataFunc func(string, string) (string, error)

func (bbc *BlueButtonClient) GetPatientData(patientID, jobID string) (string, error) {
	params := GetDefaultParams()
	params.Set("_id", patientID)
	return bbc.getData(blueButtonBasePath+"/Patient/", params, "")
}

func (bbc *BlueButtonClient) GetCoverageData(beneficiaryID, jobID string) (string, error) {
	params := GetDefaultParams()
	params.Set("beneficiary", beneficiaryID)
	return bbc.getData(blueButtonBasePath+"/Coverage/", params, "")
}

func (bbc *BlueButtonClient) GetExplanationOfBenefitData(patientID string, jobID string) (string, error) {
	params := GetDefaultParams()
	params.Set("patient", patientID)
	params.Set("excludeSAMHSA", "true")
	return bbc.getData(blueButtonBasePath+"/ExplanationOfBenefit/", params, jobID)
}

func (bbc *BlueButtonClient) GetMetadata() (string, error) {
	params := GetDefaultParams()
	return bbc.getData(blueButtonBasePath+"/metadata/", params, "")
}

func (bbc *BlueButtonClient) getData(path string, params url.Values, jobID string) (string, error) {
	m := monitoring.GetMonitor()
	txn := m.Start(path, nil, nil)
	defer m.End(txn)

	reqID := uuid.NewRandom()

	bbServer := os.Getenv("BB_SERVER_LOCATION")

	req, err := http.NewRequest("GET", bbServer+path, nil)
	if err != nil {
		return "", err
	}

	req.URL.RawQuery = params.Encode()

	addRequestHeaders(req, reqID)

	resp, err := bbc.httpClient.Do(req)
	logRequest(req, resp, jobID)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", errors.New(resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func addRequestHeaders(req *http.Request, reqID uuid.UUID) {
	// Info for BB backend: https://jira.cms.gov/browse/BLUEBUTTON-483
	req.Header.Add("BlueButton-OriginalQueryTimestamp", time.Now().String())
	req.Header.Add("BlueButton-OriginalQueryId", reqID.String())
	req.Header.Add("BlueButton-OriginalQueryCounter", "1")
	req.Header.Add("BlueButton-BeneficiaryId", "")
	req.Header.Add("BlueButton-OriginatingIpAddress", "")

	req.Header.Add("keep-alive", "")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("X-Forwarded-Host", "")

	req.Header.Add("BlueButton-OriginalUrl", req.URL.String())
	req.Header.Add("BlueButton-OriginalQuery", req.URL.RawQuery)
	req.Header.Add("BlueButton-BackendCall", "")
}

func logRequest(req *http.Request, resp *http.Response, jobID string) {
	logger.WithFields(logrus.Fields{
		"bb_query_id": req.Header.Get("BlueButton-OriginalQueryId"),
		"bb_query_ts": req.Header.Get("BlueButton-OriginalQueryTimestamp"),
		"bb_uri":      req.Header.Get("BlueButton-OriginalUrl"),
		"job_id":      jobID,
	}).Infoln("Blue Button request")

	if resp != nil {
		logger.WithFields(logrus.Fields{
			"resp_code":      resp.StatusCode,
			"bb_query_id":    resp.Header.Get("BlueButton-OriginalQueryId"),
			"bb_query_ts":    resp.Header.Get("BlueButton-OriginalQueryTimestamp"),
			"bb_uri":         resp.Header.Get("BlueButton-OriginalUrl"),
			"job_id":         jobID,
			"content_length": resp.ContentLength,
		}).Infoln("Blue Button response")
	}
}

func GetDefaultParams() (params url.Values) {
	params = url.Values{}
	params.Set("_format", "application/fhir+json")
	return params
}
