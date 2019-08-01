package metrics

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

type Dimension struct {
	Name  string
	Value string
}

type Sampler struct {
	Namespace string
	Unit      string
	Service   *cloudwatch.CloudWatch
}

func (s *Sampler) PutSample(name string, value float64, dimensions []Dimension) error {
	var d []*cloudwatch.Dimension

	for _, v := range dimensions {
		def := &cloudwatch.Dimension{
			Name:  &v.Name,
			Value: &v.Value,
		}
		d = append(d, def)
	}

	data := &cloudwatch.MetricDatum{
		Dimensions: d,
		MetricName: &name,
		Unit:       &s.Unit,
		Value:      &value,
	}

	input := &cloudwatch.PutMetricDataInput{
		MetricData: []*cloudwatch.MetricDatum{data},
		Namespace:  &s.Namespace,
	}
	fmt.Println(input)
	_, err := s.Service.PutMetricData(input)
	return err
}

func NewSampler(ns, unit string) (*Sampler, error) {
	s := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	svc := cloudwatch.New(s)
	return &Sampler{ns, unit, svc}, nil
}
