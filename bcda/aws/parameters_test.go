package bcdaaws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetParameter(t *testing.T) {
	value, err := GetParameter(t.Context(), &MockSSMClient{}, "name")
	assert.Nil(t, err)
	assert.Equal(t, "value", value)
}

func TestGetParameters(t *testing.T) {
	vals, err := GetParameters(t.Context(), &MockSSMClient{}, []string{"name", "other"})
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"name": "value1", "other": "value2"}, vals)
}
