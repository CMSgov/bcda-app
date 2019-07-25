package client

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

var ssasLogger *logrus.Logger

// SSASClient is a client for interacting with the System-to-System Authentication Service.
type SSASClient struct {
	http.Client
}

func init() {
	ssasLogger = logrus.New()
	ssasLogger.Formatter = &logrus.JSONFormatter{}
	filePath := os.Getenv("BCDA_SSAS_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filepath.Clean(filePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

	if err == nil {
		ssasLogger.SetOutput(file)
	} else {
		ssasLogger.Info("Failed to open SSAS log file; using default stderr")
	}
}

// NewSSASClient creates and returns an SSASClient.
func NewSSASClient() (*SSASClient, error) {
	transport, err := tlsTransport()
	if err != nil {
		return nil, errors.Wrap(err, "SSAS client could not be created")
	}

	var timeout int
	if timeout, err = strconv.Atoi(os.Getenv("SSAS_TIMEOUT_MS")); err != nil {
		ssasLogger.Info("Could not get SSAS timeout from environment variable; using default value of 500.")
		timeout = 500
	}

	client := &http.Client{Transport: transport, Timeout: time.Duration(timeout) * time.Millisecond}

	return &SSASClient{*client}, nil
}

func tlsTransport() (*http.Transport, error) {
	certFile := os.Getenv("SSAS_CLIENT_CERT_FILE")
	keyFile := os.Getenv("SSAS_CLIENT_KEY_FILE")
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not load SSAS keypair")
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	caFile := os.Getenv("SSAS_CLIENT_CA_FILE")
	caCert, err := ioutil.ReadFile(filepath.Clean(caFile))
	if err != nil {
		return nil, errors.Wrap(err, "could not read CA file")
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, errors.New("could not append CA certificate(s)")
	}

	tlsConfig.RootCAs = caCertPool
	tlsConfig.BuildNameToCertificate()

	return &http.Transport{TLSClientConfig: tlsConfig}, nil
}

// CreateSystem POSTs to the SSAS /system endpoint to create a system.
func (c *SSASClient) CreateSystem() ([]byte, error) {
	return nil, nil
}

// GetPublicKey GETs the SSAS /system/{systemID}/key endpoint to retrieve a system's public key.
func (c *SSASClient) GetPublicKey(systemID int) ([]byte, error) {
	return nil, nil
}

// ResetCredentials PUTs to the SSAS /system/{systemID}/credentials endpoint to reset the system's credentials.
func (c *SSASClient) ResetCredentials(systemID int) ([]byte, error) {
	return nil, nil
}

// DeleteCredentials DELETEs from the SSAS /system/{systemID}/credentials endpoint to deactivate credentials associated with the system.
func (c *SSASClient) DeleteCredentials(systemID int) ([]byte, error) {
	return nil, nil
}
