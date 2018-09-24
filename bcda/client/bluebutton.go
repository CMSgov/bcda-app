package client

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

func GetBlueButtonPatientData(patientID string) (string, error) {
	params := url.Values{}
	params.Set("_id", patientID)
	params.Set("_format", "application/fhir+json")
	return getBlueButtonData("/baseDstu3/Patient/", params)
}

func GetBlueButtonCoverageData(beneficiaryID string) (string, error) {
	params := url.Values{}
	params.Set("beneficiary", beneficiaryID)
	params.Set("_format", "application/fhir+json")
	return getBlueButtonData("/baseDstu3/Coverage/", params)
}

func GetBlueButtonExplanationOfBenefitData(patientID string) (string, error) {
	params := url.Values{}
	params.Set("patient", patientID)
	params.Set("_format", "application/fhir+json")
	return getBlueButtonData("/baseDstu3/ExplanationOfBenefit/", params)
}

func GetBlueButtonMetadata() (string, error) {
	params := url.Values{}
	params.Set("_format", "application/fhir+json")
	return getBlueButtonData("/baseDstu3/metadata/", params)
}

func getBlueButtonData(path string, params url.Values) (string, error) {
	certFile := os.Getenv("BB_CLIENT_CERT_FILE")
	keyFile := os.Getenv("BB_CLIENT_KEY_FILE")

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return "", err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}
	bbServer := fmt.Sprintf("https://%s", os.Getenv("BB_SERVER_HOST"))

	req, err := http.NewRequest("GET", bbServer+path, nil)
	if err != nil {
		return "", err
	}

	req.URL.RawQuery = params.Encode()
	addRequestHeaders(req)

	resp, err := client.Do(req)
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
