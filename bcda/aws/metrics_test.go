package bcdaaws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
)

func TestPutMetricSample(t *testing.T) {
	err := PutMetricSample(
		t.Context(),
		"Namespace",
		"Name",
		"Count",
		float64(32),
		[]types.Dimension{{Name: aws.String("name"), Value: aws.String("value")}},
	)
	assert.Nil(t, err)
}
