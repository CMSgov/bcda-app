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
	group := payload{
		GroupID:   "A9999",
		GroupName: "A9999-group",
		ACO_ID:    "A9999",
	}

	db, _ := databasetest.CreateDatabase(t, "../../../db/migrations/bcda/", true)
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
	c := &mockSSASClient{}          // new client

	err = handleCreateGroup(c, r, group)
	assert.Nil(t, err)
}
