package service

import (
	"os"
	"testing"

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
