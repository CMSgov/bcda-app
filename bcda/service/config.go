package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/sirupsen/logrus"
)

type Config struct {
	SuppressionLookbackDays int `conf:"BCDA_SUPPRESSION_LOOKBACK_DAYS" conf_default:"45"`
	CutoffDurationDays      int `conf:"CCLF_CUTOFF_DATE_DAYS" conf_default:"60"`

	RunoutConfig `conf:",squash"`

	ACOConfigs []*ACOConfig `conf:"aco_config"`

	// Un-exported fields that are computed using the exported ones above
	cutoffDuration time.Duration
}

type RunoutConfig struct {
	CutoffDurationDays int    `conf:"RUNOUT_CUTOFF_DATE_DAYS" conf_default:"180"`
	ClaimThruDate      string `conf:"RUNOUT_CLAIM_THRU_DATE" conf_default:"2020-12-31"`
	// Un-exported fields that are computed using the exported ones above
	cutoffDuration time.Duration
	claimThru      time.Time
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
	d, err := json.Marshal(config)
	if err != nil {
		return fmt.Sprintf("failed to marshal config %s", err.Error())
	}
	return string(d)
}

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
	for _, acoCfg := range cfg.ACOConfigs {
		if acoCfg.patternExp, err = regexp.Compile(acoCfg.Pattern); err != nil {
			return fmt.Errorf("failed to parse ACO model %s pattern: %w", acoCfg.Model, err)
		}
		if acoCfg.PerfYearTransition != "" {
			if acoCfg.perfYear, err = time.Parse(perfYearLayout, acoCfg.PerfYearTransition); err != nil {
				return fmt.Errorf("failed to parse perf year: %w", err)
			}
		}
	}

	return nil
}
