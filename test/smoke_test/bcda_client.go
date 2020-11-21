package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	accessToken, apiHost, proto, resourceType, clientID, clientSecret, endpoint, apiVersion string
	timeout, httpRetry                                                                      int
)

type OutputCollection []Output

type Output struct {
	Url  string `json:"url"`
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
	flag.IntVar(&httpRetry, "httpRetry", 3, "amount of times to retry an http request")
	flag.Parse()

	log.SetReportCaller(true)
}

func main() {
	c := &client{httpClient: &http.Client{Timeout: 10 * time.Second}, accessToken: accessToken, retries: httpRetry}

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

func startJob(c *client, endpoint, resourceType string) (string, error) {
	var url string

	if resourceType != "" {
		url = fmt.Sprintf("%s://%s/api/%s/%s/$export?_type=%s", proto, apiHost, apiVersion, endpoint, resourceType)
	} else {
		url = fmt.Sprintf("%s://%s/api/%s/%s/$export", proto, apiHost, apiVersion, endpoint)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Prefer", "respond-async")
	req.Header.Add("Accept", "application/fhir+json")

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusAccepted:
		return resp.Header.Get("Content-Location"), nil
	default:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read response body %s", err.Error())
		}
		return "", fmt.Errorf("request %s has unexpected response code received %d, body '%s'",
			req.URL.String(), resp.StatusCode, body)
	}
}

func getDataURLs(c *client, jobEndpoint string, timeout time.Duration) ([]string, error) {
	check := func() ([]string, error) {
		req, err := http.NewRequest("GET", jobEndpoint, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var objmap map[string]json.RawMessage
			if err := json.NewDecoder(resp.Body).Decode(&objmap); err != nil {
				return nil, err
			}

			output := objmap["output"]
			var data OutputCollection
			if err := json.Unmarshal(output, &data); err != nil {
				return nil, err
			}

			var urls []string
			for _, item := range data {
				urls = append(urls, item.Url)
			}
			return urls, nil

		case http.StatusAccepted:
			log.Infof("Job has not completed %s", resp.Header.Get("X-Progress"))
			<-time.After(5 * time.Second)
			return nil, nil
		default:
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Errorf("Failed to read response body %s", err.Error())
			}
			return nil, fmt.Errorf("request %s has unexpected response code received %d, body '%s'",
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
		return nil, err
	}
	req.Header.Add("Accept-Encoding", "gzip")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request %s has unexpected response code received %d, body '%s'",
			req.URL.String(), resp.StatusCode, body)
	}

	return body, nil
}

func validateData(encoded []byte) error {
	// Data is expected to be gzip encoded
	reader, err := gzip.NewReader(bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	defer reader.Close()

	decoded, err := ioutil.ReadAll(reader)
	if err != nil {
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

	retries int
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	if len(c.accessToken) == 0 {
		log.Info("No access token supplied. Refreshing...")
		if err := c.updateAccessToken(); err != nil {
			return nil, fmt.Errorf("failed to update access token %s", err.Error())
		}
	}
	for i := 0; i <= c.retries; i++ {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized {
			log.Info("Access token expired. Refreshing...")
			if err := c.updateAccessToken(); err != nil {
				return nil, fmt.Errorf("failed to update access token %s", err.Error())
			}
			// Retry request with new access token
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("failed to receive response after %d tries", c.retries)
}

func (c *client) updateAccessToken() error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s/auth/token", proto, apiHost), nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Add("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read response body %s", err.Error())
		}
		return fmt.Errorf("request %s has unexpected response code received %d, body '%s'",
			req.URL.String(), resp.StatusCode, body)
	}

	type tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   string `json:"expires_in,omitempty"`
	}

	var t = tokenResponse{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(body, &t); err != nil {
		return fmt.Errorf("failed to parse '%s' into token %s", string(body), err.Error())
	}

	c.accessToken = t.AccessToken
	return nil
}
