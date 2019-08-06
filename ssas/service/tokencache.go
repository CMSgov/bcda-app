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

//	BlacklistToken invalidates the specified tokenID
func (t *TokenCache) BlacklistToken(tokenID string, blacklistExpiration time.Duration) error {
	entryDate := time.Now()
	expirationDate := entryDate.Add(blacklistExpiration)
	if _, err := ssas.CreateCacheEntry(tokenID, entryDate, expirationDate); err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to blacklist token id %s: %s", tokenID, err.Error()))
	}

	ssas.TokenBlacklisted(ssas.Event{Op: "TokenBlacklist", TrackingID: tokenID, TokenID: tokenID})
	t.c.Set(tokenID, entryDate.Unix(), blacklistExpiration)

	return nil
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

//	LoadFromDatabase refreshes unexpired cache contents from the database
func (t *TokenCache) LoadFromDatabase() error {
	var (
		entries	[]ssas.CacheEntry
		items	map[string]cache.Item
		err		error
	)

	if entries, err = ssas.GetUnexpiredCacheEntries(); err != nil {
		return err
	}

	t.c.Flush()
	items = make(map[string]cache.Item)
	for _, entry := range entries {
		expDuration := entry.CacheExpiration
		item := cache.Item{Object: entry.EntryDate, Expiration: expDuration}
		items[entry.Key] = item
	}

	t.c = cache.NewFrom(defaultCacheTimeout, cacheCleanupInterval, items)
	return nil
}

// TODO: write CLI command to call BlacklistToken()
// TODO: write CLI command to call LoadFromDatabase()
// TODO: write cron job to call CLI command for LoadFromDatabase() every five minutes