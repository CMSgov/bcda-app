package models

import (
	"fmt"
	"time"
)

type Termination struct {
	Date                time.Time
	BlacklistType       Blacklist
	AttributionStrategy Attribution
	OptOutStrategy      OptOut
	ClaimsStrategy      Claims
}

// AttributionDate returns the date that should be used for attribution
// based on the associated attribution strategy.
// The returned date should be used as an upper bound when querying for
// attribution data.
func (t *Termination) AttributionDate() time.Time {
	switch t.AttributionStrategy {
	case AttributionHistorical:
		return t.Date
	case AttributionLatest:
		// By returning a zero time, we signal to the caller
		// that there should not be an upper bound placed on the
		// attribution search.
		// If we sent over time.Now(), we may unintentionally exclude results
		// because of clock skew.
		return time.Time{}
	default:
		panic(fmt.Sprintf("Unsupported attribution strategy %d supplied.", t.AttributionStrategy))
	}
}

// OptOutDate returns the date that should be used for opt-outs 
// based on the associated opt-out strategy.
// The returned date should be used as an upper bound when querying for
// opt-out data.
func (t *Termination) OptOutDate() time.Time {
	switch t.OptOutStrategy {
	case OptOutHistorical:
		return t.Date
	case OptOutLatest:
		// By returning a zero time, we signal to the caller
		// that there should not be an upper bound placed on the
		// opt-out search.
		// If we sent over time.Now(), we may unintentionally exclude results
		// because of clock skew.
		return time.Time{}
	default:
		panic(fmt.Sprintf("Unsupported opt-out strategy %d supplied.", t.OptOutStrategy))
	}
}

// ClaimsDate returns the date that should be used for claims
// based on the associated claims strategy.
// The returned date should be used as an upper bound when querying for
// claims data.
func (t *Termination) ClaimsDate() time.Time {
	switch t.ClaimsStrategy {
	case ClaimsHistorical:
		return t.Date
	case ClaimsLatest:
		// By returning a zero time, we signal to the caller
		// that there should not be an upper bound placed on the
		// claims search.
		// If we sent over time.Now(), we may unintentionally exclude results
		// because of clock skew.
		return time.Time{}
	default:
		panic(fmt.Sprintf("Unsupported claims strategy %d supplied.", t.ClaimsStrategy))
	}
}

type Blacklist uint8

const (
	// Involuntary means the caller had access revoked immediately
	Involuntary Blacklist = iota
	// Voluntary means the caller had limited access then had their access completely revoked
	Voluntary
	// Limited means the caller has limited access to the service
	Limited
)

type Attribution uint8

const (
	AttributionHistorical Attribution = iota
	AttributionLatest
)

type OptOut uint8

const (
	OptOutHistorical OptOut = iota
	OptOutLatest
)

type Claims uint8

const (
	ClaimsHistorical Claims = iota
	ClaimsLatest
)
