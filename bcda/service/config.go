package service

import (
	"regexp"
	"time"
)

type Config struct {
	CutoffDurationDays      int `conf:"CCLF_CUTOFF_DATE_DAYS"`
	SuppressionLookbackDays int `conf:"BCDA_SUPPRESSION_LOOKBACK_DAYS"`

	RunoutConfig `conf:",squash"`

	ACOConfigs []ACOConfig `conf:"aco_config"`

	cutoffDuration time.Duration
}

type RunoutConfig struct {
	CutoffDurationDays int    `conf:"RUNOUT_CUTOFF_DATE_DAYS"`
	ClaimThruDate      string `conf:"RUNOUT_CLAIM_THRU_DATE"`

	cutoffDuration time.Duration
	claimThru      time.Time
}

type ACOConfig struct {
	Model              string
	Pattern            string
	PerfYearTransition string
	LookbackYears      int

	patternExp          *regexp.Regexp
	perfYear            time.Time
	lookbackPeriodYears int

	/*
			aco_config:
		    - model: "CKCC"
		      name_pattern: "C\\d{4}"
		      performance_year_transition: "04/01"
		      lookback_period: 3
		    - model: "SSP"
		      name_pattern: "A\\d{4}"
	*/
}
