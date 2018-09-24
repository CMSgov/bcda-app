package client

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type BlueButtonClient struct {
	httpClient http.Client
}

func NewBlueButtonClient() *BlueButtonClient {
	certFile := os.Getenv("BB_CLIENT_CERT_FILE")
	keyFile := os.Getenv("BB_CLIENT_KEY_FILE")
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	return &BlueButtonClient{*client}
}

func (bbc *BlueButtonClient) GetPatientData(patientID string) (string, error) {
	params := url.Values{}
	params.Set("_id", patientID)
	params.Set("_format", "application/fhir+json")
	return bbc.getData("/baseDstu3/Patient/", params)
}

func (bbc *BlueButtonClient) GetCoverageData(beneficiaryID string) (string, error) {
	params := url.Values{}
	params.Set("beneficiary", beneficiaryID)
	params.Set("_format", "application/fhir+json")
	return bbc.getData("/baseDstu3/Coverage/", params)
}

func (bbc *BlueButtonClient) GetExplanationOfBenefitData(patientID string) (string, error) {
	params := url.Values{}
	params.Set("patient", patientID)
	params.Set("_format", "application/fhir+json")
	return bbc.getData("/baseDstu3/ExplanationOfBenefit/", params)
}

func (bbc *BlueButtonClient) GetMetadata() (string, error) {
	params := url.Values{}
	params.Set("_format", "application/fhir+json")
	return bbc.getData("/baseDstu3/metadata/", params)
}

func (bbc *BlueButtonClient) getData(path string, params url.Values) (string, error) {
	bbServer := os.Getenv("BB_SERVER_LOCATION")

	req, err := http.NewRequest("GET", bbServer+path, nil)
	if err != nil {
		return "", err
	}

	req.URL.RawQuery = params.Encode()
	addRequestHeaders(req)

	resp, err := bbc.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func addRequestHeaders(req *http.Request) {
	// https://github.com/CMSgov/bluebutton-web-server/blob/master/apps/fhir/bluebutton/utils.py#L224-L231
	req.Header.Add("BlueButton-OriginalQueryTimestamp", time.Now().String())
	req.Header.Add("BlueButton-OriginalQueryId", "1")
	req.Header.Add("BlueButton-OriginalQueryCounter", "1")
	req.Header.Add("BlueButton-BeneficiaryId", "")
	req.Header.Add("BlueButton-UserId", "")
	req.Header.Add("BlueButton-Application", "")
	req.Header.Add("BlueButton-ApplicationId", "")
	req.Header.Add("BlueButton-DeveloperId", "")
	req.Header.Add("BlueButton-Developer", "")
	req.Header.Add("BlueButton-OriginatingIpAddress", "")

	req.Header.Add("keep-alive", "")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("X-Forwarded-Host", "")

	req.Header.Add("BlueButton-OriginalUrl", "")
	req.Header.Add("BlueButton-OriginalQuery", "")
	req.Header.Add("BlueButton-BackendCall", "")
}
