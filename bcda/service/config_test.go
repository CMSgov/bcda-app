package service

import (
	"crypto/rand"
	"math/big"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/stretchr/testify/assert"
)

// TestLoadConfig verifies the configuration reference by BCDA_API_CONFIG_PATH
// can be loaded properly
func TestLoadConfig(t *testing.T) {
	t.Log("Loading configuration from " + os.Getenv("BCDA_API_CONFIG_PATH"))
	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.ACOConfigs, 13)
	for _, acoCfg := range cfg.ACOConfigs {
		assert.NotNil(t, acoCfg.patternExp)
		if acoCfg.PerfYearTransition != "" {
			assert.False(t, acoCfg.perfYear.IsZero(), "perfYear should be set")
		}
		t.Log(acoCfg.String())
	}
	// Ensure that fields with the same name can be represented by different values
	// NOTE: These values come from local.env
	assert.Equal(t, 0, cfg.CutoffDurationDays)
	assert.Equal(t, 180, cfg.RunoutConfig.CutoffDurationDays)
	assert.Equal(t, false, cfg.RateLimitConfig.All)
	assert.Equal(t, 1, len(cfg.RateLimitConfig.ACOs))
	assert.Equal(t, "A4875", cfg.RateLimitConfig.ACOs[0])
	t.Log(cfg.String())
	t.Log(cfg.RunoutConfig.String())
}

func TestIsACODisabled(t *testing.T) {
	tests := []struct {
		name     string
		cmsID    string
		expected bool
		cfg      *Config
	}{
		{"ACOIsDisabled", "TEST1234", true, &Config{ACOConfigs: []ACOConfig{{patternExp: compileRegex(t, constants.RegexACOID), Disabled: true}}}},
		{"ACOIsEnabled", "TEST1234", false, &Config{ACOConfigs: []ACOConfig{{patternExp: compileRegex(t, constants.RegexACOID), Disabled: false}}}},
		{"ACODoesNotExist", "DNE1234", true, &Config{ACOConfigs: []ACOConfig{{patternExp: compileRegex(t, constants.RegexACOID), Disabled: false}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cfg.IsACODisabled(tt.cmsID))
		})
	}
}

func TestLookbackTime(t *testing.T) {
	r, err := rand.Int(rand.Reader, big.NewInt(4))
	if err != nil {
		t.Fatalf("Failed to generate random number: %v", err)
	}
	lookback := int(r.Int64()) + 1

	now := time.Now()
	perfYearPast, perfYearFuture := now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour)

	tests := []struct {
		name        string
		cfg         *ACOConfig
		expPerfYear time.Time
	}{
		// We make the call before the performance year transition, we use the previous year as the reference time and then subtract
		// the lookback period. That's why we do the lookback+1
		{"BeforePerfYearTransition", &ACOConfig{perfYear: perfYearFuture, LookbackYears: lookback}, expectedPerfYear(perfYearFuture, lookback+1)},
		// We make the call after the performance year transition, so we use the current year as the baseline, then subtract lookback.
		{"AfterPerfYearTransition", &ACOConfig{perfYear: perfYearPast, LookbackYears: lookback}, expectedPerfYear(perfYearPast, lookback)},
		{"noPerfYear", &ACOConfig{LookbackYears: lookback}, time.Time{}},
		{"noLookback", &ACOConfig{perfYear: now}, time.Time{}},
		{"noPerfYearNoLookback", &ACOConfig{}, time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.expPerfYear.Equal(tt.cfg.LookbackTime()),
				"Times should equal. Have %s. Expected %s.", tt.cfg.LookbackTime(), tt.expPerfYear)
		})
	}
}

func expectedPerfYear(base time.Time, minusYears int) time.Time {
	return time.Date(base.Year()-minusYears, base.Month(), base.Day(), 0, 0, 0, 0, time.UTC)
}

func compileRegex(t *testing.T, pattern string) *regexp.Regexp {
	patternExp, err := regexp.Compile(pattern)
	assert.NoError(t, err)
	return patternExp
}
