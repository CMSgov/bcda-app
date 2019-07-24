package ssas

import (
	"encoding/json"
	"fmt"
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
	_ = CleanDatabase(g)
	gd.ID = ""
	_, err = CreateGroup(gd)
	assert.EqualError(s.T(), err, "group_id cannot be blank")
}

func (s *GroupsTestSuite) TestDeleteGroup() {
	group := Group{GroupID: "groups-test-delete-group-id"}
	err := s.db.Create(&group).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	system := System{GroupID: group.GroupID, ClientID: "groups-test-delete-client-id"}
	err = s.db.Create(&system).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	keyStr := "publickey"
	encrKey := EncryptionKey{
		SystemID: system.ID,
		Body:     keyStr,
	}
	err = s.db.Create(&encrKey).Error
	if err != nil {
		s.FailNow(err.Error())
	}

	err = DeleteGroup(fmt.Sprint(group.ID))
	assert.Nil(s.T(), err)
}

func TestGroupsTestSuite(t *testing.T) {
	suite.Run(t, new(GroupsTestSuite))
}
