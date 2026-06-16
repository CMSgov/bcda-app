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
	"github.com/stretchr/testify/require"
)

// TestLoadConfig verifies the configuration reference by BCDA_API_CONFIG_PATH
// can be loaded properly
func TestLoadConfig(t *testing.T) {
	t.Log("Loading configuration from " + os.Getenv("BCDA_API_CONFIG_PATH"))
	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.ACOConfigs, 15)
	for _, acoCfg := range cfg.ACOConfigs {
		assert.NotNil(t, acoCfg.patternExp)
		if acoCfg.PerfYearTransition != "" {
			assert.False(t, acoCfg.perfYear.IsZero(), "perfYear should be set")
		}
		t.Log(toJSON(acoCfg))
	}
	// Ensure that fields with the same name can be represented by different values
	// NOTE: These values come from local.env
	assert.Equal(t, 0, cfg.CutoffDurationDays)
	assert.Equal(t, 180, cfg.RunoutConfig.CutoffDurationDays)
	assert.Equal(t, false, cfg.RateLimitConfig.All)
	assert.Equal(t, 1, len(cfg.RateLimitConfig.ACOs))
	assert.Equal(t, "A4875", cfg.RateLimitConfig.ACOs[0])
	t.Log(toJSON(cfg))
	t.Log(toJSON(cfg.RunoutConfig))
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

func TestIsACOV3Enabled(t *testing.T) {
	originalDeploymentTarget := os.Getenv("DEPLOYMENT_TARGET")
	defer func() {
		os.Setenv("DEPLOYMENT_TARGET", originalDeploymentTarget)
	}()
	os.Setenv("DEPLOYMENT_TARGET", "prod")

	tests := []struct {
		name     string
		acoID    string
		expected bool
		cfg      *Config
	}{
		{
			name:     "ACOInEnabledList",
			acoID:    "A1234",
			expected: true,
			cfg: &Config{
				V3EnabledACOs: []string{"A1234", "A5678", "A9990"},
			},
		},
		{
			name:     "ACONotInEnabledList",
			acoID:    "A9999",
			expected: false,
			cfg: &Config{
				V3EnabledACOs: []string{"A1234", "A5678", "A9990"},
			},
		},
		{
			name:     "EmptyEnabledList",
			acoID:    "A1234",
			expected: false,
			cfg: &Config{
				V3EnabledACOs: []string{},
			},
		},
		{
			name:     "NilEnabledList",
			acoID:    "A1234",
			expected: false,
			cfg: &Config{
				V3EnabledACOs: nil,
			},
		},
		{
			name:     "EmptyCMSID",
			acoID:    "",
			expected: false,
			cfg: &Config{
				V3EnabledACOs: []string{"A1234", "A5678"},
			},
		},
		{
			name:     "CaseSensitiveMatch",
			acoID:    "a1234",
			expected: false,
			cfg: &Config{
				V3EnabledACOs: []string{"A1234", "A5678"},
			},
		},
		{
			name:     "ExactMatch",
			acoID:    "A1234",
			expected: true,
			cfg: &Config{
				V3EnabledACOs: []string{"A1234", "A5678"},
			},
		},
		{
			name:     "PartialMatch",
			acoID:    "A123",
			expected: false,
			cfg: &Config{
				V3EnabledACOs: []string{"A1234", "A5678"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.IsACOV3Enabled(tt.acoID)
			assert.Equal(t, tt.expected, result,
				"Expected V3 access enabled=%v for CMS ID '%s' in test case '%s'",
				tt.expected, tt.acoID, tt.name)
		})
	}
}

func TestIsACOV3Enabled_EnvironmentBased(t *testing.T) {
	originalDeploymentTarget := os.Getenv("DEPLOYMENT_TARGET")
	defer func() {
		os.Setenv("DEPLOYMENT_TARGET", originalDeploymentTarget)
	}()

	tests := []struct {
		name             string
		acoID            string
		deploymentTarget string
		enabledACOs      []string
		expected         bool
	}{
		{
			name:             "NonProduction_Dev",
			acoID:            "ANY1234",
			deploymentTarget: "dev",
			enabledACOs:      []string{"A1234", "A5678"},
			expected:         true, // Should allow any ACO in non-production
		},
		{
			name:             "NonProduction_Test",
			acoID:            "ANY1234",
			deploymentTarget: "test",
			enabledACOs:      []string{"A1234", "A5678"},
			expected:         true, // Should allow any ACO in non-production
		},
		{
			name:             "NonProduction_Sandbox",
			acoID:            "ANY1234",
			deploymentTarget: "sandbox",
			enabledACOs:      []string{"A1234", "A5678"},
			expected:         true, // Should allow any ACO in non-production
		},
		{
			name:             "NonProduction_Empty",
			acoID:            "ANY1234",
			deploymentTarget: "",
			enabledACOs:      []string{"A1234", "A5678"},
			expected:         true, // Should allow any ACO in non-production
		},
		{
			name:             "NonProduction_Unknown",
			acoID:            "ANY1234",
			deploymentTarget: "unknown",
			enabledACOs:      []string{"A1234", "A5678"},
			expected:         true, // Should allow any ACO in non-production
		},
		{
			name:             "Production_ACOInList",
			acoID:            "A1234",
			deploymentTarget: "prod",
			enabledACOs:      []string{"A1234", "A5678"},
			expected:         true, // Should allow ACO in list
		},
		{
			name:             "Production_ACONotInList",
			acoID:            "A9999",
			deploymentTarget: "prod",
			enabledACOs:      []string{"A1234", "A5678"},
			expected:         false, // Should deny ACO not in list
		},
		{
			name:             "Production_EmptyList",
			acoID:            "A1234",
			deploymentTarget: "prod",
			enabledACOs:      []string{},
			expected:         false, // Should deny when list is empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DEPLOYMENT_TARGET", tt.deploymentTarget)
			cfg := &Config{
				V3EnabledACOs: tt.enabledACOs,
			}

			result := cfg.IsACOV3Enabled(tt.acoID)
			assert.Equal(t, tt.expected, result,
				"Expected V3 access enabled=%v for CMS ID '%s' in test case '%s'",
				tt.expected, tt.acoID, tt.name)
		})
	}
}

func TestSupportedACOs(t *testing.T) {
	cfg, err := LoadConfig()
	require.NoError(t, err)

	tests := []struct {
		name        string
		cmsID       string
		isSupported bool
	}{
		{"SSP too short", "A999", false},
		{"SSP too long", "A99999", false},
		{"SSP invalid characters", "A999A", false},
		{"valid SSP", "A9999", true},

		{"NGACO too short", "V99", false},
		{"NGACO too long", "V9999", false},
		{"NGACO invalid characters", "V99V", false},
		{"valid NGACO", "V999", true},

		{"CEC too short", "E999", false},
		{"CEC too long", "E99999", false},
		{"CEC invalid characters", "E999E", false},
		{"valid CEC", "E9999", true},

		{"CKCC too short", "C999", false},
		{"CKCC too long", "C99999", false},
		{"CKCC invalid characters", "C999V", false},
		{"valid CKCC", "C9999", true},

		{"KCF too short", "K999", false},
		{"KCF too long", "K99999", false},
		{"KCF invalid characters", "K999V", false},
		{"valid KCF", "K9999", true},

		{"DC too short", "D999", false},
		{"DC too long", "D99999", false},
		{"DC invalid characters", "D999V", false},
		{"valid DC", "D9999", true},

		{"MDTCOC too short", "CT999", false},
		{"MDTCOC too long", "CT9999999", false},
		{"MDTCOC invalid characters", "CT999V", false},
		{"valid MDTCOC", "CT99999", true},

		{"CDAC too short", "DA999", false},
		{"CDAC too long", "DA9999999", false},
		{"CDAC invalid characters", "DA999V", false},
		{"valid CDAC", "DA9999", true},

		{"GUIDE too short", "GUIDE-999", false},
		{"GUIDE too long", "GUIDE-9999999", false},
		{"GUIDE invalid characters", "GUIDE99999", false},
		{"valid GUIDE", "GUIDE-99999", true},

		{"Iota too short", "IOTA12", false},
		{"Iota too long", "IOTA0123", false},
		{"Iota invalid characters 1", "IOTA12Z", false},
		{"Iota invalid characters 2", "IOTA1YZ", false},
		{"Iota invalid characters 3", "IOTAXYZ", false},
		{"valid Iota", "IOTA123", true},

		{"SBX too short", "SBXB1", false},
		{"SBX too long", "SBXPA0123", false},
		{"SBX invalid characters 1", "SBX0A123", false},
		{"SBX invalid characters 2", "SBXA0123", false},
		{"SBX invalid characters 3", "SBXADXYZ", false},
		{"valid SBX", "SBXAD123", true},

		{"Unregistered ACO", "Z1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(sub *testing.T) {
			match := cfg.IsSupportedACO(tt.cmsID)
			assert.Equal(sub, tt.isSupported, match)
		})
	}
}

func compileRegex(t *testing.T, pattern string) *regexp.Regexp {
	patternExp, err := regexp.Compile(pattern)
	assert.NoError(t, err)
	return patternExp
}
