package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTerminationDates(t *testing.T) {
	termination := &Termination{
		TerminationDate:     time.Now(),
		CutoffDate:          time.Now().Add(time.Hour),
		AttributionStrategy: AttributionHistorical,
		OptOutStrategy:      OptOutHistorical,
		ClaimsStrategy:      ClaimsHistorical,
	}

	// Historical strategies use the termination date
	assert.Equal(t, termination.TerminationDate, termination.AttributionDate())
	assert.Equal(t, termination.TerminationDate, termination.OptOutDate())
	assert.Equal(t, termination.TerminationDate, termination.ClaimsDate())

	termination.AttributionStrategy = AttributionLatest
	termination.OptOutStrategy = OptOutLatest
	termination.ClaimsStrategy = ClaimsLatest

	// Latest strategies return default time
	assert.Equal(t, time.Time{}, termination.AttributionDate())
	assert.Equal(t, time.Time{}, termination.OptOutDate())
	assert.Equal(t, time.Time{}, termination.ClaimsDate())
}
