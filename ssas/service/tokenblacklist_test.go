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
var (
	expiration = 90*time.Minute
	timeExpired = time.Now().Add(time.Minute*-5)
	timeNotExpired = time.Now().Add(time.Minute*5)
)

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

func (s *TokenCacheTestSuite) TestLoadFromDatabaseEmpty() {
	key := "tokenID"

	var blackListEntries []ssas.BlacklistEntry
	s.db.Unscoped().Find(&blackListEntries)
	assert.Len(s.T(), blackListEntries, 0)
	if err := s.t.LoadFromDatabase(); err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	assert.Len(s.T(), s.t.c.Items(), 0)

	if err := s.t.BlacklistToken(key, expiration); err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	s.db.Unscoped().Find(&blackListEntries)
	assert.Len(s.T(), blackListEntries, 1)
	if err := s.t.LoadFromDatabase(); err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	assert.Len(s.T(), s.t.c.Items(), 1)
}

func (s *TokenCacheTestSuite) TestLoadFromDatabaseSomeExpired() {
	expiredKey := "expiredKey"
	notExpiredKey := "notExpiredKey"
	var err error
	entryDate := timeExpired.Unix()
	expired := timeExpired.UnixNano()
	notExpired := timeNotExpired.UnixNano()
	entryExpired := ssas.BlacklistEntry{Key: expiredKey, EntryDate: entryDate, CacheExpiration: expired}
	entryDuplicateExpired := ssas.BlacklistEntry{Key: notExpiredKey, EntryDate: entryDate, CacheExpiration: expired}
	entryNotExpired := ssas.BlacklistEntry{Key: notExpiredKey, EntryDate: entryDate, CacheExpiration: notExpired}

	if err = s.db.Save(&entryExpired).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	if err = s.db.Save(&entryDuplicateExpired).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	if err = s.t.LoadFromDatabase(); err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	assert.Len(s.T(), s.t.c.Items(), 0)
	assert.False(s.T(), s.t.IsTokenBlacklisted(expiredKey))
	// This result changes after putting a new entry in the database that has not expired.
	assert.False(s.T(), s.t.IsTokenBlacklisted(notExpiredKey))

	if err = s.db.Save(&entryNotExpired).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	if err = s.t.LoadFromDatabase(); err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	assert.Len(s.T(), s.t.c.Items(), 1)
	assert.False(s.T(), s.t.IsTokenBlacklisted(expiredKey))
	// The second time we check, this key is blacklisted
	assert.True(s.T(), s.t.IsTokenBlacklisted(notExpiredKey))
}

func (s *TokenCacheTestSuite) TestLoadFromDatabase() {
	var err error
	entryDate := timeExpired.Unix()
	expiration := timeNotExpired.UnixNano()
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

	obj1, _, found := s.t.c.GetWithExpiration(e1.Key)
	assert.True(s.T(), found)
	insertedDate1, ok := obj1.(int64)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), entryDate, insertedDate1)

	obj2, _, found := s.t.c.GetWithExpiration(e2.Key)
	assert.True(s.T(), found)
	insertedDate2, ok := obj2.(int64)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), entryDate, insertedDate2)
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
	entryDate := timeExpired.Unix()
	expiration := timeNotExpired.UnixNano()
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