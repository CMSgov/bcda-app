package service

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
)

func LoadConfig() (cfg *Config, err error) {
	cfg = &Config{}
	if err := conf.Checkout(cfg); err != nil {
		return nil, err
	}

	if err := cfg.ComputeFields(); err != nil {
		return nil, err
	}

	log.API.Info("Successfully loaded configuration for Service.")

	return cfg, nil
}

type Config struct {
	SuppressionLookbackDays int         `conf:"BCDA_SUPPRESSION_LOOKBACK_DAYS" conf_default:"60"`
	CutoffDurationDays      int         `conf:"CCLF_CUTOFF_DATE_DAYS" conf_default:"50"`
	ACOConfigs              []ACOConfig `conf:"aco_config"`
	V3EnabledACOs           []string    `conf:"v3_enabled_acos"` // Simple list of ACOs with v3 access
	CutoffDuration          time.Duration
	RateLimitConfig         RateLimitConfig `conf:"rate_limit_config"`
	V1V2DenyRegexes         []string        `conf:"v1_v2_deny_regexes"`
	V3NoPartialClaimsModels []string        `conf:"v3_no_partial_claims_models"`
	// Use the squash tag to allow the RunoutConfigs to avoid requiring the parameters
	// to be defined as a child of RunoutConfig.
	// Ex: Without the ,squash, we would have to have RunoutConfig.RUNOUT_CUTOFF_DATE_DAYS
	// With the ,squash, we would have RUNOUT_CUTOFF_DATE_DAYS.
	RunoutConfig RunoutConfig `conf:",squash"`
}

type RunoutConfig struct {
	CutoffDurationDays int    `conf:"RUNOUT_CUTOFF_DATE_DAYS" conf_default:"180"`
	ClaimThruDate      string `conf:"RUNOUT_CLAIM_THRU_DATE" conf_default:"2025-12-31"`
	CutoffDuration     time.Duration
	// Un-exported fields that are computed using the exported ones above
	claimThru time.Time
}

type ACOConfig struct {
	Model              string
	Pattern            string          `conf:"name_pattern"`
	PerfYearTransition string          `conf:"performance_year_transition"`
	LookbackYears      int             `conf:"lookback_period"`
	Disabled           bool            `conf:"disabled" conf_default:"false"`
	Data               []string        `conf:"data"`
	IgnoreSuppressions bool            `conf:"ignore_suppressions" conf_default:"false"`
	AttributionFile    AttributionFile `conf:"attribution_file"`

	// Un-exported fields that are computed using the exported ones above
	patternExp *regexp.Regexp
	perfYear   time.Time
}

type AttributionFile struct {
	FileType        string `conf:"file_type"`
	NamePattern     string `conf:"name_pattern" `
	MetadataMatches int    `conf:"metadata_matches" `
	ModelIdentifier string `conf:"model_identifier"`
	PerformanceYear int    `conf:"file_performance_year"`
	FileDate        int    `conf:"file_date"`
}

type RateLimitConfig struct {
	All  bool     `conf:"all"`  // rate-limit requests for all ACOs
	ACOs []string `conf:"acos"` // rate-limit requests for specific ACOs
}

func toJSON(config interface{}) string {
	d, err := json.Marshal(config)
	if err != nil {
		return fmt.Sprintf("failed to marshal config %s", err.Error())
	}
	return string(d)
}

// Parse un-exported fields using the fields loaded via the config
func (cfg *Config) ComputeFields() (err error) {
	const (
		// YYYY-MM-DD
		claimThruLayout = "2006-01-02"
		// MM/DD
		perfYearLayout = "01/02"
	)

	cfg.CutoffDuration = 24 * time.Hour * time.Duration(cfg.CutoffDurationDays)
	cfg.RunoutConfig.CutoffDuration = 24 * time.Hour * time.Duration(cfg.RunoutConfig.CutoffDurationDays)
	if cfg.RunoutConfig.claimThru, err = time.Parse(claimThruLayout, cfg.RunoutConfig.ClaimThruDate); err != nil {
		return fmt.Errorf("failed to parse runout claim thru date: %w", err)
	}

	// Replace the ACO configs inline with computed columns
	for idx := range cfg.ACOConfigs {
		if cfg.ACOConfigs[idx].patternExp, err = regexp.Compile(cfg.ACOConfigs[idx].Pattern); err != nil {
			return fmt.Errorf("failed to parse ACO model %s pattern: %w", cfg.ACOConfigs[idx].Model, err)
		}
		if cfg.ACOConfigs[idx].PerfYearTransition != "" {
			if cfg.ACOConfigs[idx].perfYear, err = time.Parse(perfYearLayout, cfg.ACOConfigs[idx].PerfYearTransition); err != nil {
				return fmt.Errorf("failed to parse perf year: %w", err)
			}
		}
	}

	return nil
}

func (cfg *Config) IsACODisabled(CMSID string) bool {
	for _, ACOcfg := range cfg.ACOConfigs {
		if ACOcfg.patternExp.MatchString(CMSID) {
			return ACOcfg.Disabled
		}
	}
	// If the ACO does not exist in our config they are automatically disabled
	return true
}

func (cfg *Config) IsACOV3Enabled(ACOID string) bool {
	if os.Getenv("DEPLOYMENT_TARGET") != "prod" {
		return true
	}

	for _, aco := range cfg.V3EnabledACOs {
		if aco == ACOID {
			return true
		}
	}
	return false
}

// IsSupportedACO determines if the particular ACO is supported by checking
// its CMS_ID against the supported formats.
func (cfg *Config) IsSupportedACO(cmsID string) bool {
	for _, aco := range cfg.ACOConfigs {
		if aco.patternExp.MatchString(cmsID) {
			return true
		}
	}
	return false
}

// LookbackTime returns the timestamp that we should use as the lookback time associated with the ACO.
// We compute lookback time by evaluating the performance year transition and the number of lookback years.
func (config *ACOConfig) LookbackTime(ACOID string) time.Time {
	// GUIDE has very specific lookback windows
	isGUIDE := regexp.MustCompile(`^GUIDE-\d{4}$`).MatchString(ACOID)
	if isGUIDE {
		if slices.Contains(GUIDEEPTACOs, ACOID) {
			return GUIDEEPTLookbackDate
		} else {
			return GUIDENPTLookbackDate
		}
	}

	// default lookback windows (lookback_period not set in ACOs config)
	if config.perfYear.IsZero() || config.LookbackYears == 0 {
		return time.Time{}
	}

	// calculated lookback windows
	now := time.Now()
	var year int
	// If we passed our perf year transition, we consider us to be in the new performance year.
	// Otherwise we are still in the previous performance year.
	if now.Month() > config.perfYear.Month() || (now.Month() == config.perfYear.Month() &&
		now.Day() >= config.perfYear.Day()) {
		year = now.Year() - config.LookbackYears
	} else {
		year = now.Year() - 1 - config.LookbackYears
	}

	return time.Date(year, config.perfYear.Month(), config.perfYear.Day(), 0, 0, 0, 0, time.UTC)
}

// GUIDE has two specific lookback period windows, one for Established Program Tracks (EPTs)
// and one for New Program Tracks (NPTs).  See: https://jira.cms.gov/browse/BCDA-10152
var GUIDEEPTLookbackDate = time.Date(2021, 7, 1, 0, 0, 0, 0, time.UTC)
var GUIDENPTLookbackDate = time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC)
var GUIDEEPTACOs = []string{
	"GUIDE-0001",
	"GUIDE-0016",
	"GUIDE-0034",
	"GUIDE-0037",
	"GUIDE-0045",
	"GUIDE-0058",
	"GUIDE-0062",
	"GUIDE-0068",
	"GUIDE-0074",
	"GUIDE-0076",
	"GUIDE-0085",
	"GUIDE-0092",
	"GUIDE-0106",
	"GUIDE-0115",
	"GUIDE-0117",
	"GUIDE-0118",
	"GUIDE-0130",
	"GUIDE-0156",
	"GUIDE-0166",
	"GUIDE-0168",
	"GUIDE-0169",
	"GUIDE-0173",
	"GUIDE-0188",
	"GUIDE-0208",
	"GUIDE-0209",
	"GUIDE-0237",
	"GUIDE-0247",
	"GUIDE-0254",
	"GUIDE-0259",
	"GUIDE-0263",
	"GUIDE-0266",
	"GUIDE-0271",
	"GUIDE-0280",
	"GUIDE-0284",
	"GUIDE-0334",
	"GUIDE-0336",
	"GUIDE-0342",
	"GUIDE-0345",
	"GUIDE-0408",
	"GUIDE-0420",
	"GUIDE-0429",
	"GUIDE-0441",
	"GUIDE-0446",
	"GUIDE-0448",
	"GUIDE-0452",
	"GUIDE-0468",
	"GUIDE-0473",
	"GUIDE-0475",
	"GUIDE-0485",
	"GUIDE-0499",
	"GUIDE-0521",
	"GUIDE-0525",
	"GUIDE-0526",
	"GUIDE-0529",
	"GUIDE-0534",
	"GUIDE-0540",
	"GUIDE-0552",
	"GUIDE-0563",
	"GUIDE-0573",
	"GUIDE-0578",
	"GUIDE-0591",
	"GUIDE-0634",
	"GUIDE-0651",
	"GUIDE-0665",
	"GUIDE-0671",
	"GUIDE-0677",
	"GUIDE-0682",
	"GUIDE-0689",
	"GUIDE-0693",
	"GUIDE-0707",
	"GUIDE-0708",
	"GUIDE-0721",
	"GUIDE-0756",
	"GUIDE-0765",
	"GUIDE-0777",
	"GUIDE-0779",
	"GUIDE-0782",
	"GUIDE-0783",
	"GUIDE-0790",
	"GUIDE-0806",
	"GUIDE-0838",
	"GUIDE-0841",
	"GUIDE-0844",
	"GUIDE-0864",
	"GUIDE-0877",
	"GUIDE-0883",
	"GUIDE-0893",
	"GUIDE-0903",
	"GUIDE-0937",
	"GUIDE-0962",
	"GUIDE-0977",
	"GUIDE-0979",
	"GUIDE-1008",
	"GUIDE-1023",
	"GUIDE-1033",
}
