package bcdaaws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetParameter(t *testing.T) {
	params := map[string]string{"name": "value"}
	ssmClient := MockSSMClient{Params: params}

	value, err := GetParameter(t.Context(), &ssmClient, "name")
	assert.Nil(t, err)
	assert.Equal(t, "value", value)
}

func TestGetParameters(t *testing.T) {
	params := map[string]string{"name": "value1", "other": "value2"}
	ssmClient := MockSSMClient{Params: params}
	vals, err := GetParameters(t.Context(), &ssmClient, []string{"name", "other"})
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"name": "value1", "other": "value2"}, vals)
}
