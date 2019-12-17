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
	appTestToken, workerTestToken, apiHost, proto, resourceType, reportFilePath, endpoint string
	freq, duration                                                                        int
)

func init() {
	flag.StringVar(&appTestToken, "api_test_token", "", "access token used for api performance testing")
	flag.StringVar(&workerTestToken, "worker_test_token", "", "access token used for worker performance testing")
	flag.StringVar(&apiHost, "host", "localhost:3000", "host to send requests to")
	flag.IntVar(&duration, "duration", 60, "seconds: the total time to run the test")
	flag.IntVar(&freq, "freq", 10, "the number of requests per second")
	flag.StringVar(&proto, "proto", "http", "protocol to use")
	flag.StringVar(&resourceType, "resourceType", "", "resourceType to test")
	flag.StringVar(&reportFilePath, "report_path", "../../test_results/performance", "path to write the result.html")
	flag.StringVar(&endpoint, "endpoint", "", "base type of request endpoint")
	flag.Parse()

	// create folder if doesn't exist for storing the results
	if _, err := os.Stat(reportFilePath); os.IsNotExist(err) {
		err := os.MkdirAll(reportFilePath, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	if appTestToken != "" {
		targeter := makeTarget(appTestToken)
		apiResults := runAPITest(targeter)
		var buf bytes.Buffer
		_, err := apiResults.WriteTo(&buf)
		if err != nil {
			panic(err)
		}
		writeResults(fmt.Sprintf("%s_%s_api_plot", endpoint, resourceType), buf)
	}

	if workerTestToken != "" {
		targeter := makeTarget(workerTestToken)
		workerResults := runWorkerTest(targeter)
		var buf bytes.Buffer
		_, err := workerResults.WriteTo(&buf)
		if err != nil {
			panic(err)
		}
		// this will be monitored via new relic, but we have lots of flexibility going forward.
	}
}

func makeTarget(accessToken string) vegeta.Targeter {
	if endpoint == "Group" {
		groupId := "all"
		endpoint = fmt.Sprintf("%s/%s",endpoint, groupId)
	}

	var url string
	if resourceType != "" {
		url = fmt.Sprintf("%s://%s/api/v1/%s/$export?_type=%s", proto, apiHost, endpoint, resourceType)
	} else {
		url = fmt.Sprintf("%s://%s/api/v1/%s/$export", proto, apiHost, endpoint)
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
	fmt.Printf("running api performance for: %s_%s\n", endpoint, resourceType)
	title := plot.Title(fmt.Sprintf("apiTest_%s_%s", endpoint, resourceType))
	p := plot.New(title)
	defer p.Close()

	// 10 request every second for 60 seconds = 600 total calls
	d := time.Second * time.Duration(duration)
	rate := vegeta.Rate{Freq: freq, Per: time.Second}
	plotAttack(p, target, rate, d)

	return p
}

func runWorkerTest(target vegeta.Targeter) *plot.Plot {
	fmt.Printf("running worker performance for: %s_%s\n", endpoint, resourceType)
	title := plot.Title(fmt.Sprintf("workerTest_%s_%s", endpoint, resourceType))
	p := plot.New(title)
	defer p.Close()

	// 1 request for 300,000 beneficiaries
	d := time.Minute
	rate := vegeta.Rate{Freq: 1, Per: time.Minute}
	plotAttack(p, target, rate, d)

	return p
}

// need to make rate into some sort of pretty string format
func plotAttack(p *plot.Plot, t vegeta.Targeter, r vegeta.Rate, du time.Duration) {
	attacker := vegeta.NewAttacker()
	for results := range attacker.Attack(t, r, du, fmt.Sprintf("%dps:", r.Freq)) {
		err := p.Add(results)
		if err != nil {
			panic(err)
		}
	}
}

func writeResults(filename string, buf bytes.Buffer) {
	data := buf.Bytes()
	if len(data) > 0 {
		fn := fmt.Sprintf("%s/%s.html", reportFilePath, filename)
		fmt.Printf("Writing results: %s\n", fn)
		err := ioutil.WriteFile(fn, data, 0644)
		if err != nil {
			panic(err)
		}
	}
}
