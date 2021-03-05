package service_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/stretchr/testify/assert"
)

func TestFoo(t *testing.T) {
	cfg, err := service.LoadConfig()
	assert.NoError(t, err)
	fmt.Printf("%+v\n", cfg)
	fmt.Println(os.Getenv("BCDA_SUPPRESSION_LOOKBACK_DAYS"))
}
