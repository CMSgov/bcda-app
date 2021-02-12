package errors_test

import (
	"fmt"
	"net/http"
	"testing"

	autherrors "github.com/CMSgov/bcda-app/bcda/auth/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ProviderErrorsTestSuite struct {
	suite.Suite
}

func (s *ProviderErrorsTestSuite) TestProviderErrorReturn() {
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
		s.T().Run(fmt.Sprintf("%d", tt.respCode), func(t *testing.T) {
			err := &autherrors.ProviderError{Code: tt.respCode, Message: tt.message}
			assert.Contains(t, err.Error(), fmt.Sprintf("%d - %s", tt.expCode, tt.message))
			if pe, ok := err.(*autherrors.ProviderError); ok {
				assert.Equal(t, tt.expCode, pe.Code)
				assert.Equal(t, tt.message, pe.Message)
			}
		})
	}
}

func TestProviderErrorsTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderErrorsTestSuite))
}
