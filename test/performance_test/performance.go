package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	vegeta "github.com/tsenart/vegeta/lib"
	"github.com/tsenart/vegeta/lib/plot"
)

// might add a with metrics bool option
var (
	appTestToken, workerTestToken, apiHost, proto, endpoint, reportFilePath string
	encrypt                                                                 bool
)

func init() {
	flag.StringVar(&appTestToken, "api_test_token", "", "access token used for api performance testing")
	flag.StringVar(&workerTestToken, "worker_test_token", "", "access token used for worker performance testing")
	flag.StringVar(&apiHost, "host", "localhost:3000", "host to send requests to")
	flag.StringVar(&proto, "proto", "http", "protocol to use")
	flag.StringVar(&endpoint, "endpoint", "ExplanationOfBenefit", "endpoint to test")
	flag.BoolVar(&encrypt, "encrypt", true, "whether to disable encryption")
	flag.Parse()

	// create folder if doesn't exist for storing the results, maybe create env var later
	reportFilePath = "results/"
	if _, err := os.Stat(reportFilePath); os.IsNotExist(err) {
		err := os.MkdirAll(reportFilePath, 0755)
		if err != nil {
			panic(err)
		}
	}

}

func main() {
	var buf bytes.Buffer

	if appTestToken != "" {
		targeter := makeTarget(appTestToken)
		apiResults := runAPITest(targeter)
		apiResults.WriteTo(&buf)
	}

	if workerTestToken != "" {
		targeter := makeTarget(workerTestToken)
		workerResults := runWorkerTest(targeter)
		workerResults.WriteTo(&buf)
	}

	data := buf.Bytes()
	if len(data) > 0 {
		writeResults(data)
	}
}

func makeTarget(accessToken string) vegeta.Targeter {
	url := fmt.Sprintf("%s://%s/api/v1/%s/$export", proto, apiHost, endpoint)
	if !encrypt {
		url = fmt.Sprintf("%s?encrypt=false", url)
	}

	header := map[string][]string{
		"Prefer":        {"respond-async"},
		"Accept":        {"application/fhir+json"},
		"Authorization": {fmt.Sprintf("Bearer %s", accessToken)},
	}

	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    url,
		Header: header,
	})
	return targeter
}

func runAPITest(target vegeta.Targeter) *plot.Plot {
	fmt.Printf("running api performance for: %s\n", endpoint)
	// 600 request a minute for 1 minute
	duration := 1 * time.Minute
	rate := vegeta.Rate{Freq: 600, Per: time.Minute}

	plot := plot.New()
	defer plot.Close()

	attacker := vegeta.NewAttacker()
	for results := range attacker.Attack(target, rate, duration, fmt.Sprintf("apiTest:_%s", endpoint)) {
		plot.Add(results)
	}
	return plot
}

func runWorkerTest(target vegeta.Targeter) *plot.Plot {
	fmt.Printf("running worker performance for: %s\n", endpoint)
	// 1 request for 300,000 beneficiaries
	duration := 1 * time.Minute
	rate := vegeta.Rate{Freq: 1, Per: time.Minute}

	plot := plot.New()
	defer plot.Close()

	attacker := vegeta.NewAttacker()
	for results := range attacker.Attack(target, rate, duration, fmt.Sprintf("workerTest:_%s", endpoint)) {
		plot.Add(results)
	}
	return plot
}

func writeResults(data []byte) {
	fmt.Printf("Writing results:")
	err := ioutil.WriteFile(reportFilePath+"/performance_plot.html", data, 0755)
	if err != nil {
		panic(err)
	}
}
