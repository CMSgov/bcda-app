package ssas

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const SampleGroup string = `{  
  "group_id":"%s",
  "name": "ACO Corp Systems",
  "resources": [
    {
      "id": "xxx",
      "name": "BCDA API",
      "scopes": [
        "bcda-api"
      ]
    },
    {
      "id": "eft",
      "name": "EFT CCLF",
      "scopes": [
        "eft-app:download",
        "eft-data:read"
      ]
    }
  ],
  "scopes": [
    "user-admin",
    "system-admin"
  ],
  "users": [
    "00uiqolo7fEFSfif70h7",
    "l0vckYyfyow4TZ0zOKek",
    "HqtEi2khroEZkH4sdIzj"
  ],
  "systems": [],
  "xdata": %s
}`

const SampleXdata string = `"{\"cms_ids\":[\"T67890\",\"T54321\"]}"`

type GroupsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *GroupsTestSuite) SetupSuite() {
	s.db = GetGORMDbConnection()
	InitializeSystemModels()
}

func (s *GroupsTestSuite) TearDownSuite() {
	Close(s.db)
}

func (s *GroupsTestSuite) AfterTest() {
}

func (s *GroupsTestSuite) TestCreateGroup() {
	gid := RandomHexID()
	gd := GroupData{}
	err := json.Unmarshal([]byte(fmt.Sprintf(SampleGroup, gid, SampleXdata)), &gd)
	assert.Nil(s.T(), err)
	g, err := CreateGroup(gd)

	require.Nil(s.T(), err)
	require.NotNil(s.T(), g)
	assert.NotZero(s.T(), g.ID)
	assert.Equal(s.T(), gid, g.GroupID)
	assert.Equal(s.T(), gid, g.Data.GroupID)
	assert.Equal(s.T(), 3, len(g.Data.Users))
	assert.NotEmpty(s.T(), g.XData)
	assert.NotEmpty(s.T(), g.Data.XData)
	assert.Equal(s.T(), g.Data.XData, g.XData)

	dbGroup := Group{}
	db := GetGORMDbConnection()
	defer Close(db)
	if db.Where("id = ?", g.ID).Find(&dbGroup).RecordNotFound() {
		assert.FailNow(s.T(), fmt.Sprintf("record not found for id=%d", g.ID))
	}
	assert.Equal(s.T(), gid, dbGroup.GroupID)
	assert.Equal(s.T(), gid, dbGroup.Data.GroupID)
	assert.Equal(s.T(), g.XData, dbGroup.XData)
	assert.Equal(s.T(), g.Data, dbGroup.Data)
	assert.Equal(s.T(), dbGroup.Data.XData, dbGroup.XData)

	err = CleanDatabase(g)
	assert.Nil(s.T(), err)
	gd.GroupID = ""
	_, err = CreateGroup(gd)
	assert.EqualError(s.T(), err, "group_id cannot be blank")
}

func (s *GroupsTestSuite) TestListGroups() {
	var startingCount int
	GetGORMDbConnection().Table("groups").Count(&startingCount)
	groupBytes := []byte(fmt.Sprintf(SampleGroup, RandomHexID(), SampleXdata))
	gd := GroupData{}
	err := json.Unmarshal(groupBytes, &gd)
	require.Nil(s.T(), err)
	g1, err := CreateGroup(gd)
	require.Nil(s.T(), err)


	gd.GroupID = RandomHexID()
	gd.Name = "some-fake-name"
	g2, err := CreateGroup(gd)
	assert.Nil(s.T(), err)

	groups, err := ListGroups("test-list-groups")
	assert.Nil(s.T(), err)
	assert.Len(s.T(), groups, 2+startingCount)

	err = CleanDatabase(g1)
	assert.Nil(s.T(), err)
	err = CleanDatabase(g2)
	assert.Nil(s.T(), err)

	groups, err = ListGroups("test-list-groups")
	assert.Nil(s.T(), err)
	assert.Len(s.T(), groups, startingCount)
}

func (s *GroupsTestSuite) TestUpdateGroup() {
	gid1 := RandomHexID()
	groupBytes := []byte(fmt.Sprintf(SampleGroup, gid1, SampleXdata))
	gd := GroupData{}
	err := json.Unmarshal(groupBytes, &gd)
	assert.Nil(s.T(), err)
	orig := Group{}
	orig.Data = gd
	err = s.db.Save(&orig).Error
	require.Nil(s.T(), err)

	gd.Scopes = []string{"aScope", "anotherScope"}
	gd.GroupID = 	RandomHexID()
	gd.Name = "aNewGroupName"
	changed, err := UpdateGroup(fmt.Sprint(orig.ID), gd)
	assert.Nil(s.T(), err)

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), []string{"aScope", "anotherScope"}, changed.Data.Scopes)
	assert.NotEqual(s.T(), "aNewGroupID", changed.Data.GroupID)
	assert.NotEqual(s.T(), "aNewGroupName", changed.Data.Name)
	err = CleanDatabase(orig)
	assert.Nil(s.T(), err)
}

func (s *GroupsTestSuite) TestDeleteGroup() {
	gid := fmt.Sprintf("delete-group-%s", RandomHexID())
	group := Group{GroupID: gid}
	err := s.db.Create(&group).Error
	require.Nil(s.T(), err, "unexpected error")

	system := System{GroupID: group.GroupID, ClientID: "groups-test-delete-client-id"}
	err = s.db.Create(&system).Error
	require.Nil(s.T(), err, "unexpected error")

	keyStr := "publickey"
	encrKey := EncryptionKey{
		SystemID: system.ID,
		Body:     keyStr,
	}
	err = s.db.Create(&encrKey).Error
	require.Nil(s.T(), err, "unexpected error")

	err = DeleteGroup(fmt.Sprint(group.ID))
	assert.Nil(s.T(), err)
	err = CleanDatabase(group)
	assert.Nil(s.T(), err)
}

func (s *GroupsTestSuite) TestGetAuthorizedGroupsForOktaID() {
	group1bytes := []byte(`{"group_id":"T0001","users":["abcdef","qrstuv"],"scopes":[],"resources":[],"systems":[],"name":""}`)
	group2bytes := []byte(`{"group_id":"T0002","users":["abcdef","qrstuv"],"scopes":[],"resources":[],"systems":[],"name":""}`)
	group3bytes := []byte(`{"group_id":"T0003","users":["qrstuv"],"scopes":[],"resources":[],"systems":[],"name":""}`)

	g1 := GroupData{}
	err := json.Unmarshal(group1bytes, &g1)
	assert.Nil(s.T(), err)
	group1, _ := CreateGroup(g1)

	g2 := GroupData{}
	err = json.Unmarshal(group2bytes, &g2)
	assert.Nil(s.T(), err)
	group2, _ := CreateGroup(g2)

	g3 := GroupData{}
	err = json.Unmarshal(group3bytes, &g3)
	assert.Nil(s.T(), err)
	group3, _ := CreateGroup(g3)

	defer s.db.Unscoped().Delete(&group1)
	defer s.db.Unscoped().Delete(&group2)
	defer s.db.Unscoped().Delete(&group3)

	authorizedGroups, err := GetAuthorizedGroupsForOktaID("abcdef")
	if err != nil {
		s.FailNow(err.Error())
	}
	if len(authorizedGroups) != 2 {
		s.FailNow("oktaID should be authorized for exactly two groups")
	}
	assert.Equal(s.T(), "T0001", authorizedGroups[0])
}

func TestGroupsTestSuite(t *testing.T) {
	suite.Run(t, new(GroupsTestSuite))
}
