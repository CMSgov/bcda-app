package bcdaaws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Returns the value of a single parameter from the SSM Parameter Store
func GetParameter(ctx context.Context, client *ssm.Client, keyname string) (string, error) {
	withDecryption := true
	result, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &keyname,
		WithDecryption: &withDecryption,
	})
	if err != nil {
		return "", fmt.Errorf("error retrieving parameter %s from parameter store: %w", keyname, err)
	}

	val := *result.Parameter.Value
	if val == "" {
		return "", fmt.Errorf("no parameter store value found for %s", keyname)
	}

	return val, nil
}

// Returns a list of parameters from the SSM Parameter Store
func GetParameters(ctx context.Context, client *ssm.Client, keynames []string) (map[string]string, error) {
	withDecryption := true
	output, err := client.GetParameters(ctx, &ssm.GetParametersInput{
		Names:          keynames,
		WithDecryption: &withDecryption,
	})
	if err != nil {
		return nil, fmt.Errorf("error connecting to parameter store: %s", err)
	}

	// Unknown keys will come back as invalid, make sure we error on them
	if len(output.InvalidParameters) > 0 {
		invalidParamsStr := ""
		for i := 0; i < len(output.InvalidParameters); i++ {
			invalidParamsStr += fmt.Sprintf("%s,\n", output.InvalidParameters[i])
		}
		return nil, fmt.Errorf("invalid parameters error: %s", invalidParamsStr)
	}

	// Build the parameter map that we're going to return
	paramMap := make(map[string]string)

	for _, item := range output.Parameters {
		paramMap[*item.Name] = *item.Value
	}

	return paramMap, nil
}
