package main

import (
	"compress/gzip"
	"encoding/json"
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
	accessToken, apiHost, proto, resourceType, clientID, clientSecret, endpoint string
	timeout                                                                     int
)

type OutputCollection []Output

type Output struct {
	Url  string `json:"url"`
	Type string `json:"type"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   string `json:"expires_in,omitempty"`
}

func init() {
	flag.StringVar(&accessToken, "token", "", "access token used to make request to bcda")
	flag.StringVar(&clientID, "clientID", "", "client id for retrieving an access token")
	flag.StringVar(&clientSecret, "clientSecret", "", "client secret for retrieving an access token")
	flag.StringVar(&apiHost, "host", "localhost:3000", "host to send requests to")
	flag.StringVar(&proto, "proto", "http", "protocol to use")
	flag.StringVar(&resourceType, "resourceType", "", "resourceType to test")
	flag.IntVar(&timeout, "timeout", 300, "amount of time to wait for file to be ready and downloaded.")
	flag.StringVar(&endpoint, "endpoint", "", "base type of request endpoint in the format of Patient or Group/all or Group/new")
	flag.Parse()

	if accessToken == "" {
		if clientID == "" && clientSecret == "" {
			panic(fmt.Errorf("Must provide a token or credentials for retrieving a token"))
		}

		fmt.Println("Access Token not supplied.  Retrieving one to use.")
		accessToken = getAccessToken()
	}
}

func getAccessToken() string {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s/auth/token", proto, apiHost), nil)
	if err != nil {
		panic(err)
	}

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Add("Accept", "application/json")

	resp, err := client().Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	var t = TokenResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&t); err != nil {
		panic(fmt.Sprintf("unexpected token response format: %s", err.Error()))
	}

	return t.AccessToken
}

func startJob(endpoint, resourceType string) *http.Response {
	client := &http.Client{}
	var url string

	if resourceType != "" {
		url = fmt.Sprintf("%s://%s/api/v1/%s/$export?_type=%s", proto, apiHost, endpoint, resourceType)
	} else {
		url = fmt.Sprintf("%s://%s/api/v1/%s/$export", proto, apiHost, endpoint)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Prefer", "respond-async")
	req.Header.Add("Accept", "application/fhir+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	return resp
}

func get(location string, compressed bool) *http.Response {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET", location, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	if compressed {
		req.Header.Add("Accept-Encoding", "gzip")
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	return resp
}

func writeFile(resp *http.Response, filename string) {
	defer resp.Body.Close()

	var isGZIP bool
	for _, h := range resp.Header.Values("Content-Encoding") {
		if h == "gzip" {
			isGZIP = true
			break
		}
	}

	if !isGZIP {
		panic("Data responses should be gzip-encoded.")
	}

	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	/* #nosec */
	out, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer utils.CloseFileAndLogError(out)

	/* #nosec G110 - not concerned with OOM since we're using this in a controlled setting */
	num, err := io.Copy(out, reader)
	if err != nil && num <= 0 {
		panic(err)
	}
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

func main() {
	fmt.Printf("bulk data request to %s endpoint with %s resource types\n", endpoint, resourceType)
	end := time.Now().Add(time.Duration(timeout) * time.Second)
	if result := startJob(endpoint, resourceType); result.StatusCode == 202 {
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
				accessToken = getAccessToken()
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
					accessToken = getAccessToken()

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
		fmt.Printf("error: failed to start %s %s data aggregation job\n", endpoint, resourceType)
		os.Exit(1)
	}
}

func client() *http.Client {
	return &http.Client{Timeout: time.Second * 10}
}
