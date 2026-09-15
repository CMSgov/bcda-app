package typefilter

import (
	"fmt"
	"net/url"
	"strings"
)

type Subquery struct {
	ResourceType    string
	QueryParameters []SubqueryParam
}

type SubqueryParam struct {
	Name  string
	Value string
}

func ParseTypeFilterSubquery(s string) (Subquery, error) {
	// The subquery is url-encoded. So we will first decode so we can parse it
	decodedQuery, err := url.QueryUnescape(s)
	if err != nil {
		return Subquery{}, fmt.Errorf("failed to unescape %s", s)
	}

	// Expected format is: <resourceType>?<paramList>
	resourceType, params, ok := strings.Cut(decodedQuery, "?")
	if !ok {
		return Subquery{}, fmt.Errorf("missing question mark %s", decodedQuery)
	}

	var subqueryParams []SubqueryParam
	paramAry := strings.SplitSeq(params, "&")
	for paramPair := range paramAry {
		name, value, ok := strings.Cut(paramPair, "=")
		if !ok {
			return Subquery{}, fmt.Errorf("invalid _typeFilter parameter/value: %s", paramPair)
		}
		subqueryParams = append(subqueryParams, SubqueryParam{Name: name, Value: value})
	}
	return Subquery{ResourceType: resourceType, QueryParameters: subqueryParams}, nil
}
