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

	"github.com/cenkalti/backoff/v4"

	"github.com/CMSgov/bcda-app/bcda/client/fhir"
	models "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/pborman/uuid"
	"golang.org/x/crypto/pbkdf2"
)

var logger *logrus.Logger

const (
	clientIDHeader = "BULK-CLIENTID"
	jobIDHeader    = "BULK-JOBID"
)

// BlueButtonConfig holds the configuration settings needed to create a BlueButtonClient
// TODO (BCDA-3755): Move the other env vars used in NewBlueButtonClient to this struct
type BlueButtonConfig struct {
	BBServer   string
	BBBasePath string
}

// NewConfig generates a new BlueButtonConfig using various environment variables.
func NewConfig(basePath string) BlueButtonConfig {
	return BlueButtonConfig{
		BBServer:   conf.GetEnv("BB_SERVER_LOCATION"),
		BBBasePath: basePath,
	}
}

type ClaimsDate struct {
	LowerBound time.Time
	UpperBound time.Time
}

type APIClient interface {
	GetExplanationOfBenefit(patientID, jobID, cmsID, since string, transactionTime time.Time, claimsDate ClaimsDate) (*models.Bundle, error)
	GetPatient(patientID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error)
	GetCoverage(beneficiaryID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error)
	GetPatientByIdentifierHash(hashedIdentifier string) (string, error)
}

type BlueButtonClient struct {
	client fhir.Client

	maxTries      uint64
	retryInterval time.Duration

	bbServer   string
	bbBasePath string
}

// Ensure BlueButtonClient satisfies the interface
var _ APIClient = &BlueButtonClient{}

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.SetReportCaller(true)
	filePath := conf.GetEnv("BCDA_BB_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

	if err == nil {
		logger.SetOutput(file)
	} else {
		logger.Info("Failed to open Blue Button log file; using default stderr")
	}
}

func NewBlueButtonClient(config BlueButtonConfig) (*BlueButtonClient, error) {
	certFile := conf.GetEnv("BB_CLIENT_CERT_FILE")
	keyFile := conf.GetEnv("BB_CLIENT_KEY_FILE")
	pageSize := utils.GetEnvInt("BB_CLIENT_PAGE_SIZE", 0)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not load Blue Button keypair")
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}

	if strings.ToLower(conf.GetEnv("BB_CHECK_CERT")) != "false" {
		caFile := conf.GetEnv("BB_CLIENT_CA_FILE")
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
	if timeout, err = strconv.Atoi(conf.GetEnv("BB_TIMEOUT_MS")); err != nil {
		logger.Info("Could not get Blue Button timeout from environment variable; using default value of 500.")
		timeout = 500
	}

	hl := &httpLogger{transport, logger}
	httpClient := &http.Client{Transport: hl, Timeout: time.Duration(timeout) * time.Millisecond}
	client := fhir.NewClient(httpClient, pageSize)
	maxTries := uint64(utils.GetEnvInt("BB_REQUEST_MAX_TRIES", 3))
	retryInterval := time.Duration(utils.GetEnvInt("BB_REQUEST_RETRY_INTERVAL_MS", 1000)) * time.Millisecond
	return &BlueButtonClient{client, maxTries, retryInterval, config.BBServer, config.BBBasePath}, nil
}

func (bbc *BlueButtonClient) GetPatient(patientID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error) {
	header := make(http.Header)
	header.Add("IncludeAddressFields", "true")
	params := GetDefaultParams()
	params.Set("_id", patientID)
	updateParamWithLastUpdated(&params, since, transactionTime)

	u, err := bbc.getURL("Patient", params)
	if err != nil {
		return nil, err
	}

	return bbc.getBundleData(u, jobID, cmsID, header)
}

func (bbc *BlueButtonClient) GetPatientByIdentifierHash(hashedIdentifier string) (string, error) {
	params := GetDefaultParams()

	// FHIR spec requires a FULLY qualified namespace so this is in fact the argument, not a URL
	params.Set("identifier", fmt.Sprintf("https://bluebutton.cms.gov/resources/identifier/%s|%v", "mbi-hash", hashedIdentifier))

	u, err := bbc.getURL("Patient", params)
	if err != nil {
		return "", err
	}

	return bbc.getRawData(u)
}

func (bbc *BlueButtonClient) GetCoverage(beneficiaryID, jobID, cmsID, since string, transactionTime time.Time) (*models.Bundle, error) {
	params := GetDefaultParams()
	params.Set("beneficiary", beneficiaryID)
	updateParamWithLastUpdated(&params, since, transactionTime)

	u, err := bbc.getURL("Coverage", params)
	if err != nil {
		return nil, err
	}

	return bbc.getBundleData(u, jobID, cmsID, nil)
}

func (bbc *BlueButtonClient) GetExplanationOfBenefit(patientID, jobID, cmsID, since string, transactionTime time.Time, claimsDate ClaimsDate) (*models.Bundle, error) {
	// ServiceDate only uses yyyy-mm-dd
	const svcDateFmt = "2006-01-02"

	header := make(http.Header)
	header.Add("IncludeTaxNumbers", "true")
	params := GetDefaultParams()
	params.Set("patient", patientID)
	params.Set("excludeSAMHSA", "true")

	if !claimsDate.LowerBound.IsZero() {
		params.Add("service-date", fmt.Sprintf("ge%s", claimsDate.LowerBound.Format(svcDateFmt)))
	}
	if !claimsDate.UpperBound.IsZero() {
		params.Add("service-date", fmt.Sprintf("le%s", claimsDate.UpperBound.Format(svcDateFmt)))
	}

	updateParamWithLastUpdated(&params, since, transactionTime)

	u, err := bbc.getURL("ExplanationOfBenefit", params)
	if err != nil {
		return nil, err
	}

	return bbc.getBundleData(u, jobID, cmsID, header)
}

func (bbc *BlueButtonClient) GetMetadata() (string, error) {
	u, err := bbc.getURL("metadata", GetDefaultParams())
	if err != nil {
		return "", err
	}

	return bbc.getRawData(u)
}

func (bbc *BlueButtonClient) getBundleData(u *url.URL, jobID, cmsID string, headers http.Header) (*models.Bundle, error) {
	var b *models.Bundle
	for ok := true; ok; {
		result, nextURL, err := bbc.tryBundleRequest(u, jobID, cmsID, headers)
		if err != nil {
			return nil, err
		}

		if b == nil {
			b = result
		} else {
			b.Entries = append(b.Entries, result.Entries...)
		}

		u = nextURL
		ok = nextURL != nil
	}

	return b, nil
}

func (bbc *BlueButtonClient) tryBundleRequest(u *url.URL, jobID, cmsID string, headers http.Header) (*models.Bundle, *url.URL, error) {
	m := monitoring.GetMonitor()
	txn := m.Start(u.Path, nil, nil)
	defer m.End(txn)

	var (
		result  *models.Bundle
		nextURL *url.URL
		err     error
	)

	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = bbc.retryInterval
	b := backoff.WithMaxRetries(eb, bbc.maxTries)

	err = backoff.RetryNotify(func() error {
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			logger.Error(err)
			return err
		}

		for key, values := range headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		queryID := uuid.NewRandom()
		addRequestHeaders(req, queryID, jobID, cmsID)

		result, nextURL, err = bbc.client.DoBundleRequest(req)
		if err != nil {
			logger.Error(err)
		}
		return err
	},
		b,
		func(err error, d time.Duration) {
			logger.Infof("Blue Button request failed %s. Retry in %s", err.Error(), d.String())
		},
	)

	if err != nil {
		return nil, nil, fmt.Errorf("blue button request failed %d time(s) %s", bbc.maxTries, err.Error())
	}

	return result, nextURL, nil
}

func (bbc *BlueButtonClient) getRawData(u *url.URL) (string, error) {
	m := monitoring.GetMonitor()
	txn := m.Start(u.Path, nil, nil)
	defer m.End(txn)

	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = bbc.retryInterval
	b := backoff.WithMaxRetries(eb, bbc.maxTries)

	var result string

	err := backoff.RetryNotify(func() error {
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			logger.Error(err)
			return err
		}
		addRequestHeaders(req, uuid.NewRandom(), "", "")

		result, err = bbc.client.DoRaw(req)
		if err != nil {
			logger.Error(err)
		}
		return err
	},
		b,
		func(err error, d time.Duration) {
			logger.Infof("Blue Button request failed %s. Retry in %s", err, d.String())
		},
	)

	if err != nil {
		return "", fmt.Errorf("blue button request failed %d time(s) %s", bbc.maxTries, err.Error())
	}

	return result, nil
}

func (bbc *BlueButtonClient) getURL(path string, params url.Values) (*url.URL, error) {
	u, err := url.Parse(fmt.Sprintf("%s%s/%s/", bbc.bbServer, bbc.bbBasePath, path))
	if err != nil {
		return nil, err
	}
	u.RawQuery = params.Encode()

	return u, nil
}

func addRequestHeaders(req *http.Request, reqID uuid.UUID, jobID, cmsID string) {
	// Info for BB backend: https://jira.cms.gov/browse/BLUEBUTTON-483
	req.Header.Add("BlueButton-OriginalQueryTimestamp", time.Now().String())
	req.Header.Add("BlueButton-OriginalQueryId", reqID.String())
	req.Header.Add("BlueButton-OriginalQueryCounter", "1")
	req.Header.Add("keep-alive", "")
	req.Header.Add("BlueButton-OriginalUrl", req.URL.String())
	req.Header.Add("BlueButton-OriginalQuery", req.URL.RawQuery)
	req.Header.Add(jobIDHeader, jobID)
	req.Header.Add(clientIDHeader, cmsID)
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
	blueButtonPepper := conf.GetEnv("BB_HASH_PEPPER")
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
		"job_id":      req.Header.Get(jobIDHeader),
		"cms_id":      req.Header.Get(clientIDHeader),
	}).Infoln("request")
}

func (h *httpLogger) logResponse(req *http.Request, resp *http.Response) {
	h.l.WithFields(logrus.Fields{
		"resp_code":   resp.StatusCode,
		"bb_query_id": req.Header.Get("BlueButton-OriginalQueryId"),
		"bb_query_ts": req.Header.Get("BlueButton-OriginalQueryTimestamp"),
		"bb_uri":      req.URL.String(),
		"job_id":      req.Header.Get(jobIDHeader),
		"cms_id":      req.Header.Get(clientIDHeader),
	}).Infoln("response")
}
