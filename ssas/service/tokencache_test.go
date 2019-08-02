package service

import (
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
	t *TokenCache
}

func (s *TokenCacheTestSuite) SetupSuite() {
	s.t = NewTokenCache(defaultCacheTimeout, cacheCleanupInterval)
}

func (s *TokenCacheTestSuite) SetupTest() {
	s.t.c.Flush()
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
	assert.False(s.T(), s.t.IsTokenBlacklisted(key))
}

func (s *TokenCacheTestSuite) TestIsTokenBlacklistedFalse() {
	key := strconv.Itoa(rand.Int())
	assert.False(s.T(), s.t.IsTokenBlacklisted(key))
}

func (s *TokenCacheTestSuite) TestBlacklistToken() {
	key := strconv.Itoa(rand.Int())
	s.t.BlacklistToken(key, expiration)

	_, found := s.t.c.Get(key)
	assert.True(s.T(), found)

	// TODO: verify token is in database
}

func (s *TokenCacheTestSuite) TestBlacklistTokenKeyExists() {
	key := strconv.Itoa(rand.Int())

	s.t.BlacklistToken(key, expiration)
	obj1, found := s.t.c.Get(key)
	assert.True(s.T(), found)
	// TODO: verify token is in database

	// The value stored is the current time expressed as in Unix time.  Since the precision is one second,
	// wait to make sure the new blacklist entry has a different value
	time.Sleep(1*time.Second)

	s.t.BlacklistToken(key, expiration)
	obj2, found := s.t.c.Get(key)
	assert.True(s.T(), found)
	// TODO: verify token is in database with new value
	assert.NotEqual(s.T(), obj1, obj2)
}

func (s *TokenCacheTestSuite) TestBlacklistOrgTokens() {
	orgKey := "T0001"
	tokenID := strconv.Itoa(rand.Int())

	s.t.BlacklistOrgTokens(orgKey, tokenID, expiration)

	_, found := s.t.c.Get(orgKey)
	assert.True(s.T(), found)

	// TODO: verify token is in database
}

func (s *TokenCacheTestSuite) TestBlacklistOrgTokensKeyExists() {
	orgKey := "T0001"
	tokenID := strconv.Itoa(rand.Int())

	s.t.BlacklistOrgTokens(orgKey, tokenID, expiration)
	obj1, found := s.t.c.Get(orgKey)
	assert.True(s.T(), found)
	// TODO: verify token is in database

	time.Sleep(1*time.Second)

	s.t.BlacklistOrgTokens(orgKey, tokenID, expiration)
	obj2, found := s.t.c.Get(orgKey)
	assert.True(s.T(), found)
	// TODO: verify token is in database with new value
	assert.NotEqual(s.T(), obj1, obj2)
}

func (s *TokenCacheTestSuite) TestIsOrgTokenBlacklistedTrue() {
	blacklistTime := time.Now().Unix()
	tokenTime := time.Now().Add(-5*time.Minute).Unix()

	key := strconv.Itoa(rand.Int())
	err := s.t.c.Add(key, blacklistTime, expiration)
	if err != nil {
		assert.FailNow(s.T(), "unable to set cache value: " + err.Error())
	}
	assert.True(s.T(), s.t.IsOrgTokenBlacklisted(key, tokenTime))
}

func (s *TokenCacheTestSuite) TestIsOrgTokenBlacklistedNotInCache() {
	tokenTime := time.Now().Add(-5*time.Minute).Unix()
	key := strconv.Itoa(rand.Int())
	assert.False(s.T(), s.t.IsOrgTokenBlacklisted(key, tokenTime))
}

func (s *TokenCacheTestSuite) TestIsOrgTokenBlacklistedCacheExpired() {
	blacklistTime := time.Now().Unix()
	tokenTime := time.Now().Add(-5*time.Minute).Unix()

	minimalDuration := 1*time.Nanosecond
	key := strconv.Itoa(rand.Int())
	err := s.t.c.Add(key, blacklistTime, minimalDuration)
	if err != nil {
		assert.FailNow(s.T(), "unable to set cache value: " + err.Error())
	}
	assert.False(s.T(), s.t.IsOrgTokenBlacklisted(key, tokenTime))
}

func (s *TokenCacheTestSuite) TestIsOrgTokenBlacklistedAfterBlacklistDate() {
	blacklistTime := time.Now().Unix()
	tokenTime := time.Now().Add(5*time.Minute).Unix()

	key := strconv.Itoa(rand.Int())
	err := s.t.c.Add(key, blacklistTime, expiration)
	if err != nil {
		assert.FailNow(s.T(), "unable to set cache value: " + err.Error())
	}
	assert.False(s.T(), s.t.IsOrgTokenBlacklisted(key, tokenTime))
}

func (s *TokenCacheTestSuite) TestBeforeBlacklistedTrue() {
	// Testing with a blacklisting date in the past to avoid any relation with the current time
	blacklistTime := time.Now().Add(-30*24*time.Hour)
	blacklistUnixTime := blacklistTime.Unix()
	earlierUnixTime := blacklistTime.Add(-5*time.Minute).Unix()

	isBlacklisted := beforeBlacklisted("key_only_for_logging", blacklistUnixTime, earlierUnixTime)
	assert.True(s.T(), isBlacklisted)
}

func (s *TokenCacheTestSuite) TestBeforeBlacklistedFalse() {
	blacklistTime := time.Now().Add(-30*24*time.Hour)
	blacklistUnixTime := blacklistTime.Unix()
	laterUnixTime := blacklistTime.Add(5*time.Minute).Unix()

	isBlacklisted := beforeBlacklisted("key_only_for_logging", blacklistUnixTime, laterUnixTime)
	assert.False(s.T(), isBlacklisted)
}

func (s *TokenCacheTestSuite) TestBeforeBlacklistedError() {
	blacklistTime := time.Now().Add(-30*24*time.Hour)
	laterUnixTime := blacklistTime.Add(5*time.Minute).Unix()

	isBlacklisted := beforeBlacklisted("key_only_for_logging", "this is not a time!", laterUnixTime)
	assert.True(s.T(), isBlacklisted)
}

func TestTokenCacheTestSuite(t *testing.T) {
	suite.Run(t, new(TokenCacheTestSuite))
}