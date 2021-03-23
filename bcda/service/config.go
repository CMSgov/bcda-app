package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/sirupsen/logrus"
)

func LoadConfig() (cfg *Config, err error) {
	cfg = &Config{}
	if err := conf.Checkout(cfg); err != nil {
		return nil, err
	}

	if err := cfg.computeFields(); err != nil {
		return nil, err
	}

	logrus.Infof("Successfully loaded config %+v.", cfg)
	return cfg, nil
}

type Config struct {
	SuppressionLookbackDays int `conf:"BCDA_SUPPRESSION_LOOKBACK_DAYS" conf_default:"60"`
	CutoffDurationDays      int `conf:"CCLF_CUTOFF_DATE_DAYS" conf_default:"45"`

	AlrJobSize uint `conf:"alr_job_size" conf_default:"1000"` // Number of entries to put in a single ALR job

	// Use the squash tag to allow the RunoutConfigs to avoid requiring the parameters
	// to be defined as a child of RunoutConfig.
	// Ex: Without the ,squash, we would have to have RunoutConfig.RUNOUT_CUTOFF_DATE_DAYS
	// With the ,squash, we would have RUNOUT_CUTOFF_DATE_DAYS.
	RunoutConfig RunoutConfig `conf:",squash"`

	ACOConfigs []ACOConfig `conf:"aco_config"`

	// Un-exported fields that are computed using the exported ones above
	cutoffDuration time.Duration
}

func (config Config) String() string {
	return toJSON(config)
}

// Parse un-exported fields using the fields loaded via the config
func (cfg *Config) computeFields() (err error) {
	const (
		// YYYY-MM-DD
		claimThruLayout = "2006-01-02"
		// MM/DD
		perfYearLayout = "01/02"
	)

	cfg.cutoffDuration = 24 * time.Hour * time.Duration(cfg.CutoffDurationDays)
	cfg.RunoutConfig.cutoffDuration = 24 * time.Hour * time.Duration(cfg.RunoutConfig.CutoffDurationDays)
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

	if cfg.AlrJobSize == 0 {
		return errors.New("invalid ALR job size supplied. Must be greater than zero.")
	}

	return nil
}

type RunoutConfig struct {
	CutoffDurationDays int    `conf:"RUNOUT_CUTOFF_DATE_DAYS" conf_default:"180"`
	ClaimThruDate      string `conf:"RUNOUT_CLAIM_THRU_DATE" conf_default:"2020-12-31"`
	// Un-exported fields that are computed using the exported ones above
	cutoffDuration time.Duration
	claimThru      time.Time
}

func (config RunoutConfig) String() string {
	return toJSON(config)
}

type ACOConfig struct {
	Model              string
	Pattern            string `conf:"name_pattern"`
	PerfYearTransition string `conf:"performance_year_transition"`
	LookbackYears      int    `conf:"lookback_period"`
	// Un-exported fields that are computed using the exported ones above
	patternExp *regexp.Regexp
	perfYear   time.Time
}

func (config *ACOConfig) String() string {
	return toJSON(config)
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
	if now.Month() >= config.perfYear.Month() && now.Day() >= config.perfYear.Day() {
		year = now.Year() - config.LookbackYears
	} else {
		year = now.Year() - 1 - config.LookbackYears
	}

	return time.Date(year, config.perfYear.Month(), config.perfYear.Day(), 0, 0, 0, 0, time.UTC)
}

func toJSON(config interface{}) string {
	d, err := json.Marshal(config)
	if err != nil {
		return fmt.Sprintf("failed to marshal config %s", err.Error())
	}
	return string(d)
}
