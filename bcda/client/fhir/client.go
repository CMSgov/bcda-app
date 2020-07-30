package fhir

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	models "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/sirupsen/logrus"
)

type Client interface {
	DoBundleRequest(req *http.Request) (bundle *models.Bundle, nextReq *http.Request, err error)

	// DoRaw makes a request and return the raw response from the service
	DoRaw(req *http.Request) (string, error)
}

type BundleEntry map[string]interface{}

func NewClient(httpClient *http.Client, logger *logrus.Logger, pageSize int) Client {
	if pageSize == 0 {
		return &singleClient{httpClient, logger}
	} else {
		return &client{httpClient, logger, strconv.Itoa(pageSize)}
	}
}

// singleClient ensures that entire bundle response is read in a single response (i.e. no paging)
type singleClient struct {
	httpClient *http.Client
	logger     *logrus.Logger
}

// Ensure singleClient satisfies the interface
var _ Client = &singleClient{}

func (c *singleClient) DoBundleRequest(req *http.Request) (bundle *models.Bundle, nextReq *http.Request, err error) {

	// Ensure that we'll receive the entire bundle response in a single request
	vals := req.URL.Query()
	vals.Del("_count")
	req.URL.RawQuery = vals.Encode()

	b, err := getBundleResponse(c.httpClient, c.logger, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get bundle response: %w", err)
	}
	return b, nil, nil
}

func (c *singleClient) DoRaw(req *http.Request) (string, error) {
	resp, err := getResponse(c.httpClient, c.logger, req)
	if err != nil {
		return "", fmt.Errorf("failed to get response: %w", err)
	}
	return string(resp), nil
}

// client uses paging (controlled by pageSize) to generate the entire bundle response.
type client struct {
	httpClient *http.Client
	logger     *logrus.Logger
	pageSize   string
}

// Ensure client satisfies the interface
var _ Client = &client{}

func (c *client) DoBundleRequest(req *http.Request) (bundle *models.Bundle, nextReq *http.Request, err error) {
	const (
		nextRelation = "next" // Relation that contains the next URL that we should be requesting
	)
	// Set page size to our configured value
	vals := req.URL.Query()
	vals.Set("_count", c.pageSize)
	req.URL.RawQuery = vals.Encode()

	b, err := getBundleResponse(c.httpClient, c.logger, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get bundle response: %w", err)
	}

	var nextURL string
	for _, link := range b.Links {
		if link.Relation == nextRelation {
			nextURL = link.URL
			break
		}
	}

	// We've reached the last page
	if nextURL == "" {
		return b, nil, nil
	}

	url, err := url.Parse(nextURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL %s: %w", nextURL, err)
	}

	newReq := req.Clone(req.Context())
	newReq.URL = url

	return b, newReq, nil
}

func (c *client) DoRaw(req *http.Request) (string, error) {
	resp, err := getResponse(c.httpClient, c.logger, req)
	if err != nil {
		return "", fmt.Errorf("failed to get response: %w", err)
	}
	return string(resp), nil
}

func getBundleResponse(c *http.Client, logger *logrus.Logger, req *http.Request) (*models.Bundle, error) {
	body, err := getResponse(c, logger, req)
	if err != nil {
		return nil, err
	}

	var b models.Bundle
	if err := json.Unmarshal(body, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

func getResponse(c *http.Client, logger *logrus.Logger, req *http.Request) ([]byte, error) {
	go logRequest(logger, req)
	resp, err := c.Do(req)
	if resp != nil {
		defer func() {
			_, _ = io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}()
		logResponse(logger, req, resp)
	}
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		// Attempt to read the body in case it offers valuable troubleshooting info
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("received incorrect status code %d body %s",
			resp.StatusCode, string(body))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func logRequest(logger *logrus.Logger, req *http.Request) {
	logger.WithFields(logrus.Fields{
		"bb_query_id": req.Header.Get("BlueButton-OriginalQueryId"),
		"bb_query_ts": req.Header.Get("BlueButton-OriginalQueryTimestamp"),
		"bb_uri":      req.Header.Get("BlueButton-OriginalUrl"),
		"job_id":      req.Header.Get("BCDA-JOBID"),
		"cms_id":      req.Header.Get("BCDA-CMSID"),
	}).Infoln("request")
}

func logResponse(logger *logrus.Logger, req *http.Request, resp *http.Response) {
	logger.WithFields(logrus.Fields{
		"resp_code":   resp.StatusCode,
		"bb_query_id": req.Header.Get("BlueButton-OriginalQueryId"),
		"bb_query_ts": req.Header.Get("BlueButton-OriginalQueryTimestamp"),
		"bb_uri":      req.Header.Get("BlueButton-OriginalUrl"),
		"job_id":      req.Header.Get("BCDA-JOBID"),
		"cms_id":      req.Header.Get("BCDA-CMSID"),
	}).Infoln("response")
}
