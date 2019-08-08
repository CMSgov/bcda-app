package service

import (
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

// Using a constant for this makes the tests more readable; any arbitrary value longer than the test execution time
// should work
const expiration = 90*time.Minute

type TokenCacheTestSuite struct {
	suite.Suite
	t *Blacklist
	db *gorm.DB
}

func (s *TokenCacheTestSuite) SetupSuite() {
	ssas.InitializeBlacklistModels()
	s.db = ssas.GetGORMDbConnection()
	s.t = NewBlacklist(defaultCacheTimeout, cacheCleanupInterval)
}

func (s *TokenCacheTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *TokenCacheTestSuite) TearDownTest() {
	s.t.c.Flush()
	err := s.db.Exec("DELETE FROM blacklist_entries;").Error
	assert.Nil(s.T(), err)
}

func (s *TokenCacheTestSuite) TestLoadFromDatabaseExpired() {
	var err error
	entryDate := time.Now().Add(time.Minute*-5).Unix()
	expiration := time.Now().Add(time.Minute*-5).UnixNano()
	e1 := ssas.BlacklistEntry{Key: "key1", EntryDate: entryDate, CacheExpiration: expiration}
	e2 := ssas.BlacklistEntry{Key: "key2", EntryDate: entryDate, CacheExpiration: expiration}

	if err = s.db.Save(&e1).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	if err = s.db.Save(&e2).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	if err = s.t.LoadFromDatabase(); err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	assert.Len(s.T(), s.t.c.Items(), 0)
	assert.False(s.T(), s.t.IsTokenBlacklisted(e1.Key))
	assert.False(s.T(), s.t.IsTokenBlacklisted(e2.Key))

	err = s.db.Unscoped().Delete(&e1).Error
	assert.Nil(s.T(), err)
	err = s.db.Unscoped().Delete(&e2).Error
	assert.Nil(s.T(), err)
}

func (s *TokenCacheTestSuite) TestLoadFromDatabase() {
	var err error
	entryDate := time.Now().Add(time.Minute*-5).Unix()
	expiration := time.Now().Add(time.Minute*5).UnixNano()
	e1 := ssas.BlacklistEntry{Key: "key1", EntryDate: entryDate, CacheExpiration: expiration}
	e2 := ssas.BlacklistEntry{Key: "key2", EntryDate: entryDate, CacheExpiration: expiration}

	if err = s.db.Save(&e1).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	if err = s.db.Save(&e2).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	if err = s.t.LoadFromDatabase(); err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	assert.Len(s.T(), s.t.c.Items(), 2)
	assert.True(s.T(), s.t.IsTokenBlacklisted(e1.Key))
	assert.True(s.T(), s.t.IsTokenBlacklisted(e2.Key))

	obj1, exp1, found := s.t.c.GetWithExpiration(e1.Key)
	assert.True(s.T(), found)
	insertedDate1, ok := obj1.(int64)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), entryDate, insertedDate1)
	assert.Equal(s.T(), expiration, exp1.UnixNano())

	obj2, exp2, found := s.t.c.GetWithExpiration(e2.Key)
	assert.True(s.T(), found)
	insertedDate2, ok := obj2.(int64)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), entryDate, insertedDate2)
	assert.Equal(s.T(), expiration, exp2.UnixNano())

	err = s.db.Unscoped().Delete(&e1).Error
	assert.Nil(s.T(), err)
	err = s.db.Unscoped().Delete(&e2).Error
	assert.Nil(s.T(), err)
}

func (s *TokenCacheTestSuite) TestIsTokenBlacklistedTrue() {
	key := strconv.Itoa(rand.Int())
	err := s.t.c.Add(key, "value does not matter", expiration)
	if err != nil {
		assert.FailNow(s.T(), "unable to set cache value: " + err.Error())
	}
	assert.True(s.T(), s.t.IsTokenBlacklisted(key))
}

func (s *TokenCacheTestSuite) TestIsTokenBlacklistedExpired() {
	minimalDuration := 1*time.Nanosecond
	key := strconv.Itoa(rand.Int())
	err := s.t.c.Add(key, "value does not matter", minimalDuration)
	if err != nil {
		assert.FailNow(s.T(), "unable to set cache value: " + err.Error())
	}
	time.Sleep(minimalDuration*5)
	assert.False(s.T(), s.t.IsTokenBlacklisted(key))
}

func (s *TokenCacheTestSuite) TestIsTokenBlacklistedFalse() {
	key := strconv.Itoa(rand.Int())
	assert.False(s.T(), s.t.IsTokenBlacklisted(key))
}

func (s *TokenCacheTestSuite) TestBlacklistToken() {
	key := strconv.Itoa(rand.Int())
	if err := s.t.BlacklistToken(key, expiration); err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	_, found := s.t.c.Get(key)
	assert.True(s.T(), found)

	entries, err := ssas.GetUnexpiredBlacklistEntries()
	assert.Nil(s.T(), err)
	assert.Len(s.T(), entries, 1)
	assert.Equal(s.T(), key, entries[0].Key)
}

func (s *TokenCacheTestSuite) TestStartCacheRefreshTicker() {
	stopCacheRefreshTicker()

	var err error
	entryDate := time.Now().Add(time.Minute*-5).Unix()
	expiration := time.Now().Add(time.Minute*5).UnixNano()
	key1 := "key1"
	key2 := "key2"

	e1 := ssas.BlacklistEntry{Key: key1, EntryDate: entryDate, CacheExpiration: expiration}
	if err = s.db.Save(&e1).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	assert.False(s.T(), s.t.IsTokenBlacklisted(key1))
	assert.False(s.T(), s.t.IsTokenBlacklisted(key2))

	ticker := s.t.startCacheRefreshTicker(time.Millisecond*250)
	defer ticker.Stop()

	time.Sleep(time.Millisecond*350)
	assert.True(s.T(), s.t.IsTokenBlacklisted(key1))
	assert.False(s.T(), s.t.IsTokenBlacklisted(key2))

	e2 := ssas.BlacklistEntry{Key: key2, EntryDate: entryDate, CacheExpiration: expiration}
	if err = s.db.Save(&e2).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	time.Sleep(time.Millisecond*250)
	assert.True(s.T(), s.t.IsTokenBlacklisted(key1))
	assert.True(s.T(), s.t.IsTokenBlacklisted(key2))

	err = s.db.Unscoped().Delete(&e1).Error
	assert.Nil(s.T(), err)
	err = s.db.Unscoped().Delete(&e2).Error
	assert.Nil(s.T(), err)
}

func (s *TokenCacheTestSuite) TestBlacklistTokenKeyExists() {
	key := strconv.Itoa(rand.Int())

	// Place key in blacklist
	if err := s.t.BlacklistToken(key, expiration); err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	// Verify key exists in cache
	obj1, found := s.t.c.Get(key)
	assert.True(s.T(), found)

	// Verify key exists in database
	entries1, err := ssas.GetUnexpiredBlacklistEntries()
	assert.Nil(s.T(), err)
	assert.Len(s.T(), entries1, 1)
	assert.Equal(s.T(), key, entries1[0].Key)
	assert.Equal(s.T(), obj1, entries1[0].EntryDate)

	// The value stored is the current time expressed as in Unix time.
	// Wait to make sure the new blacklist entry has a different value
	time.Sleep(2*time.Second)

	// Place key in cache a second time; the expiration will be different
	if err := s.t.BlacklistToken(key, expiration); err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	// Verify retrieving key from cache gets new value (timestamp)
	obj2, found := s.t.c.Get(key)
	assert.True(s.T(), found)
	assert.NotEqual(s.T(), obj1, obj2)

	// Verify both keys are in the database, and that they are in time order
	entries2, err := ssas.GetUnexpiredBlacklistEntries()
	assert.Nil(s.T(), err)
	assert.Len(s.T(), entries2, 2)
	assert.Equal(s.T(), key, entries2[1].Key)
	assert.Equal(s.T(), obj2, entries2[1].EntryDate)

	// Verify that the blacklisted object changed in both cache and database
	assert.NotEqual(s.T(), obj1, obj2)
	assert.NotEqual(s.T(), entries1[0].CacheExpiration, entries2[1].CacheExpiration)

	// Show that loading the cache from the database preserves the most recent entry, even if two
	//   objects with the same key are unexpired
	err = s.t.LoadFromDatabase()
	assert.Nil(s.T(), err)
	obj3, found := s.t.c.Get(key)
	assert.True(s.T(), found)
	assert.Equal(s.T(), obj2, obj3)
	assert.NotEqual(s.T(), obj1, obj3)
}

func TestTokenCacheTestSuite(t *testing.T) {
	suite.Run(t, new(TokenCacheTestSuite))
}