package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ccoveille/go-safecast"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/CMSgov/bcda-app/bcda/client/fhir"
	"github.com/CMSgov/bcda-app/bcda/constants"
	fhirModels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/pborman/uuid"
)

var logger logrus.FieldLogger

const (
	clientIDHeader      = "BULK-CLIENTID"
	jobIDHeader         = "BULK-JOBID"
	TransactionIDHeader = "TRANSACTIONID"
)

// BlueButtonConfig holds the configuration settings needed to create a BlueButtonClient
// TODO (BCDA-3755): Move the other env vars used in NewBlueButtonClient to this struct
type BlueButtonConfig struct {
	BBServer   string
	BBBasePath string
}

// NewConfig generates a new BlueButtonConfig using various environment variables.
func NewConfig(basePath string) BlueButtonConfig {
	var server string
	if basePath == constants.BFDV3Path {
		server = conf.GetEnv("V3_BB_SERVER_LOCATION")
	} else {
		server = conf.GetEnv("BB_SERVER_LOCATION")
	}
	return BlueButtonConfig{
		BBServer:   server,
		BBBasePath: basePath,
	}
}

type ClaimsWindow struct {
	LowerBound time.Time
	UpperBound time.Time
}

type APIClient interface {
	GetExplanationOfBenefit(jobData worker_types.JobEnqueueArgs, patientID string, claimsWindow ClaimsWindow) (*fhirModels.Bundle, error)
	GetPatient(jobData worker_types.JobEnqueueArgs, patientID string) (*fhirModels.Bundle, error)
	GetCoverage(jobData worker_types.JobEnqueueArgs, beneficiaryID string) (*fhirModels.Bundle, error)
	GetPatientByMbi(jobData worker_types.JobEnqueueArgs, mbi string) (string, error)
	GetClaim(jobData worker_types.JobEnqueueArgs, mbi string, claimsWindow ClaimsWindow) (*fhirModels.Bundle, error)
	GetClaimResponse(jobData worker_types.JobEnqueueArgs, mbi string, claimsWindow ClaimsWindow) (*fhirModels.Bundle, error)
}

type BlueButtonClient struct {
	client fhir.Client

	maxTries      uint64
	retryInterval time.Duration

	bbServer   string
	BBBasePath string
}

// Ensure BlueButtonClient satisfies the interface
var _ APIClient = &BlueButtonClient{}

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
		caFilePaths := strings.Split(conf.GetEnv("BB_CLIENT_CA_FILE"), ",")
		caCertPool := x509.NewCertPool()

		for _, caFile := range caFilePaths {
			caCert, err := os.ReadFile(filepath.Clean(caFile))
			if err != nil {
				return nil, errors.Wrap(err, "could not read CA file")
			}

			if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
				return nil, errors.New("could not append CA certificate(s)")
			}
		}

		tlsConfig.RootCAs = caCertPool
	} else {
		tlsConfig.InsecureSkipVerify = true
		logger.Warn("Blue Button certificate check disabled")
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		// Ensure that we have compression enabled. This allows the transport to request for gzip content
		// and handle the decompression transparently.
		// See: https://golang.org/src/net/http/transport.go?s=3396:10950#L182 for more information
		DisableCompression: false,
	}
	var timeout int
	if timeout, err = strconv.Atoi(conf.GetEnv("BB_TIMEOUT_MS")); err != nil {
		logger.Warn(errors.Wrap(err, "Could not get Blue Button timeout from environment variable; using default value of 10000."))
		timeout = 10000
	}

	hl := &httpLogger{transport, logger}
	httpClient := &http.Client{Transport: hl, Timeout: time.Duration(timeout) * time.Millisecond}
	client := fhir.NewClient(httpClient, pageSize)
	maxTries, err := safecast.ToUint64(utils.GetEnvInt("BB_REQUEST_MAX_TRIES", 3))
	if err != nil {
		logger.Warn(errors.Wrap(err, "Could not convert Blue Button max retries from environment variable"))
	}
	retryInterval := time.Duration(utils.GetEnvInt("BB_REQUEST_RETRY_INTERVAL_MS", 1000)) * time.Millisecond
	return &BlueButtonClient{client, maxTries, retryInterval, config.BBServer, config.BBBasePath}, nil
}

// SetLogger sets the logger to be used in the client.
// Since both the API and worker use the bluebutton client, we need
// to be able to set the appropriate logger based on who is using the client.
func SetLogger(log logrus.FieldLogger) {
	logger = log
}

func (bbc *BlueButtonClient) GetPatient(jobData worker_types.JobEnqueueArgs, patientID string) (*fhirModels.Bundle, error) {
	header := make(http.Header)
	header.Add("IncludeAddressFields", "true")
	params := GetDefaultParams()
	params.Set("_id", patientID)
	updateParamWithLastUpdated(&params, jobData.Since, jobData.TransactionTime)

	u, err := bbc.getURL("Patient", params)
	if err != nil {
		return nil, err
	}

	return bbc.makeBundleDataRequest("GET", u, jobData, header, nil)
}

func (bbc *BlueButtonClient) GetPatientByMbi(jobData worker_types.JobEnqueueArgs, mbi string) (string, error) {
	headers := createURLEncodedHeader()
	params := GetDefaultParams()
	params.Set("identifier", fmt.Sprintf("http://hl7.org/fhir/sid/us-mbi|%s", mbi))

	u, err := bbc.getURL("Patient/_search", url.Values{})
	if err != nil {
		return "", err
	}

	return bbc.getRawData("POST", jobData, u, headers, strings.NewReader(params.Encode()))
}

func (bbc *BlueButtonClient) GetCoverage(jobData worker_types.JobEnqueueArgs, beneficiaryID string) (*fhirModels.Bundle, error) {
	params := GetDefaultParams()
	params.Set("beneficiary", beneficiaryID)
	updateParamWithLastUpdated(&params, jobData.Since, jobData.TransactionTime)

	u, err := bbc.getURL("Coverage", params)
	if err != nil {
		return nil, err
	}

	return bbc.makeBundleDataRequest("GET", u, jobData, nil, nil)
}

func (bbc *BlueButtonClient) GetClaim(jobData worker_types.JobEnqueueArgs, mbi string, claimsWindow ClaimsWindow) (*fhirModels.Bundle, error) {
	headers := createURLEncodedHeader()
	params := GetDefaultParams()
	updateParamsWithClaimsDefaults(&params, mbi)
	updateParamWithServiceDate(&params, claimsWindow)
	updateParamWithLastUpdated(&params, jobData.Since, jobData.TransactionTime)

	u, err := bbc.getURL("Claim/_search", url.Values{})
	if err != nil {
		return nil, err
	}

	return bbc.makeBundleDataRequest("POST", u, jobData, headers, strings.NewReader(params.Encode()))
}

func (bbc *BlueButtonClient) GetClaimResponse(jobData worker_types.JobEnqueueArgs, mbi string, claimsWindow ClaimsWindow) (*fhirModels.Bundle, error) {
	headers := createURLEncodedHeader()
	params := GetDefaultParams()
	updateParamsWithClaimsDefaults(&params, mbi)
	updateParamWithServiceDate(&params, claimsWindow)
	updateParamWithLastUpdated(&params, jobData.Since, jobData.TransactionTime)

	u, err := bbc.getURL("ClaimResponse/_search", url.Values{})
	if err != nil {
		return nil, err
	}

	return bbc.makeBundleDataRequest("POST", u, jobData, headers, strings.NewReader(params.Encode()))
}

func (bbc *BlueButtonClient) GetExplanationOfBenefit(jobData worker_types.JobEnqueueArgs, patientID string, claimsWindow ClaimsWindow) (*fhirModels.Bundle, error) {
	header := make(http.Header)
	header.Add("IncludeTaxNumbers", "true")
	params := GetDefaultParams()
	params.Set("patient", patientID)

	if bbc.BBBasePath != constants.BFDV3Path { // TODO: V3
		params.Set("excludeSAMHSA", "true")
	}

	updateParamWithServiceDate(&params, claimsWindow)
	updateParamWithLastUpdated(&params, jobData.Since, jobData.TransactionTime)
	updateParamWithTypeFilter(&params, jobData.TypeFilter)

	u, err := bbc.getURL("ExplanationOfBenefit", params)
	if err != nil {
		return nil, err
	}

	return bbc.makeBundleDataRequest("GET", u, jobData, header, nil)
}

func (bbc *BlueButtonClient) GetMetadata() (string, error) {
	u, err := bbc.getURL("metadata", GetDefaultParams())
	if err != nil {
		return "", err
	}
	jobData := worker_types.JobEnqueueArgs{}

	return bbc.getRawData("GET", jobData, u, nil, nil)
}

func (bbc *BlueButtonClient) makeBundleDataRequest(method string, u *url.URL, jobData worker_types.JobEnqueueArgs, headers http.Header, body io.Reader) (*fhirModels.Bundle, error) {
	var b *fhirModels.Bundle
	for ok := true; ok; {
		result, nextURL, err := bbc.tryBundleRequest(method, u, jobData, headers, body)
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

func (bbc *BlueButtonClient) tryBundleRequest(method string, u *url.URL, jobData worker_types.JobEnqueueArgs, headers http.Header, body io.Reader) (*fhirModels.Bundle, *url.URL, error) {
	m := monitoring.GetMonitor()
	txn := m.Start(u.Path, nil, nil)
	defer m.End(txn)

	var (
		result  *fhirModels.Bundle
		nextURL *url.URL
		err     error
	)

	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = bbc.retryInterval
	b := backoff.WithMaxRetries(eb, bbc.maxTries)

	err = backoff.RetryNotify(func() error {
		req, err := http.NewRequest(method, u.String(), body)
		if err != nil {
			logger.Error(err)
			return err
		}
		req = newrelic.RequestWithTransactionContext(req, txn)
		for key, values := range headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		addDefaultRequestHeaders(req, uuid.NewRandom(), jobData)

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

func (bbc *BlueButtonClient) getRawData(method string, jobData worker_types.JobEnqueueArgs, u *url.URL, headers http.Header, body io.Reader) (string, error) {
	m := monitoring.GetMonitor()
	txn := m.Start(u.Path, nil, nil)
	defer m.End(txn)

	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = bbc.retryInterval
	b := backoff.WithMaxRetries(eb, bbc.maxTries)

	var result string

	err := backoff.RetryNotify(func() error {
		req, err := http.NewRequest(method, u.String(), body)
		if err != nil {
			logger.Error(err)
			return err
		}

		req = newrelic.RequestWithTransactionContext(req, txn)
		for key, values := range headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		addDefaultRequestHeaders(req, uuid.NewRandom(), jobData)

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
	u, err := url.Parse(fmt.Sprintf("%s%s/%s/", bbc.bbServer, bbc.BBBasePath, path))
	if err != nil {
		return nil, err
	}
	u.RawQuery = params.Encode()

	return u, nil
}

func addDefaultRequestHeaders(req *http.Request, reqID uuid.UUID, jobData worker_types.JobEnqueueArgs) {
	// Info for BB backend: https://jira.cms.gov/browse/BLUEBUTTON-483
	req.Header.Add("keep-alive", "")
	req.Header.Add(constants.BBHeaderTS, time.Now().String())
	req.Header.Add(constants.BBHeaderOriginQID, reqID.String())
	req.Header.Add(constants.BBHeaderOriginQC, "1")
	req.Header.Add(constants.BBHeaderOriginURL, req.URL.String())
	req.Header.Add("IncludeIdentifiers", "mbi")
	req.Header.Add(jobIDHeader, strconv.Itoa(jobData.ID))
	req.Header.Add(clientIDHeader, jobData.CMSID)
	req.Header.Add(TransactionIDHeader, jobData.TransactionID)

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

func updateParamWithServiceDate(params *url.Values, claimsWindow ClaimsWindow) {
	// ServiceDate only uses yyyy-mm-dd
	const isoDate = "2006-01-02"

	if !claimsWindow.LowerBound.IsZero() {
		params.Add("service-date", fmt.Sprintf("ge%s", claimsWindow.LowerBound.Format(isoDate)))
	}

	if !claimsWindow.UpperBound.IsZero() {
		params.Add("service-date", fmt.Sprintf("le%s", claimsWindow.UpperBound.Format(isoDate)))
	}
}

func updateParamWithLastUpdated(params *url.Values, since string, transactionTime time.Time) {
	// upper bound will always be set
	params.Set("_lastUpdated", "le"+transactionTime.Format(time.RFC3339Nano))

	// only set the lower bound parameter if it exists and begins with "gt" (to align with what is expected in _lastUpdated)
	if len(since) > 0 && strings.HasPrefix(since, "gt") {
		params.Add("_lastUpdated", since)
	}
}

func updateParamWithTypeFilter(params *url.Values, typeFilter [][]string) {
	for _, paramPair := range typeFilter {
		params.Add(paramPair[0], paramPair[1])
	}
}

func updateParamsWithClaimsDefaults(params *url.Values, mbi string) {
	params.Set("excludeSAMHSA", "true")
	params.Set("includeTaxNumbers", "true")
	params.Set("isHashed", "false")
	params.Set("mbi", mbi)
}

func createURLEncodedHeader() http.Header {
	headers := make(http.Header)
	headers.Add("Content-Type", "application/x-www-form-urlencoded")

	return headers
}

type httpLogger struct {
	t *http.Transport
	l logrus.FieldLogger
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
		"bb_query_id": req.Header.Get(constants.BBHeaderOriginQID),
		"bb_query_ts": req.Header.Get(constants.BBHeaderTS),
		"bb_uri":      req.URL.String(),
		"job_id":      req.Header.Get(jobIDHeader),
		"cms_id":      req.Header.Get(clientIDHeader),
	}).Infoln("request")
}

func (h *httpLogger) logResponse(req *http.Request, resp *http.Response) {
	h.l.WithFields(logrus.Fields{
		"resp_status":    resp.StatusCode,
		"bb_query_id":    req.Header.Get(constants.BBHeaderOriginQID),
		"bb_query_ts":    req.Header.Get(constants.BBHeaderTS),
		"bb_uri":         req.URL.String(),
		"job_id":         req.Header.Get(jobIDHeader),
		"cms_id":         req.Header.Get(clientIDHeader),
		"transaction_id": req.Header.Get(TransactionIDHeader),
	}).Infoln("response")
}
