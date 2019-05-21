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
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/pborman/uuid"
	"golang.org/x/crypto/pbkdf2"
)

var logger *logrus.Logger

const blueButtonBasePath = "/v1/fhir"
const blueButtonPepper = "b8ebdcc47fdd852b8b0201835c6273a9177806e84f2d9dc4f7ecaff08681e86d74195c6aef2db06d3d44c9d0b8f93c3e6c43d90724b605ac12585b9ab5ee9c3f00d5c0d284e6b8e49d502415c601c28930637b58fdca72476e31c22ad0f24ecd761020d6a4bcd471f0db421d21983c0def1b66a49a230f85f93097e9a9a8e0a4f4f0add775213cbf9ecfc1a6024cb021bd1ed5f4981a4498f294cca51d3939dfd9e6a1045350ddde7b6d791b4d3b884ee890d4c401ef97b46d1e57d40efe5737248dd0c4cec29c23c787231c4346cab9bb973f140a32abaa0a2bd5c0b91162f8d2a7c9d3347aafc76adbbd90ec5bfe617a3584e94bc31047e3bb6850477219a9"

type APIClient interface {
	GetExplanationOfBenefitData(patientID, jobID string) (string, error)
	GetPatientData(patientID, jobID string) (string, error)
	GetCoverageData(beneficiaryID, jobID string) (string, error)
	GetBlueButtonIdentifier(hashedHICN string) (string, error)
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

func (bbc *BlueButtonClient) GetBlueButtonIdentifier(hashedHICN string) (string, error) {
	params := url.Values{}
	// FHIR spec requires a FULLY qualified namespace so this is in fact the argument, not a URL
	params.Set("identifier", fmt.Sprintf("http://bluebutton.cms.hhs.gov/identifier#hicnHash|%v", hashedHICN))
	// This is intentionally not set by the default values because it needs to be second due to a bug at Blue Button (20190520)
	params.Set("_format", "application/fhir+json")
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

	req.URL.RawQuery = encode(params)

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

func HashHICN(toHash string) (hashedValue string) {
	pepper, err := hex.DecodeString(blueButtonPepper)
	// not sure how this can happen
	if err != nil {
		return ""
	}
	return hex.EncodeToString(pbkdf2.Key([]byte(toHash), pepper, 1000, 32, sha256.New))
}

// Copied from go URL Package, but removes sorting because BlueButton is particular about the order of params
func encode(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}

	for _, k := range keys {
		vs := v[k]
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}
