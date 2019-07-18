package ssas

import (
	"encoding/json"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const SampleGroup string = `{  
	"id":"A12345",
	"name":"ACO Corp Systems",
	"users":[  
		"00uiqolo7fEFSfif70h7",
		"l0vckYyfyow4TZ0zOKek",
		"HqtEi2khroEZkH4sdIzj"
	],
	"scopes":[  
		"user-admin",
		"system-admin"
	],
	"resources":[  
		{  
			"id":"xxx",
			"name":"BCDA API",
			"scopes":[  
				"bcda-api"
			]
		},
		{  
			"id":"eft",
			"name":"EFT CCLF",
			"scopes":[  
				"eft-app:download",
				"eft-data:read"
			]
		}
	],
	"system":
		{  
		"client_id":"4tuhiOIFIwriIOH3zn",
		"software_id":"4NRB1-0XZABZI9E6-5SM3R",
		"client_name":"ACO System A"
		}
}`

type GroupsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *GroupsTestSuite) SetupSuite() {
	s.db = GetGORMDbConnection()
	InitializeGroupModels()
	InitializeSystemModels()
}

func (s *GroupsTestSuite) TearDownSuite() {
	Close(s.db)
}

func (s *GroupsTestSuite) AfterTest() {
}

func (s *GroupsTestSuite) TestCreateGroup() {
	groupBytes := []byte(SampleGroup)
	gd := GroupData{}
	err := json.Unmarshal(groupBytes, &gd)
	assert.Nil(s.T(), err)
	g, err := CreateGroup(gd)
	assert.Nil(s.T(), err)
	err = CleanDatabase(g)
	assert.Nil(s.T(), err)
	gd.ID = ""
	_, err = CreateGroup(gd)
	assert.EqualError(s.T(), err, "group_id cannot be blank")
}

func (s *GroupsTestSuite) TestUpdateGroup() {
	groupBytes := []byte(SampleGroup)
	gd := GroupData{}
	err := json.Unmarshal(groupBytes, &gd)
	assert.Nil(s.T(), err)
	g, err := CreateGroup(gd)
	assert.Nil(s.T(), err)

	gd.Scopes = []string{"aScope", "anotherScope"}
	gd.ID = "aNewGroupID"
	gd.Name = "aNewGroupName"
	newG, err := UpdateGroup(string(g.ID), gd)
	assert.Nil(s.T(), err)

	newGDBytes, _ := newG.Data.MarshalJSON()
	newGD := GroupData{}
	err = newGD.Scan(newGDBytes)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), []string{"aScope", "anotherScope"}, newGD.Scopes)
	assert.NotEqual(s.T(), "aNewGroupID", newGD.ID)
	assert.NotEqual(s.T(), "aNewGroupName", newGD.Name)
	err = CleanDatabase(g)
	assert.Nil(s.T(), err)
}

func TestGroupsTestSuite(t *testing.T) {
	suite.Run(t, new(GroupsTestSuite))
}
