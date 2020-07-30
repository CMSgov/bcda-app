package fhir

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strconv"
	"testing"

	models "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var logger = logrus.New()

func TestSingleRequestBundle(t *testing.T) {
	r := &requestHandler{
		countChecker: func(r *http.Request) {
			assert.Empty(t, r.URL.Query().Get("_count"))
		},
	}
	s := httptest.NewServer(r)
	defer s.Close()

	client := NewClient(http.DefaultClient, logger, 0)

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/bundleFull.json", s.URL), nil)
	assert.NoError(t, err)

	bundle, nextReq, err := client.DoBundleRequest(req)

	assert.NoError(t, err)
	assert.Nil(t, nextReq)
	assert.NotNil(t, bundle)

	assertEqualsBundle(t, "./testdata/bundleFull.json", bundle)
}

func TestMultipleRequestBundle(t *testing.T) {
	count := 10
	r := &requestHandler{
		countChecker: func(r *http.Request) {
			assert.Equal(t, strconv.Itoa(count), r.URL.Query().Get("_count"))
		},
	}
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	assert.NoError(t, err)
	testHost := u.Host

	client := NewClient(http.DefaultClient, logger, count)

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/bundlePartial1.json", s.URL), nil)
	assert.NoError(t, err)

	var bundle *models.Bundle
	for ok := true; ok; {
		b, next, err := client.DoBundleRequest(req)
		assert.NoError(t, err)
		assert.NotNil(t, b)

		if bundle == nil {
			bundle = b
		} else {
			bundle.Entries = append(bundle.Entries, b.Entries...)
		}

		if next == nil {
			ok = false
			continue
		}

		// The partial files do not know the correct port, so we'll update the request
		// to point to the test server
		next.URL.Host = testHost
		req = next
	}

	assert.Equal(t, 3, r.numRequestsReceived)
	assertEqualsBundle(t, "./testdata/bundlePartialComplete.json", bundle)
}

func TestRawRequest(t *testing.T) {
	msg := "Hello world!"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, msg)
	}))
	defer s.Close()

	client := NewClient(http.DefaultClient, logger, 0)

	req, err := http.NewRequest("GET", s.URL, nil)
	assert.NoError(t, err)

	resp, err := client.DoRaw(req)
	assert.NoError(t, err)
	assert.Equal(t, msg, resp)
}

func assertEqualsBundle(t *testing.T, pathToExpected string, actual *models.Bundle) {
	data, err := ioutil.ReadFile(pathToExpected)
	assert.NoError(t, err)
	var expected models.Bundle
	err = json.Unmarshal([]byte(data), &expected)
	assert.NoError(t, err)
	assert.Equal(t, expected, *actual)
}

type requestHandler struct {
	numRequestsReceived int
	countChecker        func(r *http.Request)
}

func (rh *requestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const (
		rootPath = "./testdata"
	)
	rh.numRequestsReceived++

	file, err := os.Open(path.Join(rootPath, r.URL.Path))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	defer file.Close()
	if _, err = io.Copy(w, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}
