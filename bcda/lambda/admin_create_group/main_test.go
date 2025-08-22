package main

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
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
