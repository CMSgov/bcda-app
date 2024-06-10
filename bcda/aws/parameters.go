package bcdaaws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// Makes this easier to mock and unit test
var ssmNew = ssm.New
var ssmsvcGetParameter = (*ssm.SSM).GetParameter
var ssmsvcGetParameters = (*ssm.SSM).GetParameters

func GetParameter(s *session.Session, keyname string) (string, error) {
	ssmsvc := ssmNew(s)

	withDecryption := true
	result, err := ssmsvcGetParameter(ssmsvc, &ssm.GetParameterInput{
		Name:           &keyname,
		WithDecryption: &withDecryption,
	})

	if err != nil {
		return "", fmt.Errorf("Error retrieving parameter %s from parameter store: %w", keyname, err)
	}

	val := *result.Parameter.Value

	if val == "" {
		return "", fmt.Errorf("No parameter store value found for %s", keyname)
	}

	return val, nil
}

// Returns a list of parameters from the SSM Parameter Store
func GetParameters(s *session.Session, keynames []*string) (map[string]string, error) {
	// Create an SSM client and pull down keys from the param store
	ssmsvc := ssmNew(s)

	withDecryption := true
	params, err := ssmsvcGetParameters(ssmsvc, &ssm.GetParametersInput{
		Names:          keynames,
		WithDecryption: &withDecryption,
	})
	if err != nil {
		return nil, fmt.Errorf("error connecting to parameter store: %s", err)
	}

	// Unknown keys will come back as invalid, make sure we error on them
	if len(params.InvalidParameters) > 0 {
		invalidParamsStr := ""
		for i := 0; i < len(params.InvalidParameters); i++ {
			invalidParamsStr += fmt.Sprintf("%s,\n", *params.InvalidParameters[i])
		}
		return nil, fmt.Errorf("invalid parameters error: %s", invalidParamsStr)
	}

	// Build the parameter map that we're going to return
	var paramMap map[string]string = make(map[string]string)

	for _, item := range params.Parameters {
		paramMap[*item.Name] = *item.Value
	}
	return paramMap, nil
}
