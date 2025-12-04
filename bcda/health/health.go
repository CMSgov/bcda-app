package health

import (
	"context"
	"database/sql"
	"sync"
	"time"

	backoff "github.com/cenkalti/backoff/v4"

	ssasClient "github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/client"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/middleware"
)

type introspectCache struct {
	result    string
	ok        bool
	timestamp time.Time
	mu        sync.RWMutex
}

type HealthChecker struct {
	db              *sql.DB
	introspectCache *introspectCache
}

const (
	introspectCacheTTL     = 5 * time.Minute
	introspectRetryCount   = 3
	introspectRetryInitial = 100 * time.Millisecond
)

func NewHealthChecker(db *sql.DB) HealthChecker {
	return HealthChecker{
		db:              db,
		introspectCache: &introspectCache{},
	}
}

func (h HealthChecker) IsDatabaseOK() (result string, ok bool) {
	if err := h.db.Ping(); err != nil {
		log.API.Error("Health check: database ping error: ", err.Error())
		return "database ping error", false
	}

	return "ok", true
}

func (h HealthChecker) IsWorkerDatabaseOK() (result string, ok bool) {
	if err := h.db.Ping(); err != nil {
		log.Worker.Error("Health check: database ping error: ", err.Error())
		return "database ping error", false
	}

	return "ok", true
}

func (h HealthChecker) IsBlueButtonOK() bool {
	bbc, err := client.NewBlueButtonClient(client.NewConfig("/v1/fhir"))
	if err != nil {
		log.Worker.Error("Health check: Blue Button client error: ", err.Error())
		return false
	}

	_, err = bbc.GetMetadata()
	if err != nil {
		log.Worker.Error("Health check: Blue Button connection error: ", err.Error())
		return false
	}

	return true
}

func (h HealthChecker) IsSsasOK() (result string, ok bool) {
	c, err := ssasClient.NewSSASClient()
	if err != nil {
		log.Auth.Errorf("no client for SSAS. no provider set; %s", err.Error())
		return "No client for SSAS. no provider set", false
	}
	if err := c.GetHealth(); err != nil {
		log.API.Error("Health check: ssas health check error: ", err.Error())
		return "Cannot connect to SSAS", false
	}
	return "ok", true
}

func (h HealthChecker) IsSsasIntrospectOK() (result string, ok bool) {
	// Check cache first
	h.introspectCache.mu.RLock()
	if h.introspectCache.timestamp.Add(introspectCacheTTL).After(time.Now()) {
		result := h.introspectCache.result
		ok := h.introspectCache.ok
		h.introspectCache.mu.RUnlock()
		return result, ok
	}
	h.introspectCache.mu.RUnlock()

	// Cache expired or missing, perform actual check
	h.introspectCache.mu.Lock()
	defer h.introspectCache.mu.Unlock()

	// Double-check after acquiring write lock
	if h.introspectCache.timestamp.Add(introspectCacheTTL).After(time.Now()) {
		return h.introspectCache.result, h.introspectCache.ok
	}

	// Get BCDA admin credentials
	clientID := conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	clientSecret := conf.GetEnv("BCDA_SSAS_SECRET")
	if clientID == "" || clientSecret == "" {
		result := "BCDA admin credentials not configured"
		h.introspectCache.result = result
		h.introspectCache.ok = false
		h.introspectCache.timestamp = time.Now()
		log.API.Error("Health check: SSAS introspect - missing BCDA admin credentials")
		return result, false
	}

	// Create SSAS client
	c, err := ssasClient.NewSSASClient()
	if err != nil {
		result := "Failed to create SSAS client"
		h.introspectCache.result = result
		h.introspectCache.ok = false
		h.introspectCache.timestamp = time.Now()
		log.API.Error("Health check: SSAS introspect - failed to create client: ", err.Error())
		return result, false
	}

	// This verifies SSAS is up and credentials are valid without requiring token generation
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.CtxTransactionKey, "health-check")

	// Call introspect endpoint with retry
	_, err = h.introspectWithRetry(c, ctx, "None")
	if err != nil {
		result := "SSAS introspect check failed"
		h.introspectCache.result = result
		h.introspectCache.ok = false
		h.introspectCache.timestamp = time.Now()
		log.API.Error("Health check: SSAS introspect - introspect call failed after retries: ", err.Error())
		return result, false
	}

	// Success - update cache
	h.introspectCache.result = "ok"
	h.introspectCache.ok = true
	h.introspectCache.timestamp = time.Now()
	return "ok", true
}

// isRetryableError determines if an error should be retried
func isRetryableError(err error) bool {
	switch e := err.(type) {
	case *customErrors.RequestTimeoutError:
		return true
	case *customErrors.UnexpectedSSASError:
		// Retry on 5xx errors, but not 4xx (client errors)
		if e.SsasStatusCode >= 500 {
			return true
		}
		return false
	default:
		// Retry on network errors and other unexpected errors
		// Don't retry on auth failures (401), bad requests (400), parsing errors, etc.
		if _, ok := err.(*customErrors.SSASErrorUnauthorized); ok {
			return false
		}
		if _, ok := err.(*customErrors.SSASErrorBadRequest); ok {
			return false
		}
		if _, ok := err.(*customErrors.InternalParsingError); ok {
			return false
		}
		if _, ok := err.(*customErrors.ConfigError); ok {
			return false
		}
		// For other errors (like network errors), retry
		return true
	}
}

// introspectWithRetry attempts to call introspect with exponential backoff retry
func (h HealthChecker) introspectWithRetry(c *ssasClient.SSASClient, ctx context.Context, tokenString string) ([]byte, error) {
	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = introspectRetryInitial
	eb.MaxInterval = 2 * time.Second
	eb.Multiplier = 2.0
	b := backoff.WithMaxRetries(eb, introspectRetryCount)

	var result []byte
	var lastErr error

	err := backoff.RetryNotify(func() error {
		var err error
		result, err = c.CallSSASIntrospect(ctx, tokenString)
		if err != nil {
			lastErr = err
			if !isRetryableError(err) {
				// Don't retry non-retryable errors
				return backoff.Permanent(err)
			}
			return err
		}
		return nil
	}, b, func(err error, d time.Duration) {
		log.API.Warnf("Health check: SSAS introspect - introspect request failed, retrying in %s: %s", d.String(), err.Error())
	})

	if err != nil {
		return nil, lastErr
	}

	return result, nil
}
