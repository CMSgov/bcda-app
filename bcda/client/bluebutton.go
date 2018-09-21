package client

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func GetBlueButtonPatientData() (string, error) {
	return getBlueButtonData("/baseDstu3/Patient/?_id={}&_format=application%2Ffhir%2Bjson")
}

func GetBlueButtonCoverageData() (string, error) {
	return getBlueButtonData("/baseDstu3/Coverage/?beneficiary={}&_format=application%2Ffhir%2Bjson")
}

func GetBlueButtonExplanationOfBenefitData() (string, error) {
	return getBlueButtonData("/baseDstu3/ExplanationOfBenefit/?patient={}&_format=application%2Ffhir%2Bjson")
}

func GetBlueButtonMetadata() (string, error) {
	return getBlueButtonData("/baseDstu3/metadata/?_format=application%2Ffhir%2Bjson")
}

func getBlueButtonData(path string) (string, error) {
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

	resp, err := client.Get(bbServer + path)
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
