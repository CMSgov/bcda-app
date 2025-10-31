package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/stretchr/testify/assert"
)

type mockSSASClient struct {
}

func (s *mockSSASClient) CreateGroup(groupId string, name string, acoCMSID string) ([]byte, error) {
	return []byte(`{"group_id":"00001"}`), nil
}

func TestHandleCreateGroup(t *testing.T) {
	tests := []struct {
		name    string
		payload payload
		err     string
	}{
		{"valid UUID", payload{GroupID: "A8888", GroupName: "A8888-group", ACO_ID: "7b864368-bab2-4e34-9793-0b74f8b5ba46"}, ""},
		{"valid CMS ID", payload{GroupID: "A8888", GroupName: "A8888-group", ACO_ID: "A8888"}, ""},
		{"invalid aco id not found UUID", payload{GroupID: "foo", GroupName: "foo-group", ACO_ID: "a7ff9610-0977-4a90-867e-f6b2b4c8b6a"}, "ACO ID is invalid or not found"},
		{"aco id valid but not found CMS ID", payload{GroupID: "A9999", GroupName: "A9999-group", ACO_ID: "A1111"}, "no ACO record found for"},
		{"missing ACO id", payload{GroupID: "A9999", GroupName: "A9999-group", ACO_ID: ""}, "missing one or more required field(s)"},
		{"valid ID but missing required fields", payload{GroupID: "", GroupName: "A9999-group", ACO_ID: "A9999"}, "missing one or more required field(s)"},
	}

	db, _, _ := databasetest.CreateDatabase(t, "../../../db/migrations/bcda/", true)
	tf, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("testdata/"),
	)

	if err != nil {
		assert.FailNowf(t, "Failed to setup test fixtures", err.Error())
	}
	if err = tf.Load(); err != nil {
		assert.FailNowf(t, "Failed to load test fixtures", err.Error())
	}

	r := postgres.NewRepository(db) // test database
	c := &mockSSASClient{}          // mock ssas client

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = handleCreateGroup(c, r, tt.payload)
			if tt.err != "" {
				assert.Contains(t, err.Error(), tt.err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestSetupEnvironment(t *testing.T) {
	env := conf.GetEnv("ENV")

	// store env vars to restore later
	origDBURL := os.Getenv("DATABASE_URL")
	origSSASURL := os.Getenv("SSAS_URL")
	origBCDASSASClientID := os.Getenv("BCDA_SSAS_CLIENT_ID")
	origBCDASSASSecret := os.Getenv("BCDA_SSAS_SECRET")
	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	origBCDACAFile := os.Getenv("BCDA_CA_FILE")
	t.Cleanup(func() {
		// restore original env vars
		err := os.Setenv("DATABASE_URL", origDBURL)
		assert.Nil(t, err)
		err = os.Setenv("SSAS_URL", origSSASURL)
		assert.Nil(t, err)
		err = os.Setenv("BCDA_SSAS_CLIENT_ID", origBCDASSASClientID)
		assert.Nil(t, err)
		err = os.Setenv("BCDA_SSAS_SECRET", origBCDASSASSecret)
		assert.Nil(t, err)
		err = os.Setenv("SSAS_USE_TLS", origSSASUseTLS)
		assert.Nil(t, err)
		err = os.Setenv("BCDA_CA_FILE", origBCDACAFile)
		assert.Nil(t, err)
	})

	cleanupParam1 := testUtils.SetParameter(t, "/slack/token/workflow-alerts", "slack-val")
	t.Cleanup(func() { cleanupParam1() })
	cleanupParam2 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/DATABASE_URL", env), "test-DB_URL")
	t.Cleanup(func() { cleanupParam2() })
	cleanupParam3 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/SSAS_URL", env), "test-SSAS_URL")
	t.Cleanup(func() { cleanupParam3() })
	cleanupParam4 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_CLIENT_ID", env), "test-BCDA_SSAS_CLIENT_ID")
	t.Cleanup(func() { cleanupParam4() })
	cleanupParam5 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/BCDA_SSAS_SECRET", env), "test-BCDA_SSAS_SECRET")
	t.Cleanup(func() { cleanupParam5() })
	cleanupParam6 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/api/BCDA_CA_FILE.pem", env), "test-BCDA_CA_FILE")
	t.Cleanup(func() { cleanupParam6() })

	slackName, err := setupEnv(context.Background())
	assert.Nil(t, err)

	assert.Equal(t, "slack-val", slackName)
	assert.Equal(t, "true", os.Getenv("SSAS_USE_TLS"))
	assert.Equal(t, "test-SSAS_URL", os.Getenv("SSAS_URL"))
	assert.Equal(t, "test-BCDA_SSAS_CLIENT_ID", os.Getenv("BCDA_SSAS_CLIENT_ID"))
	assert.Equal(t, "test-BCDA_SSAS_SECRET", os.Getenv("BCDA_SSAS_SECRET"))
	assert.Equal(t, "true", os.Getenv("SSAS_USE_TLS"))
	assert.Equal(t, "/tmp/BCDA_CA_FILE.pem", os.Getenv("BCDA_CA_FILE"))
	assert.FileExists(t, "/tmp/BCDA_CA_FILE.pem")
}
