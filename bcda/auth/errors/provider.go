package errors

import (
	"fmt"
)

// ProviderError defines HTTP status codes as well as an error message from providers.
type ProviderError struct {
	Code    int
	Message string
}

func (pe *ProviderError) Error() string {
	return fmt.Sprintf("An error was returned from the authentication provider; %d - %s", pe.Code, pe.Message)
}
