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
	assert.NotNil(s.T(), g)
	assert.Equal(s.T(), "A12345", g.GroupID)
	assert.Equal(s.T(), "A12345", g.Data.ID)
	assert.Equal(s.T(), 3, len(g.Data.Users))

	dbGroup := Group{}
	db := GetGORMDbConnection()
	defer Close(db)
	if db.Where("id = ?", g.ID).Find(&dbGroup).RecordNotFound() {
		assert.FailNow(s.T(), fmt.Sprintf("record not found for id=%d", g.ID))
	}
	assert.Equal(s.T(), "A12345", dbGroup.GroupID)
	assert.Equal(s.T(), "A12345", dbGroup.Data.ID)
	assert.Equal(s.T(), 3, len(dbGroup.Data.Users))

	err = CleanDatabase(g)
	assert.Nil(s.T(), err)
	gd.ID = ""
	_, err = CreateGroup(gd)
	assert.EqualError(s.T(), err, "group_id cannot be blank")
}

func (s *GroupsTestSuite) TestListGroups() {
	groupBytes := []byte(SampleGroup)
	gd := GroupData{}
	err := json.Unmarshal(groupBytes, &gd)
	assert.Nil(s.T(), err)
	g1, err := CreateGroup(gd)
	assert.Nil(s.T(), err)

	gd.ID = "some-fake-id"
	gd.Name = "some-fake-name"
	g2, err := CreateGroup(gd)
	assert.Nil(s.T(), err)

	groups, err := ListGroups("test-list-groups")
	assert.Nil(s.T(), err)
	assert.Len(s.T(), groups, 2)

	err = CleanDatabase(g1)
	assert.Nil(s.T(), err)
	err = CleanDatabase(g2)
	assert.Nil(s.T(), err)
}

func (s *GroupsTestSuite) TestUpdateGroup() {
	groupBytes := []byte(SampleGroup)
	gd := GroupData{}
	err := json.Unmarshal(groupBytes, &gd)
	assert.Nil(s.T(), err)
	g := Group{}
	g.Data = gd
	err = s.db.Save(&g).Error
	assert.Nil(s.T(), err)

	gd.Scopes = []string{"aScope", "anotherScope"}
	gd.ID = "aNewGroupID"
	gd.Name = "aNewGroupName"
	newG, err := UpdateGroup(fmt.Sprint(g.ID), gd)
	assert.Nil(s.T(), err)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), []string{"aScope", "anotherScope"}, newG.Data.Scopes)
	assert.NotEqual(s.T(), "aNewGroupID", newG.Data.ID)
	assert.NotEqual(s.T(), "aNewGroupName", newG.Data.Name)
	err = CleanDatabase(g)
	assert.Nil(s.T(), err)
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
	err = CleanDatabase(group)
	assert.Nil(s.T(), err)

}

func TestGroupsTestSuite(t *testing.T) {
	suite.Run(t, new(GroupsTestSuite))
}
