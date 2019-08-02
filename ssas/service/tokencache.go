package service

import (
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/patrickmn/go-cache"
	"time"
)

var (
	Cache                TokenCache
	// This default cache timeout value should never be used, since individual cache elements have their own timeouts
	defaultCacheTimeout   = 24*time.Hour
	// TODO: set the cacheCleanupInterval from an env var
	cacheCleanupInterval  = 30*time.Minute
)

func init() {
	NewTokenCache(defaultCacheTimeout, cacheCleanupInterval)
}

//	NewTokenCache allows for easy TokenCache{} creation and manipulation during testing, and should not be called
//	outside a test suite
func NewTokenCache(cacheTimeout time.Duration, cleanupInterval time.Duration) *TokenCache {
	tc := TokenCache{}
	tc.c = cache.New(cacheTimeout, cleanupInterval)
	return &tc
}

type TokenCache struct {
	c *cache.Cache
}

//	BlacklistOrgTokens invalidates all tokens generated for the specified organization before this time
func (t *TokenCache) BlacklistOrgTokens(orgKey string, tokenID string, blacklistExpiration time.Duration) {
	ssas.TokenBlacklisted(ssas.Event{Op: "OrgTokenBlacklist", TrackingID: orgKey, TokenID: tokenID})
	// TODO: save org/date to database
	t.c.Set(orgKey, time.Now().Unix(), blacklistExpiration)
}

//	IsOrgTokenBlacklisted tests whether this organization has invalidated tokens created at this time
func (t *TokenCache) IsOrgTokenBlacklisted(orgKey string, tokenUnixDate int64) bool {
	bEvent := ssas.Event{Op: "TokenVerification", TrackingID: orgKey}
	if allClearDate, found := t.c.Get(orgKey); found && beforeBlacklisted(orgKey, allClearDate, tokenUnixDate) {
		ssas.BlacklistedTokenPresented(bEvent)
		return true
	}
	return false
}

//	BlacklistToken invalidates the specified tokenID
func (t *TokenCache) BlacklistToken(tokenID string, blacklistExpiration time.Duration) {
	ssas.TokenBlacklisted(ssas.Event{Op: "TokenBlacklist", TrackingID: tokenID, TokenID: tokenID})
	// TODO: save tokenID to database
	t.c.Set(tokenID, time.Now().Unix(), blacklistExpiration)
}

//	IsTokenBlacklisted tests whether this tokenID has been invalidated
func (t *TokenCache) IsTokenBlacklisted(tokenID string) bool {
	bEvent := ssas.Event{Op: "TokenVerification", TrackingID: tokenID, TokenID: tokenID}
	if _, found := t.c.Get(tokenID); found {
		ssas.BlacklistedTokenPresented(bEvent)
		return true
	}
	return false
}

//	LoadFromDatabase refreshes the cache contents from the database
func (t *TokenCache) LoadFromDatabase() error {
	// TODO: make this work
	return nil
}

func beforeBlacklisted(key string, timeObj interface{}, tokenSignUnixDate int64) bool {
	var (
		allClearUnixDate int64
		ok bool
	)

	if allClearUnixDate, ok = timeObj.(int64); !ok {
		// When in doubt, assume an unreadable timestamp invalidates all tokens for this org
		ssas.BlacklistedTokenPresented(ssas.Event{Op: "TestIfInBlacklistSigningPeriod",
			Help: fmt.Sprintf("unable to parse all clear date for key %s", key)})
		return true
	}
	return tokenSignUnixDate < allClearUnixDate
}

// TODO: write CLI command to call BlacklistOrgToken()
// TODO: write CLI command to call LoadFromDatabase()
// TODO: write cron job to call CLI command for LoadFromDatabase() every five minutes