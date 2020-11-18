package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/utils"
)

var (
	accessToken, apiHost, proto, resourceType, clientID, clientSecret, endpoint, apiVersion string
	timeout                                                                                 int
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
	flag.Parse()

}

func main() {
	c := &client{httpClient: &http.Client{Timeout: 10 * time.Second}, accessToken: accessToken}

	fmt.Printf("bulk data request to %s endpoint with %s resource types\n", endpoint, resourceType)
	end := time.Now().Add(time.Duration(timeout) * time.Second)

	resp, err := startJob(c, endpoint, resourceType)
	if err != nil {
		fmt.Printf("Failed to start job %s", err.Error())
		os.Exit(1)
	}

	defer resp.Body.Close()

	if result := startJob(endpoint, resourceType); result.StatusCode == 202 {
		if err := result.Body.Close(); err != nil {
			fmt.Println("Failed to close body " + err.Error())
		}

		for {
			<-time.After(5 * time.Second)

			if time.Now().After(end) {
				fmt.Println("timeout exceeded...")
				os.Exit(1)
			}

			fmt.Println("checking job status...")
			status := get(result.Header["Content-Location"][0], false)

			// Acquire new token if the current token has expired
			if status.StatusCode == 401 {
				fmt.Println("acquire new token...")
				if accessToken, err = getAccessToken(); err != nil {
					fmt.Printf("Failed to get access token %s", err.Error())
					os.Exit(1)
				}
			} else if status.StatusCode == 200 {
				fmt.Println("file is ready for download...")

				defer status.Body.Close()

				var objmap map[string]*json.RawMessage
				err := json.NewDecoder(status.Body).Decode(&objmap)
				if err != nil {
					panic(err)
				}
				output := (*json.RawMessage)(objmap["output"])
				var data OutputCollection
				if err := json.Unmarshal(*output, &data); err != nil {
					panic(err)
				}

				for _, fileItem := range data {

					// Acquire new access token for each file download
					if accessToken, err = getAccessToken(); err != nil {
						fmt.Printf("Failed to get access token %s", err.Error())
						os.Exit(1)
					}
					fmt.Printf("fetching: %s\n", fileItem.Url)
					download := get(fileItem.Url, true)
					if download.StatusCode == 200 {
						filename := "/tmp/" + path.Base(fileItem.Url)
						fmt.Printf("writing download to disk: %s\n", filename)
						writeFile(download, filename)

						fmt.Println("validating file...")
						fi, err := os.Stat(filename)
						if err != nil {
							panic(err)
						}
						if fi.Size() <= 0 {
							fmt.Println("Error: file is empty!.")
							os.Exit(1)
						}

						fmt.Println("reading file contents...")
						// #nosec
						bytes, err := ioutil.ReadFile(filename)
						if err != nil {
							panic(err)
						}
						output := string(bytes)

						if !isValidNDJSONText(output) {
							fmt.Println("Error: file is not in valid NDJSON format!")
							os.Exit(1)
						}

						fmt.Println("done.")

					} else {
						fmt.Printf("error: unable to request file download... status is: %s\n", download.Status)
						os.Exit(1)
					}
				}
				break

			} else {
				fmt.Println("  => job is still pending. waiting...")
			}
		}
	} else {
		fmt.Printf("error: failed to start %s data aggregation job\n", result.Request.URL.String())
		body, err := ioutil.ReadAll(result.Body)
		if err != nil {
			fmt.Printf("Failed to read response body %s\n", err.Error())
		}
		if err := result.Body.Close(); err != nil {
			fmt.Println("Failed to close body " + err.Error())
		}
		fmt.Printf("respCode %d respBody %s\n", result.StatusCode, string(body))
		os.Exit(1)
	}
}

func startJob(c *client, endpoint, resourceType string) (*http.Response, error) {
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
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func get(c *client, location string, compressed bool) (*http.Response, error) {
	req, err := http.NewRequest("GET", location, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	if compressed {
		req.Header.Add("Accept-Encoding", "gzip")
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func writeFile(resp *http.Response, filename string) error {
	defer resp.Body.Close()

	var isGZIP bool
	for _, h := range resp.Header.Values("Content-Encoding") {
		if h == "gzip" {
			isGZIP = true
			break
		}
	}

	if !isGZIP {
		return errors.New("Data responses should be gzip-encoded.")
	}

	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer reader.Close()

	/* #nosec */
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer utils.CloseFileAndLogError(out)

	/* #nosec G110 - not concerned with OOM since we're using this in a controlled setting */
	num, err := io.Copy(out, reader)
	if err != nil {
		return err
	}
	if num <= 0 {
		return errors.New("failed to write data")
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
			fmt.Println(line)
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

	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

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

<<<<<<< HEAD
			} else if status.StatusCode == http.StatusInternalServerError {
				fmt.Printf("Job failed! Path %s\n", status.Request.URL.String())
				os.Exit(1)
			} else {
				fmt.Println("  => job is still pending. waiting...")
			}
		}
	} else {
		fmt.Printf("error: failed to start %s data aggregation job\n", result.Request.URL.String())
		body, err := ioutil.ReadAll(result.Body)
		if err != nil {
			fmt.Printf("Failed to read response body %s\n", err.Error())
		}
		if err := result.Body.Close(); err != nil {
			fmt.Println("Failed to close body " + err.Error())
		}
		fmt.Printf("respCode %d respBody %s\n", result.StatusCode, string(body))
		os.Exit(1)
=======
	if err = json.Unmarshal(body, &t); err != nil {
		return fmt.Errorf("failed to parse '%s' into token %s", string(body), err.Error())
>>>>>>> WIP
	}

	c.accessToken = t.AccessToken
	return nil
}
