package bcdaaws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type CustomSSMClient interface {
	GetParameter(ctx context.Context, input *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	GetParameters(ctx context.Context, input *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
}

type MockSSMClient struct{}

func (m *MockSSMClient) GetParameter(ctx context.Context, input *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	output := &ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			Name: input.Name,
			Value: aws.String("value"),
		},
	}
	return output, nil
}

func (m *MockSSMClient) GetParameters(ctx context.Context, input *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error) {
	params := []types.Parameter{}
	for i, name := range input.Names {
		params = append(params, types.Parameter{
			Name: aws.String(name),
			Value: aws.String(fmt.Sprintf("value%d", i+1)),
		})
	}
	output := &ssm.GetParametersOutput{Parameters: params,}

	return output, nil
}
