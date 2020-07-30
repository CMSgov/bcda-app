// fhir package contains structs representing FHIR data.
// These data models are a lighter weight definition contain certain fields needed for BCDA
package fhir

import "time"

type Resource struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id"`
	Meta         struct {
		LastUpdated time.Time  `json:"lastUpdated"`
	} `json:"meta"`
	Total uint `json:"total"`
}

type Bundle struct {
	Resource
	Links []struct {
		Relation string `json:"relation"`
		URL      string `json:"url"`
	} `json:"link"`
	Entries []BundleEntry `json:"entry"`
}

type BundleEntry map[string]interface{}
