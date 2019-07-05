package ssas

import "time"

type Credentials struct {
	UserID       string
	ClientID     string
	ClientSecret string
	TokenString	 string
	ClientName   string
	ExpiresAt	 time.Time
}
