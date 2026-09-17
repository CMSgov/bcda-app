package search

import (
	"fmt"
	"net/url"
	"strings"
)

type TypeFilterSubquery struct {
	ResourceType    string
	QueryParameters []TypeFilterSubqueryParam
}

type TypeFilterSubqueryParam struct {
	Name  string
	Value string
}

func ParseTypeFilterSubquery(s string) (TypeFilterSubquery, error) {
	// The subquery is url-encoded. So we will first decode so we can parse it
	decodedQuery, err := url.QueryUnescape(s)
	if err != nil {
		return TypeFilterSubquery{}, fmt.Errorf("failed to unescape %s", s)
	}

	// Expected format is: <resourceType>?<paramList>
	resourceType, params, ok := strings.Cut(decodedQuery, "?")
	if !ok {
		return TypeFilterSubquery{}, fmt.Errorf("missing question mark %s", decodedQuery)
	}

	var subqueryParams []TypeFilterSubqueryParam
	paramAry := strings.SplitSeq(params, "&")
	for paramPair := range paramAry {
		name, value, ok := strings.Cut(paramPair, "=")
		if !ok {
			return TypeFilterSubquery{}, fmt.Errorf("invalid _typeFilter parameter/value: %s", paramPair)
		}
		subqueryParams = append(subqueryParams, TypeFilterSubqueryParam{Name: name, Value: value})
	}
	return TypeFilterSubquery{ResourceType: resourceType, QueryParameters: subqueryParams}, nil
}

func ValidateTypeFilterSubquery(subquery TypeFilterSubquery) error {
	if subquery.ResourceType != "ExplanationOfBenefit" {
		return fmt.Errorf("invalid _typeFilter Resource Type (Only EOBs valid): %s", subquery.ResourceType)
	}

	var serviceDateParams []DateParam
	for _, param := range subquery.QueryParameters {
		switch param.Name {
		case TAG:
			tag, err := ParseTokenParam(param)
			if err != nil {
				return err
			}
			err = ValidateTag(tag)
			if err != nil {
				return err
			}
		case SERVICE_DATE:
			date, err := ParseDateParam(param)
			if err != nil {
				return err
			}
			serviceDateParams = append(serviceDateParams, date)
		case OUTCOME:
			outcome, err := ParseStringParam(param)
			if err != nil {
				return err
			}
			err = ValidateOutcome(outcome)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid _typeFilter subquery parameter: %s", param.Name)
		}
	}
	err := ValidateServiceDates(serviceDateParams)
	if err != nil {
		return err
	}
	return nil
}
