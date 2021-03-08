package service_test

import (
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/stretchr/testify/assert"
)

// TestLoadConfig verifies the configuration reference by BCDA_API_CONFIG_PATH
// can be loaded properly
func TestLoadConfig(t *testing.T) {
	t.Log("Loading configuration from " + os.Getenv("BCDA_API_CONFIG_PATH"))
	cfg, err := service.LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.ACOConfigs, 6)
	// Ensure that fields with the same name can be represented by different values
	// NOTE: These values come from local.env
	assert.Equal(t, cfg.CutoffDurationDays, 0)
	assert.Equal(t, cfg.RunoutConfig.CutoffDurationDays, 180)
}
