package models

import (
	"encoding/json"
	"strings"
)

func (r *Reference) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	if r.Reference != "" {
		m["reference"] = r.Reference
	}
	if r.Display != "" {
		m["display"] = r.Display
	}
	if r.Identifier != nil {
		m["identifier"] = r.Identifier
	}
	// TODO: validate spec rule: "At least one of reference, identifier and display SHALL be present (unless an extension is provided)."

	return json.Marshal(m)
}

type reference Reference

func (r *Reference) UnmarshalJSON(data []byte) (err error) {
	ref := reference{}
	if err = json.Unmarshal(data, &ref); err == nil {
		splitURL := strings.Split(ref.Reference, "/")
		if len(splitURL) >= 2 {
			ref.ReferencedID = splitURL[len(splitURL)-1]
			ref.Type = splitURL[len(splitURL)-2]
		}

		external := strings.HasPrefix(ref.Reference, "http")
		ref.External = &external

		*r = Reference(ref)
		return
	}
	return err
}
