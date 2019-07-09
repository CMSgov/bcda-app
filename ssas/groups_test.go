package ssas

import (
	"encoding/json"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GroupsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *GroupsTestSuite) SetupSuite() {
	s.db = GetGORMDbConnection()
	InitializeGroupModels()
}

func (s *GroupsTestSuite) TearDownSuite() {
	Close(s.db)
}

func (s *GroupsTestSuite) AfterTest() {
}

func (s *GroupsTestSuite) TestCreateGroup() {
	groupBytes := []byte(`{
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
		"systems":[  
			{  
				"client_id":"4tuhiOIFIwriIOH3zn",
				"software_id":"4NRB1-0XZABZI9E6-5SM3R",
				"client_name":"ACO System A",
				"client_uri":"https://www.acocorpsite.com"
			}
		]
	}`)

	gd := GroupData{}
	err := json.Unmarshal(groupBytes, &gd)
	assert.Nil(s.T(), err)
	g, err := CreateGroup(gd)
	assert.Nil(s.T(), err)
	s.db.Unscoped().Delete(&g)
}

func TestGroupsTestSuite(t *testing.T) {
	suite.Run(t, new(GroupsTestSuite))
}
