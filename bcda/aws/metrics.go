package bcdaaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type Sampler struct {
	Ctx       context.Context
	Namespace string
	Unit      string
	Service   *cloudwatch.Client
}

func PutMetricSample(
	ctx context.Context,
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

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	client := cloudwatch.NewFromConfig(cfg)

	_, err = client.PutMetricData(ctx, input)

	return err
}
