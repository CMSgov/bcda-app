package bcdaaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type CustomSSMClient interface {
	GetParameter(ctx context.Context, input *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	GetParameters(ctx context.Context, input *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
	PutParameter(ctx context.Context, input *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
}

type MockSSMClient struct {
	Params map[string]string
}

func (m *MockSSMClient) GetParameter(ctx context.Context, input *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	value := ""
	if m.Params != nil {
		value = m.Params[*input.Name]
	}
	output := &ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			Name:  input.Name,
			Value: &value,
		},
	}
	return output, nil
}

func (m *MockSSMClient) GetParameters(ctx context.Context, input *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error) {
	params := []types.Parameter{}
	for _, name := range input.Names {
		value := ""
		if m.Params != nil {
			value = m.Params[name]
		}
		params = append(params, types.Parameter{
			Name:  aws.String(name),
			Value: aws.String(value),
		})
	}
	output := &ssm.GetParametersOutput{Parameters: params}

	return output, nil
}

func (m *MockSSMClient) PutParameter(ctx context.Context, input *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	if m.Params == nil {
		m.Params = make(map[string]string)
	}
	m.Params[*input.Name] = *input.Value
	output := &ssm.PutParameterOutput{}
	return output, nil
}
