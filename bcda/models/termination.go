package models

type Termination struct {
	BlacklistType       Blacklist
	AttributionStrategy Attribution
	OptOutStrategy      OptOut
	ClaimsStrategy      Claims
}

type Blacklist uint8

const (
	Involuntary Blacklist = iota
	BlacklistedVoluntary
	Voluntary
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
	ClaimsHistorical OptOut = iota
	ClaimsLatest
)
