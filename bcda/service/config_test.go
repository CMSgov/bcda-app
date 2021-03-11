package service

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestLoadConfig verifies the configuration reference by BCDA_API_CONFIG_PATH
// can be loaded properly
func TestLoadConfig(t *testing.T) {
	t.Log("Loading configuration from " + os.Getenv("BCDA_API_CONFIG_PATH"))
	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.ACOConfigs, 6)
	for _, acoCfg := range cfg.ACOConfigs {
		assert.NotNil(t, acoCfg.patternExp)
		if acoCfg.PerfYearTransition != "" {
			assert.False(t, acoCfg.perfYear.IsZero(), "perfYear should be set")
		}
		t.Log(acoCfg.String())
	}
	// Ensure that fields with the same name can be represented by different values
	// NOTE: These values come from local.env
	assert.Equal(t, cfg.CutoffDurationDays, 0)
	assert.Equal(t, cfg.RunoutConfig.CutoffDurationDays, 180)
	t.Log(cfg.String())
	t.Log(cfg.RunoutConfig.String())
}

func TestLookbackTime(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	now := time.Now()
	perfYearPast, perfYearFuture := now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour)
	lookback := rand.Intn(4) + 1

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
