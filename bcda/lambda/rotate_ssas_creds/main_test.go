package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfigureLogger(t *testing.T) {
	t.Run("fields are populated", func(t *testing.T) {
		entry := configureLogger("test-env", "testapp")
		require.NotNil(t, entry)
		assert.Equal(t, "test-env", entry.Data["environment"])
		assert.Equal(t, "testapp", entry.Data["application"])
	})

	t.Run("uses JSON formatter with correct settings", func(t *testing.T) {
		entry := configureLogger("dev", "bcda")
		require.NotNil(t, entry)
		formatter, ok := entry.Logger.Formatter.(*logrus.JSONFormatter)
		require.True(t, ok, "expected *logrus.JSONFormatter")
		assert.True(t, formatter.DisableHTMLEscape)
		assert.Equal(t, time.RFC3339Nano, formatter.TimestampFormat)
	})

	t.Run("ReportCaller is enabled", func(t *testing.T) {
		entry := configureLogger("prod", "bcda")
		assert.True(t, entry.Logger.ReportCaller)
	})

	t.Run("accepts empty strings", func(t *testing.T) {
		entry := configureLogger("", "")
		require.NotNil(t, entry)
		assert.Equal(t, "", entry.Data["environment"])
		assert.Equal(t, "", entry.Data["application"])
	})
}

func TestRotateCreds(t *testing.T) {
	systemID := "98"
	credsParam := "TestRotateACO"
	newCreds := shortCreds{ClientID: systemID, ClientSecret: "abc123"}
	marshalledCreds, _ := json.Marshal(newCreds)

	logger := configureLogger("test", "testapp")
	ssmClient := bcdaaws.MockSSMClient{}
	mockSSAS := &client.MockSSASHTTPClient{}
	mockSSAS.On("ResetCredentials", mock.Anything).Return(marshalledCreds, nil)
	handler := RotateSSASCredsHandler{logger: logger, ssmClient: &ssmClient, ssas: mockSSAS, slackClient: nil}

	rs := rotationSystem{CredsParam: credsParam, SystemId: systemID}
	err := handler.rotateCreds(t.Context(), rs)
	assert.Nil(t, err)

	fullCredsParamName := fmt.Sprintf("/bcda/local/creds/%s", credsParam)
	newCredsParam, err := ssmClient.GetParameter(t.Context(), &ssm.GetParameterInput{Name: &fullCredsParamName})
	assert.Nil(t, err)
	assert.Equal(t, string(marshalledCreds), *newCredsParam.Parameter.Value)

}
