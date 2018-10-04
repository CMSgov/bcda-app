package main

import (
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
	"testing"
)

const BADUUID = "QWERTY-ASDFG-ZXCVBN-POIUYT"

type MainTestSuite struct {
	testUtils.AuthTestSuite
	testApp *cli.App
}

func (s *MainTestSuite) SetupTest() {
	s.testApp = setUpApp()
}

func (s *MainTestSuite) TearDownTest() {
	testUtils.PrintSeparator()
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
func (s *MainTestSuite) TestSetup() {
	assert.Equal(s.T(), 1, 1)
	app := setUpApp()
	assert.Equal(s.T(), app.Name, Name)
	assert.Equal(s.T(), app.Usage, Usage)
}

func (s *MainTestSuite) TestAutoMigrate() {
	// Plenty of other tests will rely on this happening
	// Other tests run these lines so as long as this doesn't error it sb fine
	autoMigrate()
}

func (s *MainTestSuite) TestCreateACO() {
	db := database.GetGORMDbConnection()
	//args := []string{CreateACO, "--name", "TEST_ACO"}
	//err := s.testApp.Run(args)
	//assert.Nil(s.T(), err)
	s.SetupAuthBackend()
	ACOName := "UNIT TEST ACO"
	acoUUID, err := createACO(ACOName)
	assert.NotNil(s.T(), acoUUID)
	assert.Nil(s.T(), err)
	var testACO auth.ACO
	db.First(&testACO, "Name=?", "UNIT TEST ACO")
	assert.Equal(s.T(), testACO.UUID.String(), acoUUID)

	// Might as well roll into user creation here bc otherwsise I will just be rewriting this code
	name, email := "Unit Test", "UnitTest@mail.com"
	userUUID, err := createUser(acoUUID, name, email)
	assert.NotNil(s.T(), userUUID)
	assert.Nil(s.T(), err)
	var testUser auth.User
	db.First(&testUser, "Email=?", email)
	assert.Equal(s.T(), testUser.UUID.String(), userUUID)

	// We have a user and an ACO, time for a token
	accessTokenString, err := createAccessToken(acoUUID, userUUID)
	assert.NotNil(s.T(), accessTokenString)
	assert.Nil(s.T(), err)

	// Bad/Negative tests

	// No ACO Name
	badACOName := ""
	badACO, err := createACO(badACOName)
	assert.Equal(s.T(), badACO, "")
	assert.NotNil(s.T(), err)

	// Blank UUID
	badUserUUID, err := createUser("", name, email)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badUserUUID)

	// Blank UUID
	badUserUUID, err = createUser(BADUUID, name, email)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badUserUUID)

	// Blank Name
	badUserUUID, err = createUser(acoUUID, "", email)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badUserUUID)

	// Blank E-mail address
	badUserUUID, err = createUser(acoUUID, name, "")
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badUserUUID)

	// Blank ACO UUID
	badAccessTokenString, err := createAccessToken("", userUUID)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badAccessTokenString)

	// Bad ACO UUID
	badAccessTokenString, err = createAccessToken(BADUUID, userUUID)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badAccessTokenString)

	// Blank User UUID
	badAccessTokenString, err = createAccessToken(acoUUID, "")
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badAccessTokenString)

	// Bad User UUID
	badAccessTokenString, err = createAccessToken(acoUUID, BADUUID)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badAccessTokenString)

}

func (s *MainTestSuite) TestCreateUser() {

}
