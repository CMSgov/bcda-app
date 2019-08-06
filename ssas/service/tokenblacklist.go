package service

import (
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/cfg"
	"github.com/patrickmn/go-cache"
	"github.com/pborman/uuid"
	"time"
)

var (
	Cache Blacklist
	// This default cache timeout value should never be used, since individual cache elements have their own timeouts
	defaultCacheTimeout   = 24*time.Hour
	cacheCleanupInterval  time.Duration
)

func init() {
	cacheCleanupInterval = time.Duration(cfg.GetEnvInt("SSAS_TOKEN_BLACKLIST_CACHE_EXP_MINUTES", 15)) * time.Minute
	NewBlacklist(defaultCacheTimeout, cacheCleanupInterval)
}

//	NewBlacklist allows for easy Blacklist{} creation and manipulation during testing, and should not be called
//		outside a test suite
func NewBlacklist(cacheTimeout time.Duration, cleanupInterval time.Duration) *Blacklist {
	trackingID := uuid.NewRandom().String()
	event := ssas.Event{Op: "InitBlacklist", TrackingID: trackingID}
	ssas.OperationStarted(event)

	tc := Blacklist{}
	tc.c = cache.New(cacheTimeout, cleanupInterval)

	if err := tc.LoadFromDatabase(); err != nil {
		event.Help = "unable to load blacklist from database: " + err.Error()
		ssas.OperationFailed(event)
	}

	ssas.OperationSucceeded(event)
	return &tc
}

type Blacklist struct {
	c *cache.Cache
}

//	BlacklistToken invalidates the specified tokenID
func (t *Blacklist) BlacklistToken(tokenID string, blacklistExpiration time.Duration) error {
	entryDate := time.Now()
	expirationDate := entryDate.Add(blacklistExpiration)
	if _, err := ssas.CreateBlacklistEntry(tokenID, entryDate, expirationDate); err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to blacklist token id %s: %s", tokenID, err.Error()))
	}

	ssas.TokenBlacklisted(ssas.Event{Op: "TokenBlacklist", TrackingID: tokenID, TokenID: tokenID})
	t.c.Set(tokenID, entryDate.Unix(), blacklistExpiration)

	return nil
}

//	IsTokenBlacklisted tests whether this tokenID is in the blacklist cache.
//	- Tokens should expire before blacklist entries, so a tokenID for a recently expired token may return "true."
//	- This tests the cache only, so if a tokenID has been blacklisted on a different instance, it will return "false"
//		until the cache is refreshed.
func (t *Blacklist) IsTokenBlacklisted(tokenID string) bool {
	bEvent := ssas.Event{Op: "TokenVerification", TrackingID: tokenID, TokenID: tokenID}
	if _, found := t.c.Get(tokenID); found {
		ssas.BlacklistedTokenPresented(bEvent)
		return true
	}
	return false
}

//	LoadFromDatabase refreshes unexpired blacklist entries from the database
func (t *Blacklist) LoadFromDatabase() error {
	var (
		entries	[]ssas.BlacklistEntry
		items	map[string]cache.Item
		err		error
	)

	if entries, err = ssas.GetUnexpiredBlacklistEntries(); err != nil {
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

// TODO: write an admin endpoint to call BlacklistToken()
// TODO: implement RevokeAccessToken() in the BCDA Auth SSAS provider to call the new admin endpoint
// TODO: implement a timer to call LoadFromDatabase() periodically (every five minutes?)
