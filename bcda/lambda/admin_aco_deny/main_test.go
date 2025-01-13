package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestHandleACODenies(t *testing.T) {
	ctx := context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	tsAddr := ts.Listener.Addr().String()
	slackURL := "http://" + tsAddr + "/webhook"
	defer ts.Close()

	mock, err := pgxmock.NewConn()
	assert.Nil(t, err)
	defer mock.Close(ctx)

	err = handleACODenies(ctx, mock, payload{testACODenies}, slackURL)
	assert.Nil(t, err)
}
