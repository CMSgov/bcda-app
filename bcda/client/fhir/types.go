package fhir

type TypeFilterParameter struct {
	ResourceType    string
	QueryParameters []TypeFilterSubqueryParam
}

type TypeFilterSubqueryParam struct {
	Name  string
	Value string
}
