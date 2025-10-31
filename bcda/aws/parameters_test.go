package bcdaaws

import (
	"errors"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
)

func TestGetParameter(t *testing.T) {
	key1 := "key1"
	val1 := "val1"
	cleanupParam1 := testUtils.SetParameter(t, key1, val1)
	defer cleanupParam1()

	tests := []struct {
		desc          string
		keyname       string
		expectedValue string
		expectedErr   error
	}{
		{
			desc:          "Happy path",
			keyname:       key1,
			expectedValue: val1,
			expectedErr:   nil,
		},
		{
			desc:          "Missing parameter",
			keyname:       "asdf",
			expectedValue: "",
			expectedErr:   errors.New("error retrieving parameter asdf from parameter store"),
		},
	}

	client := testUtils.TestSSMClient(t, testUtils.TestAWSConfig(t))
	for _, test := range tests {
		value, err := GetParameter(t.Context(), client, test.keyname)
		assert.Equal(t, test.expectedValue, value)

		if test.expectedErr == nil {
			assert.Nil(t, err)
		} else {
			assert.Contains(t, err.Error(), test.expectedErr.Error())
		}
	}
}

func TestGetParameters(t *testing.T) {
	key1 := "key1"
	key2 := "key2"
	val1 := "val1"
	val2 := "val2"

	cleanupParam1 := testUtils.SetParameter(t, key1, val1)
	cleanupParam2 := testUtils.SetParameter(t, key2, val2)
	defer cleanupParam1()
	defer cleanupParam2()

	tests := []struct {
		desc string
		keys []string
		vals map[string]string
		err  error
	}{
		{
			desc: "Happy path",
			keys: []string{key1, key2},
			vals: map[string]string{key1: val1, key2: val2},
			err:  nil,
		},
		{
			desc: "Invalid parameter",
			keys: []string{"invalid", key2},
			vals: nil,
			err:  errors.New("invalid parameters error: invalid,\n"),
		},
	}

	client := testUtils.TestSSMClient(t, testUtils.TestAWSConfig(t))
	for _, test := range tests {
		vals, err := GetParameters(t.Context(), client, test.keys)

		assert.Equal(t, test.vals, vals)
		assert.Equal(t, test.err, err)
	}
}
