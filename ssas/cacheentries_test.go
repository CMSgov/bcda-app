package ssas

import (
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type CacheEntriesTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *CacheEntriesTestSuite) SetupSuite() {
	s.db = GetGORMDbConnection()
	InitializeCacheModels()
}

func (s *CacheEntriesTestSuite) TearDownSuite() {
	Close(s.db)
}

func (s *CacheEntriesTestSuite) TestGetUnexpiredCacheEntries() {
	var err error
	entryDate := time.Now().Add(time.Minute*-5).UnixNano()
	expiration := time.Now().Add(time.Minute*5).UnixNano()
	e1 := CacheEntry{Key: "key1", EntryDate: entryDate, CacheExpiration: expiration}
	e2 := CacheEntry{Key: "key2", EntryDate: entryDate, CacheExpiration: expiration}

	if err = s.db.Save(&e1).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	if err = s.db.Save(&e2).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	entries, err := GetUnexpiredCacheEntries()
	assert.Nil(s.T(), err)
	assert.Len(s.T(), entries, 2)

	err = s.db.Unscoped().Delete(&e1).Error
	assert.Nil(s.T(), err)
	err = s.db.Unscoped().Delete(&e2).Error
	assert.Nil(s.T(), err)
}

func TestCacheEntriesTestSuite(t *testing.T) {
	suite.Run(t, new(CacheEntriesTestSuite))
}
