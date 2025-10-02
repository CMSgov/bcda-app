package service

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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
	CutoffDurationDays      int         `conf:"CCLF_CUTOFF_DATE_DAYS" conf_default:"45"`
	ACOConfigs              []ACOConfig `conf:"aco_config"`
	V3EnabledACOs           []string    `conf:"v3_enabled_acos"` // Simple list of ACOs with v3 access
	CutoffDuration          time.Duration
	RateLimitConfig         RateLimitConfig `conf:"rate_limit_config"`
	// Use the squash tag to allow the RunoutConfigs to avoid requiring the parameters
	// to be defined as a child of RunoutConfig.
	// Ex: Without the ,squash, we would have to have RunoutConfig.RUNOUT_CUTOFF_DATE_DAYS
	// With the ,squash, we would have RUNOUT_CUTOFF_DATE_DAYS.
	RunoutConfig RunoutConfig `conf:",squash"`
}

type RunoutConfig struct {
	CutoffDurationDays int    `conf:"RUNOUT_CUTOFF_DATE_DAYS" conf_default:"180"`
	ClaimThruDate      string `conf:"RUNOUT_CLAIM_THRU_DATE" conf_default:"2024-12-31"`
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

func (config Config) String() string {
	return toJSON(config)
}

func (config RunoutConfig) String() string {
	return toJSON(config)
}

func (config *ACOConfig) String() string {
	return toJSON(config)
}

func (config RateLimitConfig) String() string {
	return toJSON(config)
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

func (config *Config) IsACODisabled(CMSID string) bool {
	for _, ACOcfg := range config.ACOConfigs {
		if ACOcfg.patternExp.MatchString(CMSID) {
			return ACOcfg.Disabled
		}
	}
	// If the ACO does not exist in our config they are automatically disabled
	return true
}

func (config *Config) IsACOV3Enabled(ACOID string) bool {
	if os.Getenv("DEPLOYMENT_TARGET") != "prod" {
		return true
	}

	for _, aco := range config.V3EnabledACOs {
		if aco == ACOID {
			return true
		}
	}
	return false
}

// LookbackTime returns the timestamp that we should use as the lookback time associated with the ACO.
// We compute lookback time by evaluating the performance year transition and the number of lookback years.
func (config *ACOConfig) LookbackTime() time.Time {
	if config.perfYear.IsZero() || config.LookbackYears == 0 {
		return time.Time{}
	}
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
