package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/sirupsen/logrus"
)

var (
	accessToken, apiHost, proto, resourceType, clientID, clientSecret, endpoint, apiVersion string
	timeout, httpRetry                                                                      int
	log                                                                                     logrus.FieldLogger
)

const httpTimeout = 10 * time.Second

type OutputCollection []Output

type Output struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

func init() {
	flag.StringVar(&accessToken, "token", "", "access token used to make request to bcda")
	flag.StringVar(&clientID, "clientID", "", "client id for retrieving an access token")
	flag.StringVar(&clientSecret, "clientSecret", "", "client secret for retrieving an access token")
	flag.StringVar(&apiHost, "host", "localhost:3000", "host to send requests to")
	flag.StringVar(&proto, "proto", "http", "protocol to use")
	flag.StringVar(&resourceType, "resourceType", "", "resourceType to test")
	flag.StringVar(&apiVersion, "apiVersion", "v1", "resourceType to test")
	flag.IntVar(&timeout, "timeout", 300, "amount of time to wait for file to be ready and downloaded.")
	flag.StringVar(&endpoint, "endpoint", "", "base type of request endpoint in the format of Patient or Group/all or Group/new")
	flag.IntVar(&httpRetry, "httpRetry", 4, "amount of times to retry an http request")
	flag.Parse()
}

func main() {
	c := NewClient(accessToken, httpRetry)

	// Set the timeouts before throwing an error
	http.DefaultClient.Timeout = httpTimeout
	c.httpClient.Timeout = httpTimeout

	logFields := logrus.Fields{
		"endpoint":      endpoint,
		"resourceTypes": resourceType,
	}

	l := logrus.StandardLogger()
	l.SetReportCaller(true)
	l.SetFormatter(&logrus.JSONFormatter{})
	log = logrus.NewEntry(l).WithFields(logFields)

	log.Infof("bulk data request to %s endpoint with %s resource types", endpoint, resourceType)

	jobURL, err := startJob(c, endpoint, resourceType)
	if err != nil {
		log.Errorf("Failed to start job %s", err.Error())
		os.Exit(1)
	}
	urls, err := getDataURLs(c, jobURL, time.Duration(timeout)*time.Second)
	if err != nil {
		log.Errorf("Failed to get data URLs %s", err.Error())
		os.Exit(1)
	}

	for _, u := range urls {
		data, err := getData(c, u)
		if err != nil {
			log.Errorf("Failed to get data from URL %s %s", u, err.Error())
		}

		log.Infof("Finished downloading data from url %s", u)
		if err := validateData(data); err != nil {
			log.Errorf("Failed to validate data %s", err.Error())
		}
		log.Infof("Finished validating data from url %s", u)
	}
}

func NewClient(accessToken string, retries int) *client {
	c := &client{accessToken: accessToken}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = retries
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			log.Info("Access token expired. Refreshing...")
			if err := c.updateAccessToken(); err != nil {
				return true, fmt.Errorf("failed to update access token %s", err.Error())
			}
			return true, nil
		}

		// The default policy does the rest of the retry lifting, it will retry:
		//   It will not retry when ctx contains Canceled or DeadlineExceeded
		//   It will not retry when err is a redirect, invalid protocol scheme, or TLS cert verification error
		//   It will retry all 5xx errors EXCEPT 501 Not Implemented
		//   Is will retry a 429 Too Many Requests
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	c.httpClient = retryClient.StandardClient()
	return c
}

func startJob(c *client, endpoint, resourceType string) (string, error) {
	var url string

	if resourceType != "" {
		url = fmt.Sprintf("%s://%s/api/%s/%s/$export?_type=%s", proto, apiHost, apiVersion, endpoint, resourceType)
	} else {
		url = fmt.Sprintf("%s://%s/api/%s/%s/$export", proto, apiHost, apiVersion, endpoint)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("Failed to construct resquest with URL %s", url, err.Error())
		panic(err)
	}

	req.Header.Add("Prefer", "respond-async")
	req.Header.Add("Accept", "application/fhir+json")

	resp, err := c.Do(req)
	if err != nil {
		log.Errorf("Exception occurred during request call %s", req, err.Error())
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		return resp.Header.Get("Content-Location"), nil
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf(constants.RespBodyErr, err.Error())
		}
		return "", fmt.Errorf(constants.RespCodeErr,
			req.URL.String(), resp.StatusCode, body)
	}
}

func getDataURLs(c *client, jobEndpoint string, timeout time.Duration) ([]string, error) {
	check := func() ([]string, error) {
		req, err := http.NewRequest("GET", jobEndpoint, nil)
		if err != nil {
			log.Errorf("Failure to construct request %s", jobEndpoint, err.Error())
			return nil, err
		}
		resp, err := c.Do(req)
		if err != nil {
			log.Errorf("Exception occurred during call %s", req, err.Error())
			return nil, err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var objmap map[string]json.RawMessage
			if err := json.NewDecoder(resp.Body).Decode(&objmap); err != nil {
				log.Errorf("Failure to decode the response body %s", resp.Body, err.Error())
				return nil, err
			}

			output := objmap["output"]
			var data OutputCollection
			if err := json.Unmarshal(output, &data); err != nil {
				log.Errorf("Failure to unmarshal data %s", &data, err.Error())
				return nil, err
			}

			var urls []string
			for _, item := range data {
				urls = append(urls, item.URL)
			}
			return urls, nil

		case http.StatusAccepted:
			log.Infof("Job has not completed %s", resp.Header.Get("X-Progress"))
			<-time.After(5 * time.Second)
			return nil, nil
		default:
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Errorf(constants.RespBodyErr, err.Error())
			}
			return nil, fmt.Errorf(constants.RespCodeErr,
				req.URL.String(), resp.StatusCode, body)
		}
	}

	expire := time.After(timeout)
	for {
		select {
		case <-expire:
			return nil, fmt.Errorf("failed to get response in %s", timeout.String())
		default:
			urls, err := check()
			if err != nil {
				log.Errorf("Failure on url check %s", urls, err.Error())
				return nil, err
			}
			if len(urls) == 0 {
				continue
			}
			return urls, nil
		}
	}
}

func getData(c *client, dataURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", dataURL, nil)
	if err != nil {
		log.Errorf("Failure to construct request %s", dataURL, err.Error())
		return nil, err
	}
	req.Header.Add("Accept-Encoding", "gzip")
	resp, err := c.Do(req)
	if err != nil {
		log.Errorf("Exception occurred when making the call %s", req, err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failure to read response body %s", req, err.Error())
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(constants.RespCodeErr,
			req.URL.String(), resp.StatusCode, body)
	}

	return body, nil
}

func validateData(encoded []byte) error {
	// Data is expected to be gzip encoded
	reader, err := gzip.NewReader(bytes.NewReader(encoded))
	if err != nil {
		log.Errorf("Failure to construct gzip reader %s", encoded, err.Error())
		return err
	}
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		log.Errorf("Failure to read decoded information in gzip %s", reader, err.Error())
		return err
	}

	if !isValidNDJSONText(string(decoded)) {
		return errors.New("data is not valid NDJSON format")
	}

	return nil
}

func isValidNDJSONText(data string) bool {
	isValid := true

	// blank file is not valid
	if len(data) == 0 {
		return false
	}

	for _, line := range strings.Split(data, "\n") {
		if len(line) == 0 {
			continue
		}
		if !json.Valid([]byte(line)) {
			isValid = false
			log.Info(line)
			break
		}
	}

	return isValid
}

type client struct {
	httpClient  *http.Client
	accessToken string
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	if len(c.accessToken) == 0 {
		log.Info("No access token supplied. Refreshing...")
		if err := c.updateAccessToken(); err != nil {
			return nil, fmt.Errorf("failed to update access token %s", err.Error())
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to receive response with error: %w", err)
	}

	return resp, nil
}

func (c *client) updateAccessToken() error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s/auth/token", proto, apiHost), nil)
	if err != nil {
		log.Errorf("Failure to construct request %s", req, err.Error())
		return err
	}

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Add("Accept", "application/json")

	// The retry logic may try to update the access token, to avoid a recursive
	// retry loop we dont want to use the retry logic on updating the access token
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("Exception occurred when making request call %s", req, err.Error())
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf(constants.RespBodyErr, err.Error())
		}
		return fmt.Errorf(constants.RespCodeErr,
			req.URL.String(), resp.StatusCode, body)
	}

	type tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in,omitempty"`
		TokenType   string `json:"token_type"`
	}

	var t = tokenResponse{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failure to read response body %s", resp.Body, err.Error())
		return err
	}

	if err = json.Unmarshal(body, &t); err != nil {
		return fmt.Errorf("failed to parse '%s' into token %s", string(body), err.Error())
	}

	c.accessToken = t.AccessToken
	return nil
}
