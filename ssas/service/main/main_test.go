package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/ssas"
)

type MainTestSuite struct {
	suite.Suite
}

func (s *MainTestSuite) SetupSuite() {
	ssas.InitializeGroupModels()
	ssas.InitializeSystemModels()
	addFixtureData()
}

func (s *MainTestSuite) TestResetSecret() {
	fixtureClientID := "0c527d2e-2e8a-4808-b11d-0fa06baf8254"
	output := captureOutput(func() { resetSecret(fixtureClientID) })
	assert.NotEqual(s.T(), "", output)
}

func (s *MainTestSuite) TestNewAdminSystem() {
	output := captureOutput(func() { newAdminSystem("Main Test System") })
	assert.NotEqual(s.T(), "", output)
}

func (s *MainTestSuite) TestMainLog() {
	var str bytes.Buffer
	ssas.Logger.SetOutput(&str)
	main()
	output := str.String()
	assert.Contains(s.T(), output, "Home of")
}

func (s *MainTestSuite) TestAddFixtureData() {
	q := `select distinct g.id as gid, g.group_id, s.id as sid, s.client_name, ek.id as ekid, s.id as scrtid
	from groups g
	join systems s on g.group_id = s.group_id
	join encryption_keys ek on ek.system_id = s.id
	join secrets sc on sc.system_id = s.id
	where g.group_id in ('admin', '0c527d2e-2e8a-4808-b11d-0fa06baf8254');`
	// if you run the query above against the db, you will see a result like this:
	// gid |               group_id               | sid |  client_name   | ekid | scrtid
	// -----+--------------------------------------+-----+----------------+------+--------
	// 15 | admin                                |  13 | BCDA API Admin |   13 |     13
	// 16 | 0c527d2e-2e8a-4808-b11d-0fa06baf8254 |  14 | ACO Dev        |   14 |     14
	// (2 rows)
	//
	// only complete fixture data will be included in the result

	type result struct {
		GID        uint		`json:"gid"`
		GroupID    string	`json:"group_id"`
		SID        uint		`json:"sid"`
		ClientName string	`json:"client_name"`
		EKID       uint		`json:"ekid"`
		ScrtID     uint		`json:"scrtid"`
	}
	rows, err := ssas.GetGORMDbConnection().Raw(q).Rows()
	require.Nil(s.T(), err, "error checking fixture data")
	defer rows.Close()

	foundAdmin := false
	foundConsumer := false
	for rows.Next() {
		var r result
		err := rows.Scan(&r.GID, &r.GroupID, &r.SID, &r.ClientName, &r.EKID, &r.ScrtID)
		require.Nil(s.T(), err, "error scanning data")
		switch r.GroupID {
		case "admin":
			foundAdmin = true
		case "0c527d2e-2e8a-4808-b11d-0fa06baf8254":
			foundConsumer = true
		}
	}

	assert.True(s.T(), foundAdmin)
	assert.True(s.T(), foundConsumer)
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

func captureOutput(f func()) string {
	var (
		buf     bytes.Buffer
		outOrig io.Writer
	)

	outOrig = output
	output = &buf
	f()
	output = outOrig
	return buf.String()
}
