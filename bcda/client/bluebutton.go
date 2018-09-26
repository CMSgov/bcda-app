package client

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pborman/uuid"
)

var logger *logrus.Logger

type BlueButtonClient struct {
	httpClient http.Client
}

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	filePath := os.Getenv("BCDA_BB_LOG")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		logger.SetOutput(file)
	} else {
		logger.Info("Failed to log to file; using default stderr")
	}
}

func NewBlueButtonClient() (*BlueButtonClient, error) {
	certFile := os.Getenv("BB_CLIENT_CERT_FILE")
	keyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	// TODO Fix when Blue Button has a static cert: https://jira.cms.gov/browse/BLUEBUTTON-484
	if os.Getenv("BB_SERVER_LOCATION") != "https://fhir.backend.bluebutton.hhsdevcloud.us" {
		caFile := os.Getenv("BB_CLIENT_CA_FILE")
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	} else {
		tlsConfig.InsecureSkipVerify = true
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	return &BlueButtonClient{*client}, nil
}

func (bbc *BlueButtonClient) GetPatientData(patientID string) (string, error) {
	params := url.Values{}
	params.Set("_id", patientID)
	params.Set("_format", "application/fhir+json")
	return bbc.getData("/baseDstu3/Patient/", params)
}

func (bbc *BlueButtonClient) GetCoverageData(beneficiaryID string) (string, error) {
	params := url.Values{}
	params.Set("beneficiary", beneficiaryID)
	params.Set("_format", "application/fhir+json")
	return bbc.getData("/baseDstu3/Coverage/", params)
}

func (bbc *BlueButtonClient) GetExplanationOfBenefitData(patientID string) (string, error) {
	params := url.Values{}
	params.Set("patient", patientID)
	params.Set("_format", "application/fhir+json")
	return bbc.getData("/baseDstu3/ExplanationOfBenefit/", params)
}

func (bbc *BlueButtonClient) GetMetadata() (string, error) {
	params := url.Values{}
	params.Set("_format", "application/fhir+json")
	return bbc.getData("/baseDstu3/metadata/", params)
}

func (bbc *BlueButtonClient) getData(path string, params url.Values) (string, error) {
	reqID := uuid.NewRandom()

	bbServer := os.Getenv("BB_SERVER_LOCATION")

	req, err := http.NewRequest("GET", bbServer+path, nil)
	if err != nil {
		return "", err
	}

	req.URL.RawQuery = params.Encode()

	addRequestHeaders(req, reqID)

	resp, err := bbc.httpClient.Do(req)
	logRequest(req, resp)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func addRequestHeaders(req *http.Request, reqID uuid.UUID) {
	// Info for BB backend: https://jira.cms.gov/browse/BLUEBUTTON-483
	// Populating header values: https://jira.cms.gov/browse/BCDA-334
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

func logRequest(req *http.Request, resp *http.Response) {
	logger.WithFields(logrus.Fields{
		"bb_query_id": req.Header.Get("BlueButton-OriginalQueryId"),
		"bb_query_ts": req.Header.Get("BlueButton-OriginalQueryTimestamp"),
		"bb_uri":      req.Header.Get("BlueButton-OriginalUrl"),
	}).Infoln("Blue Button request")

	if resp != nil {
		logger.WithFields(logrus.Fields{
			"resp_code":      resp.StatusCode,
			"bb_query_id":    resp.Header.Get("BlueButton-OriginalQueryId"),
			"bb_query_ts":    resp.Header.Get("BlueButton-OriginalQueryTimestamp"),
			"bb_uri":         resp.Header.Get("BlueButton-OriginalUrl"),
			"content_length": resp.ContentLength,
		}).Infoln("Blue Button response")
	}
}
