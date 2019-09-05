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
	TokenBlacklist Blacklist
	// This default cache timeout value should never be used, since individual cache elements have their own timeouts
	defaultCacheTimeout   = 24*time.Hour
	cacheCleanupInterval  time.Duration
	TokenCacheLifetime	  time.Duration
	cacheRefreshFreq	  time.Duration
	cacheRefreshTicker	  *time.Ticker
)

func init() {
	cacheCleanupInterval = time.Duration(cfg.GetEnvInt("SSAS_TOKEN_BLACKLIST_CACHE_CLEANUP_MINUTES", 15)) * time.Minute
	TokenCacheLifetime	 = time.Duration(cfg.GetEnvInt("SSAS_TOKEN_BLACKLIST_CACHE_TIMEOUT_MINUTES", 60*24)) * time.Minute
	cacheRefreshFreq	 = time.Duration(cfg.GetEnvInt("SSAS_TOKEN_BLACKLIST_CACHE_REFRESH_MINUTES", 5)) * time.Minute
}

// This function should only be called by main
func StartBlacklist() {
	NewBlacklist(defaultCacheTimeout, cacheCleanupInterval)
}

//	NewBlacklist allows for easy Blacklist{} creation and manipulation during testing, and, outside a test suite,
//	should not be called
func NewBlacklist(cacheTimeout time.Duration, cleanupInterval time.Duration) *Blacklist {
	// In case a Blacklist timer has already been started:
	stopCacheRefreshTicker()

	trackingID := uuid.NewRandom().String()
	event := ssas.Event{Op: "InitBlacklist", TrackingID: trackingID}
	ssas.OperationStarted(event)

	bl := Blacklist{ID: trackingID}
	bl.c = cache.New(cacheTimeout, cleanupInterval)

	if err := bl.LoadFromDatabase(); err != nil {
		event.Help = "unable to load blacklist from database: " + err.Error()
		ssas.OperationFailed(event)
		// Log this failure, but allow the cache to operate.  It's conceivable the next cache refresh will work.
		// Any alerts should be based on the error logged in BlackList.LoadFromDatabase().
	} else
	{
		ssas.OperationSucceeded(event)
	}

	cacheRefreshTicker = bl.startCacheRefreshTicker(cacheRefreshFreq)

	TokenBlacklist = bl
	return &bl
}

type Blacklist struct {
	c 	*cache.Cache
	ID	string
}

//	BlacklistToken invalidates the specified tokenID
func (t *Blacklist) BlacklistToken(tokenID string, blacklistExpiration time.Duration) error {
	entryDate := time.Now()
	expirationDate := entryDate.Add(blacklistExpiration)
	if _, err := ssas.CreateBlacklistEntry(tokenID, entryDate, expirationDate); err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to blacklist token id %s: %s", tokenID, err.Error()))
	}

	// Add to cache only after token is blacklisted in database
	ssas.TokenBlacklisted(ssas.Event{Op: "TokenBlacklist", TrackingID: tokenID, TokenID: tokenID})
	t.c.Set(tokenID, entryDate.Unix(), blacklistExpiration)

	return nil
}

//	IsTokenBlacklisted tests whether this tokenID is in the blacklist cache.
//	- Tokens should expire before blacklist entries, so a tokenID for a recently expired token may return "true."
//	- This queries the cache only, so if a tokenID has been blacklisted on a different instance, it will return "false"
//		until the cached blacklist is refreshed from the database.
func (t *Blacklist) IsTokenBlacklisted(tokenID string) bool {
	bEvent := ssas.Event{Op: "TokenVerification", TrackingID: t.ID, TokenID: tokenID}
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
		err		error
	)

	if entries, err = ssas.GetUnexpiredBlacklistEntries(); err != nil {
		ssas.CacheSyncFailure(ssas.Event{Op: "BlacklistLoadFromDatabase", TrackingID: t.ID, Help: err.Error()})
		return err
	}

	t.c.Flush()

	// If the key already exists in the cache, it will be updated.
	for _, entry := range entries {
		cacheDuration := time.Since(time.Unix(0, entry.CacheExpiration))
		t.c.Set(entry.Key, entry.EntryDate, cacheDuration)
	}
	return nil
}

func (t *Blacklist) startCacheRefreshTicker(refreshFreq time.Duration) *time.Ticker {
	event := ssas.Event{Op: "CacheRefreshTicker", TrackingID: t.ID}
	ssas.ServiceStarted(event)

	ticker := time.NewTicker(refreshFreq)

	go func() {
		for range ticker.C {
			// Errors are logged in LoadFromDatabase()
			_ = t.LoadFromDatabase()
		}
	}()

	return ticker
}

func stopCacheRefreshTicker() {
	if cacheRefreshTicker != nil {
		cacheRefreshTicker.Stop()
	}
}