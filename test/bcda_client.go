package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	accessToken, apiHost, proto string
)

type OutputCollection []Output

type Output struct {
	Url  string `json:"url"`
	Type string `json:"type"`
}

func init() {
	flag.StringVar(&accessToken, "token", "", "access token used to make request to bcda")
	flag.StringVar(&apiHost, "host", "localhost:3000", "host to send requests to")
	flag.StringVar(&proto, "proto", "http", "protocol to use")
	flag.Parse()

	if accessToken == "" {
		fmt.Println("error: access token is required")
		os.Exit(1)
	}
}

func startJob() *http.Response {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET", fmt.Sprintf("%s://%s/api/v1/ExplanationOfBenefit/$export", proto, apiHost), nil)

	req.Header.Add("Prefer", "respond-async")
	req.Header.Add("Accept", "application/fhir+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	return resp
}

func checkStatus(location string) *http.Response {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET", location, nil)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	return resp
}

func getFile(location string) *http.Response {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET", location, nil)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	return resp
}

func writeFile(resp *http.Response) {
	defer resp.Body.Close()
	out, err := os.Create("/tmp/download.json")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	io.Copy(out, resp.Body)
}

func main() {
	fmt.Println("making request to start data aggregation job")
	if result := startJob(); result.StatusCode == 202 {
		for {
			<-time.After(1 * time.Second)

			fmt.Println("checking job status...")
			status := checkStatus(result.Header["Content-Location"][0])

			if status.StatusCode == 200 {
				fmt.Println("file is ready for download...")
				defer status.Body.Close()

				var objmap map[string]*json.RawMessage
				json.NewDecoder(status.Body).Decode(&objmap)
				output := (*json.RawMessage)(objmap["output"])

				var data OutputCollection
				if err := json.Unmarshal(*output, &data); err != nil {
					panic(err)
				}

				fmt.Printf("fetching: %s\n", data[0].Url)
				download := getFile(data[0].Url)
				if download.StatusCode == 200 {
					fmt.Println("writing download to disk: /tmp/download.json")
					writeFile(download)
					fmt.Println("done.")
				} else {
					fmt.Println("error: unable to request file download.")
					os.Exit(1)
				}

				break
			}
			fmt.Println("  => job is still pending. waiting...")
		}
	} else {
		fmt.Println("error: failed to start data aggregation job")
		os.Exit(1)
	}
}
