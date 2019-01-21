package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"
)

var (
	accessToken, apiHost, proto, endpoint string
	timeout                               int
	encrypt                               bool
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
	flag.StringVar(&endpoint, "endpoint", "ExplanationOfBenefit", "endpoint to test")
	flag.IntVar(&timeout, "timeout", 300, "amount of time to wait for file to be ready and downloaded.")
	flag.BoolVar(&encrypt, "encrypt", false, "whether to request encryption of data")
	flag.Parse()

	if accessToken == "" {
		fmt.Println("Access Token not supplied.  Retrieving one to use.")
		accessToken = getAccessToken()
	}
}

func getAccessToken() string {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET", fmt.Sprintf("%s://%s/api/v1/token", proto, apiHost), nil)
	if err != nil {
		panic(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(respData)
}

func startJob(resourceType string) *http.Response {
	client := &http.Client{}

	var url string = fmt.Sprintf("%s://%s/api/v1/%s/$export", proto, apiHost, resourceType)
	if encrypt {
		url = fmt.Sprintf("%s?encrypt=true", url)
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

func checkStatus(location string) *http.Response {
	client := &http.Client{}
	req, err := http.NewRequest(
		"GET", location, nil)
	if err != nil {
		panic(err)
	}

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
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	return resp
}

func writeFile(resp *http.Response, filename string) {
	defer resp.Body.Close()
	/* #nosec */
	out, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	num, err := io.Copy(out, resp.Body)
	if err != nil && num <= 0 {
		panic(err)
	}
}

func isValidNDJSONFile(filename string) bool {
	isValid := true
	/* #nosec */
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	r := bufio.NewReader(file)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if !json.Valid([]byte(line)) {
			isValid = false
			break
		}
	}

	return isValid
}

func main() {
	fmt.Printf("making request to start %s data aggregation job\n", endpoint)
	end := time.Now().Add(time.Duration(timeout) * time.Second)
	if result := startJob(endpoint); result.StatusCode == 202 {
		for {
			<-time.After(5 * time.Second)

			if time.Now().After(end) {
				fmt.Println("timeout exceeded...")
				os.Exit(1)
			}

			fmt.Println("checking job status...")
			status := checkStatus(result.Header["Content-Location"][0])

			if status.StatusCode == 200 {
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

				encryptData := map[string]string{}
				if encrypt {
					encOutput := (*json.RawMessage)(objmap["KeyMap"])
					if err := json.Unmarshal(*encOutput, &encryptData); err != nil {
						panic(err)
					}
				}

				fmt.Printf("fetching: %s\n", data[0].Url)
				download := getFile(data[0].Url)
				if download.StatusCode == 200 {
					filename := "/tmp/" + path.Base(data[0].Url)
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

					if encrypt {
						fmt.Println("decrypting the file...")
						encryptedKey, err := hex.DecodeString(encryptData[path.Base(data[0].Url)])
						if err != nil {
							panic(err)
						}
						privateKeyFile := os.Getenv("ATO_PRIVATE_KEY_FILE")
						privateKey := getPrivateKey(privateKeyFile)
						filename = decryptFile(privateKey, encryptedKey, filename)
						fmt.Printf("writing decrypted file to disk: %s\n", filename)
					}

					fmt.Println("validating file content...")
					if !isValidNDJSONFile(filename) {
						fmt.Println("Error: file is not in valid NDJSON format!")
						os.Exit(1)
					}

					fmt.Println("done.")
				} else {
					fmt.Printf("error: unable to request file download... status is: %s\n", download.Status)
					os.Exit(1)
				}

				break
			}
			fmt.Println("  => job is still pending. waiting...")
		}
	} else {
		fmt.Printf("error: failed to start %s data aggregation job\n", endpoint)
		os.Exit(1)
	}
}
