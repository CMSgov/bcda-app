package bcdaaws

import (
	"context"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func PutMetricSample(
	ctx context.Context,
	client CustomCloudwatchClient,
	namespace string,
	name string,
	unit types.StandardUnit,
	value float64,
	dimensions []types.Dimension,
) error {
	data := types.MetricDatum{
		Dimensions: dimensions,
		MetricName: &name,
		Unit:       unit,
		Value:      &value,
	}

	input := &cloudwatch.PutMetricDataInput{
		MetricData: []types.MetricDatum{data},
		Namespace:  aws.String(namespace),
	}

	_, err := client.PutMetricData(ctx, input, func(o *cloudwatch.Options) {
		o.Region = constants.DefaultRegion
	})

	return err
}
