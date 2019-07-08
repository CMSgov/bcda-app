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


func (s *GroupsTestSuite) TestGroupModel() {
groupBytes := []byte(`{  
		"group_id":"A12345",
		"data":{  
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
		}
	}`)

group := Group{}
err := json.Unmarshal(groupBytes, &group)
assert.Nil(s.T(), err)
db := GetGORMDbConnection()
defer Close(db)
err = db.Save(&group).Error
assert.Nil(s.T(), err)
db.Unscoped().Delete(&group)
}

func TestGroupsTestSuite(t *testing.T) {
	suite.Run(t, new(GroupsTestSuite))
}
