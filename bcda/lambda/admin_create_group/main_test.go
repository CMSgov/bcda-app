package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
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

	dbContainer, err := db.NewTestDatabaseContainer()
	require.NoError(t, err)

	defer func() {
		if err := testcontainers.TerminateContainer(dbContainer.Container); err != nil {
			t.Log(fmt.Errorf("failed to terminate container: %w", err))
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = dbContainer.ExecuteDir("testdata/")
			require.NoError(t, err)
			db, err := dbContainer.NewSqlDbConnection()
			require.NoError(t, err)
			defer func() {
				db.Close()
				err = dbContainer.RestoreSnapshot("Base")
				if err != nil {
					t.FailNow()
				}
			}()
			r := postgres.NewRepository(db) // test database
			c := &mockSSASClient{}          // mock ssas client
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

	slackName, err := setupEnv(context.Background(), &bcdaaws.MockSSMClient{})
	assert.Nil(t, err)

	assert.Equal(t, "value1", slackName)
	assert.Equal(t, "true", os.Getenv("SSAS_USE_TLS"))
	assert.Equal(t, "value3", os.Getenv("SSAS_URL"))
	assert.Equal(t, "value4", os.Getenv("BCDA_SSAS_CLIENT_ID"))
	assert.Equal(t, "value5", os.Getenv("BCDA_SSAS_SECRET"))
	assert.Equal(t, "true", os.Getenv("SSAS_USE_TLS"))
	assert.Equal(t, "/tmp/BCDA_CA_FILE.pem", os.Getenv("BCDA_CA_FILE"))
	assert.FileExists(t, "/tmp/BCDA_CA_FILE.pem")
}
