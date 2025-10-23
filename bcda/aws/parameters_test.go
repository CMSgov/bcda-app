package bcdaaws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/stretchr/testify/assert"
)

func TestGetParameter(t *testing.T) {
	key1 := "key1"
	val1 := "val1"
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	assert.Nil(t, err)
	client := ssm.NewFromConfig(cfg)

	paramInput := ssm.PutParameterInput{
		Name:      &key1,
		Value:     &val1,
		Overwrite: aws.Bool(true),
		Type:      "String",
	}

	_, err = client.PutParameter(t.Context(), &paramInput)
	assert.Nil(t, err)

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
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	assert.Nil(t, err)
	client := ssm.NewFromConfig(cfg)

	paramInput1 := ssm.PutParameterInput{
		Name:      &key1,
		Value:     &val1,
		Overwrite: aws.Bool(true),
		Type:      "String",
	}
	_, err = client.PutParameter(t.Context(), &paramInput1)
	assert.Nil(t, err)

	paramInput2 := ssm.PutParameterInput{
		Name:      &key2,
		Value:     &val2,
		Overwrite: aws.Bool(true),
		Type:      "String",
	}
	_, err = client.PutParameter(t.Context(), &paramInput2)
	assert.Nil(t, err)

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

	for _, test := range tests {
		vals, err := GetParameters(t.Context(), client, test.keys)

		assert.Equal(t, test.vals, vals)
		assert.Equal(t, test.err, err)
	}
}
