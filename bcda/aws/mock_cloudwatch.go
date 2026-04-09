package bcdaaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type CustomCloudwatchClient interface {
	PutMetricData(ctx context.Context, input *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
}

type MockCloudwatchClient struct{}

func (m *MockCloudwatchClient) PutMetricData(ctx context.Context, input *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error) {
	output := &cloudwatch.PutMetricDataOutput{}

	return output, nil
}
