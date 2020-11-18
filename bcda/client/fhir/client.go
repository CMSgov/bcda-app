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
)

type Client interface {
	DoBundleRequest(req *http.Request) (bundle *models.Bundle, nextURL *url.URL, err error)

	// DoRaw makes a request and return the raw response from the service
	DoRaw(req *http.Request) (string, error)
}

type BundleEntry map[string]interface{}

func NewClient(httpClient *http.Client, pageSize int) Client {
	if pageSize == 0 {
		return &singleClient{httpClient}
	} else {
		return &client{httpClient, strconv.Itoa(pageSize)}
	}
}

// singleClient ensures that entire bundle response is read in a single response (i.e. no paging)
type singleClient struct {
	httpClient *http.Client
}

// Ensure singleClient satisfies the interface
var _ Client = &singleClient{}

func (c *singleClient) DoBundleRequest(req *http.Request) (bundle *models.Bundle, nextURL *url.URL, err error) {

	// Ensure that we'll receive the entire bundle response in a single request
	vals := req.URL.Query()
	vals.Del("_count")
	req.URL.RawQuery = vals.Encode()

	b, err := getBundleResponse(c.httpClient, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get bundle response: %w", err)
	}
	return b, nil, nil
}

func (c *singleClient) DoRaw(req *http.Request) (string, error) {
	resp, err := getResponse(c.httpClient, req)
	if err != nil {
		return "", fmt.Errorf("failed to get response: %w", err)
	}
	return string(resp), nil
}

// client uses paging (controlled by pageSize) to generate the entire bundle response.
type client struct {
	httpClient *http.Client
	pageSize   string
}

// Ensure client satisfies the interface
var _ Client = &client{}

func (c *client) DoBundleRequest(req *http.Request) (bundle *models.Bundle, nextURL *url.URL, err error) {
	const (
		nextRelation = "next" // Relation that contains the next URL that we should be requesting
	)
	// Set page size to our configured value
	vals := req.URL.Query()
	vals.Set("_count", c.pageSize)
	req.URL.RawQuery = vals.Encode()

	b, err := getBundleResponse(c.httpClient, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get bundle response: %w", err)
	}

	var next string
	for _, link := range b.Links {
		if link.Relation == nextRelation {
			next = link.URL
			break
		}
	}

	// We've reached the last page
	if next == "" {
		return b, nil, nil
	}

	url, err := url.Parse(next)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL %s: %w", nextURL, err)
	}
	return b, url, nil
}

func (c *client) DoRaw(req *http.Request) (string, error) {
	resp, err := getResponse(c.httpClient, req)
	if err != nil {
		return "", fmt.Errorf("failed to get response: %w", err)
	}
	return string(resp), nil
}

func getBundleResponse(c *http.Client, req *http.Request) (*models.Bundle, error) {
	body, err := getResponse(c, req)
	if err != nil {
		return nil, err
	}

	var b models.Bundle
	if err := json.Unmarshal(body, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

func getResponse(c *http.Client, req *http.Request) (body []byte, err error) {
	resp, err := c.Do(req)
	if resp != nil {
		/* #nosec -- it's OK for us to ignore errors when attempt to cleanup response body */
		defer func() {
			_, _ = io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}()
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
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
