package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	ssasClient "github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/client"
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

const introspectCacheTTL = 5 * time.Minute

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

	// Create a minimal request with context for GetToken
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.CtxTransactionKey, "health-check")
	req, err := http.NewRequestWithContext(ctx, "GET", "/", nil)
	if err != nil {
		result := "Failed to create request"
		h.introspectCache.result = result
		h.introspectCache.ok = false
		h.introspectCache.timestamp = time.Now()
		log.API.Error("Health check: SSAS introspect - failed to create request: ", err.Error())
		return result, false
	}

	// Get token
	tokenInfo, err := c.GetToken(ssasClient.Credentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}, *req)
	if err != nil {
		result := "Failed to get BCDA admin token"
		h.introspectCache.result = result
		h.introspectCache.ok = false
		h.introspectCache.timestamp = time.Now()
		log.API.Error("Health check: SSAS introspect - failed to get token: ", err.Error())
		return result, false
	}

	// Parse token response to extract access_token
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in,omitempty"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal([]byte(tokenInfo), &tokenResp); err != nil {
		result := "Failed to parse token response"
		h.introspectCache.result = result
		h.introspectCache.ok = false
		h.introspectCache.timestamp = time.Now()
		log.API.Error("Health check: SSAS introspect - failed to parse token: ", err.Error())
		return result, false
	}

	if tokenResp.AccessToken == "" {
		result := "Empty access token in response"
		h.introspectCache.result = result
		h.introspectCache.ok = false
		h.introspectCache.timestamp = time.Now()
		log.API.Error("Health check: SSAS introspect - empty access token")
		return result, false
	}

	// Call introspect endpoint
	_, err = c.CallSSASIntrospect(ctx, tokenResp.AccessToken)
	if err != nil {
		result := "SSAS introspect check failed"
		h.introspectCache.result = result
		h.introspectCache.ok = false
		h.introspectCache.timestamp = time.Now()
		log.API.Error("Health check: SSAS introspect - introspect call failed: ", err.Error())
		return result, false
	}

	// Success - update cache
	h.introspectCache.result = "ok"
	h.introspectCache.ok = true
	h.introspectCache.timestamp = time.Now()
	return "ok", true
}
