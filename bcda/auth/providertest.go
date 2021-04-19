package auth

import (
	"testing"
)

// SetMockProvider sets the current provider to the one that's supplied in this function.
// It leverages the Cleanup() func to ensure the original provider is restored at the end of the test.
func SetMockProvider(t *testing.T, other *MockProvider) {
	// Ensure that we restore the original provider when the test completes
	originalProvider := provider
	t.Cleanup(func() {
		provider = originalProvider
	})
	provider = other
}
