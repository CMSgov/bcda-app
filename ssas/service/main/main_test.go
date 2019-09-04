package main

import (
	"bytes"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"io"
	"testing"
)

type MainTestSuite struct {
	suite.Suite
}

func (s *MainTestSuite) SetupSuite() {
	ssas.InitializeGroupModels()
	ssas.InitializeSystemModels()
}

func (s *MainTestSuite) TestResetCredentials() {
	addFixtureData()
	fixtureClientID := "0c527d2e-2e8a-4808-b11d-0fa06baf8254"
	output := captureOutput(func() {resetCredentials(fixtureClientID)})
	assert.NotEqual(s.T(), "", output)
}

func (s *MainTestSuite) TestMainLog() {
	var str bytes.Buffer
	ssas.Logger.SetOutput(&str)
	main()
	output := str.String()
	assert.Contains(s.T(), output, "Future home of")
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

func captureOutput(f func()) string {
	var (
		buf bytes.Buffer
		outOrig io.Writer
	)

	outOrig = output
	output = &buf
	f()
	output = outOrig
	return buf.String()
}