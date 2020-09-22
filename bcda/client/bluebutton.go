package client

import (
	"context"
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

	"github.com/cenkalti/backoff"

	"github.com/CMSgov/bcda-app/bcda/client/fhir"
	models "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/utils"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/pborman/uuid"
	"golang.org/x/crypto/pbkdf2"
)

var logger *logrus.Logger

const blueButtonBasePath = "/v1/fhir"

// BlueButtonConfig holds the configuration settings needed to create a BlueButtonClient
// TODO (BCDA-3755): Move the other env vars used in NewBlueButtonClient to this struct
type BlueButtonConfig struct {
	BBServer string
}

// NewConfig generates a new BlueButtonConfig using various environment variables.
func NewConfig() BlueButtonConfig {
	return BlueButtonConfig{
		BBServer: os.Getenv("BB_SERVER_LOCATION"),
	}
}

type APIClient interface {
	GetExplanationOfBenefit(patientID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error)
	GetPatient(patientID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error)
	GetCoverage(beneficiaryID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error)
	GetPatientByIdentifierHash(hashedIdentifier string) (string, error)
}

type BlueButtonClient struct {
	client fhir.Client

	maxTries      uint64
	retryInterval time.Duration

	bbServer string
}

// Ensure BlueButtonClient satisfies the interface
var _ APIClient = &BlueButtonClient{}

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

func NewBlueButtonClient(config BlueButtonConfig) (*BlueButtonClient, error) {
	certFile := os.Getenv("BB_CLIENT_CERT_FILE")
	keyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	pageSize := utils.GetEnvInt("BB_CLIENT_PAGE_SIZE", 0)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not load Blue Button keypair")
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}

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
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		// Ensure that we have compression enabled. This allows the transport to request for gzip content
		// and handle the decompression transparently.
		// See: https://golang.org/src/net/http/transport.go?s=3396:10950#L182 for more information
		DisableCompression: false,
	}
	var timeout int
	if timeout, err = strconv.Atoi(os.Getenv("BB_TIMEOUT_MS")); err != nil {
		logger.Info("Could not get Blue Button timeout from environment variable; using default value of 500.")
		timeout = 500
	}

	hl := &httpLogger{transport, logger}
	httpClient := &http.Client{Transport: hl, Timeout: time.Duration(timeout) * time.Millisecond}
	client := fhir.NewClient(httpClient, pageSize)
	maxTries := uint64(utils.GetEnvInt("BB_REQUEST_MAX_TRIES", 3))
	retryInterval := time.Duration(utils.GetEnvInt("BB_REQUEST_RETRY_INTERVAL_MS", 1000)) * time.Millisecond
	return &BlueButtonClient{client, maxTries, retryInterval, config.BBServer}, nil
}

type BeneDataFunc func(string, string, string, string, time.Time) (*models.Bundle, error)

func (bbc *BlueButtonClient) GetPatient(patientID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error) {
	params := GetDefaultParams()
	params.Set("_id", patientID)
	updateParamWithLastUpdated(&params, since, transactionTime)
	return bbc.getBundleData(blueButtonBasePath+"/Patient/", params, jobID, cmsID)
}

func (bbc *BlueButtonClient) GetPatientByIdentifierHash(hashedIdentifier string) (string, error) {
	params := GetDefaultParams()

	// FHIR spec requires a FULLY qualified namespace so this is in fact the argument, not a URL
	params.Set("identifier", fmt.Sprintf("https://bluebutton.cms.gov/resources/identifier/%s|%v", "mbi-hash", hashedIdentifier))
	return bbc.getRawData(blueButtonBasePath+"/Patient/", params, "", "")
}

func (bbc *BlueButtonClient) GetCoverage(beneficiaryID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error) {
	params := GetDefaultParams()
	params.Set("beneficiary", beneficiaryID)
	updateParamWithLastUpdated(&params, since, transactionTime)
	return bbc.getBundleData(blueButtonBasePath+"/Coverage/", params, jobID, cmsID)
}

func (bbc *BlueButtonClient) GetExplanationOfBenefit(patientID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error) {
	params := GetDefaultParams()
	params.Set("patient", patientID)
	params.Set("excludeSAMHSA", "true")
	updateParamWithLastUpdated(&params, since, transactionTime)
	return bbc.getBundleData(blueButtonBasePath+"/ExplanationOfBenefit/", params, jobID, cmsID)
}

func (bbc *BlueButtonClient) GetMetadata() (string, error) {
	return bbc.getRawData(blueButtonBasePath+"/metadata/", GetDefaultParams(), "", "")
}

func (bbc *BlueButtonClient) getBundleData(path string, params url.Values, jobID, cmsID string) (*models.Bundle, error) {
	req, err := bbc.getRequest(path, params)
	if err != nil {
		return nil, err
	}

	var b *models.Bundle
	for ok := true; ok; {
		result, nextReq, err := bbc.tryBundleRequest(req, jobID, cmsID)
		if err != nil {
			return nil, err
		}

		if b == nil {
			b = result
		} else {
			b.Entries = append(b.Entries, result.Entries...)
		}

		req = nextReq
		ok = nextReq != nil
	}

	return b, nil
}

func (bbc *BlueButtonClient) tryBundleRequest(req *http.Request, jobID, cmsID string) (*models.Bundle, *http.Request, error) {
	m := monitoring.GetMonitor()
	txn := m.Start(req.URL.Path, nil, nil)
	defer m.End(txn)

	queryID := uuid.NewRandom()
	addRequestHeaders(req, queryID, jobID, cmsID)

	var (
		result  *models.Bundle
		nextReq *http.Request
		err     error
	)

	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = bbc.retryInterval
	b := backoff.WithMaxRetries(eb, bbc.maxTries)

	err = backoff.RetryNotify(func() error {
		result, nextReq, err = bbc.client.DoBundleRequest(req)
		if err != nil {
			logger.Error(err)
		}
		return err
	},
		b,
		func(err error, d time.Duration) {
			logger.Infof("Blue Button request %s retry in %s ms...", queryID, d.String())
		},
	)

	if err != nil {
		return nil, nil, fmt.Errorf("Blue Button request %s failed %d time(s)", queryID, bbc.maxTries)
	}

	return result, nextReq, nil
}

func (bbc *BlueButtonClient) getRawData(path string, params url.Values, jobID, cmsID string) (string, error) {
	m := monitoring.GetMonitor()
	txn := m.Start(path, nil, nil)
	defer m.End(txn)

	req, err := bbc.getRequest(path, params)
	if err != nil {
		return "", err
	}

	queryID := uuid.NewRandom()
	addRequestHeaders(req, queryID, jobID, cmsID)

	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = bbc.retryInterval
	b := backoff.WithMaxRetries(eb, bbc.maxTries)

	var result string

	err = backoff.RetryNotify(func() error {
		result, err = bbc.client.DoRaw(req)
		if err != nil {
			logger.Error(err)
		}
		return err
	},
		b,
		func(err error, d time.Duration) {
			logger.Infof("Blue Button request %s retry in %s", queryID, d.String())
		},
	)

	if err != nil {
		return "", fmt.Errorf("Blue Button request %s failed %d time(s)", queryID, bbc.maxTries)
	}

	return result, nil
}

func (bbc *BlueButtonClient) getRequest(path string, params url.Values) (*http.Request, error) {
	req, err := http.NewRequest("GET", bbc.bbServer+path, nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = params.Encode()

	return req, nil
}

func addRequestHeaders(req *http.Request, reqID uuid.UUID, jobID, cmsID string) {
	// Info for BB backend: https://jira.cms.gov/browse/BLUEBUTTON-483
	req.Header.Add("BlueButton-OriginalQueryTimestamp", time.Now().String())
	req.Header.Add("BlueButton-OriginalQueryId", reqID.String())
	req.Header.Add("BlueButton-OriginalQueryCounter", "1")
	req.Header.Add("keep-alive", "")
	req.Header.Add("BlueButton-OriginalUrl", req.URL.String())
	req.Header.Add("BlueButton-OriginalQuery", req.URL.RawQuery)
	req.Header.Add("BCDA-JOBID", jobID)
	req.Header.Add("BCDA-CMSID", cmsID)
	req.Header.Add("IncludeIdentifiers", "mbi")

	// We SHOULD NOT be specifying "Accept-Encoding: gzip" on the request header.
	// If we specify this header at the client level, then we must be responsible for decompressing the response.
	// This header should be automatically set by the underlying http.Transport which will handle the decompression transparently
	// Details: https://golang.org/src/net/http/transport.go#L2432
	// req.Header.Add("Accept-Encoding", "gzip")

	// Do not set BB-specific headers with blank values
	// Leaving them here, commented out, in case we want to set them to real values in future
	//req.Header.Add("BlueButton-BeneficiaryId", "")
	//req.Header.Add("BlueButton-OriginatingIpAddress", "")
	//req.Header.Add("BlueButton-BackendCall", "")

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

func updateParamWithLastUpdated(params *url.Values, since string, transactionTime time.Time) {
	// upper bound will always be set
	params.Set("_lastUpdated", "le"+transactionTime.Format(time.RFC3339Nano))

	// only set the lower bound parameter if it exists and begins with "gt" (to align with what is expected in _lastUpdated)
	if len(since) > 0 && strings.HasPrefix(since, "gt") {
		params.Add("_lastUpdated", since)
	}
}

type httpLogger struct {
	t *http.Transport
	l *logrus.Logger
}

func (h *httpLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	go h.logRequest(req.Clone(context.Background()))
	resp, err := h.t.RoundTrip(req)
	if resp != nil {
		h.logResponse(req, resp)
	}
	return resp, err
}

func (h *httpLogger) logRequest(req *http.Request) {
	h.l.WithFields(logrus.Fields{
		"bb_query_id": req.Header.Get("BlueButton-OriginalQueryId"),
		"bb_query_ts": req.Header.Get("BlueButton-OriginalQueryTimestamp"),
		"bb_uri":      req.URL.String(),
		"job_id":      req.Header.Get("BCDA-JOBID"),
		"cms_id":      req.Header.Get("BCDA-CMSID"),
	}).Infoln("request")
}

func (h *httpLogger) logResponse(req *http.Request, resp *http.Response) {
	h.l.WithFields(logrus.Fields{
		"resp_code":   resp.StatusCode,
		"bb_query_id": req.Header.Get("BlueButton-OriginalQueryId"),
		"bb_query_ts": req.Header.Get("BlueButton-OriginalQueryTimestamp"),
		"bb_uri":      req.URL.String(),
		"job_id":      req.Header.Get("BCDA-JOBID"),
		"cms_id":      req.Header.Get("BCDA-CMSID"),
	}).Infoln("response")
}
