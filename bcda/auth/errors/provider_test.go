package errors_test

import (
	"fmt"
	"net/http"
	"testing"

	autherrors "github.com/CMSgov/bcda-app/bcda/auth/errors"
	"github.com/stretchr/testify/assert"
)

func TestProviderErrorReturn(t *testing.T) {
	tests := []struct {
		respCode int
		expCode  int
		message  string
	}{
		{http.StatusBadRequest, 400, "bad request"},
		{http.StatusUnauthorized, 401, "unauthorized"},
		{http.StatusNotFound, 404, "not found"},
		{http.StatusInternalServerError, 500, "internal server error"},
	}

	for _, tt := range tests {
		err := &autherrors.ProviderError{Code: tt.respCode, Message: tt.message}
		assert.Contains(t, err.Error(), fmt.Sprintf("%d - %s", tt.expCode, tt.message))
		assert.Equal(t, tt.expCode, err.Code)
		assert.Equal(t, tt.message, err.Message)
	}
}
