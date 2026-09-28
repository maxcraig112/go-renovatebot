package schemagen

import "encoding/json"

// stripSelfReferences removes "$ref": "#" entries that appear alongside
// other sibling keywords (e.g. "default", "items") in Renovate's generated
// schema. Renovate emits these on every custom manager's array field as an
// inert decoration; the sibling "items" keyword already carries the real
// type. go-jsonschema does not support resolving a root self-reference, so
// these are stripped before generation rather than worked around downstream.
func stripSelfReferences(schemaJSON []byte) ([]byte, error) {
	var doc interface{}
	if err := json.Unmarshal(schemaJSON, &doc); err != nil {
		return nil, err
	}

	doc = stripSelfRefValue(doc)

	return json.Marshal(doc)
}

func stripSelfRefValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		if len(t) > 1 {
			if ref, ok := t["$ref"].(string); ok && ref == "#" {
				delete(t, "$ref")
			}
		}
		for k, val := range t {
			t[k] = stripSelfRefValue(val)
		}
		return t
	case []interface{}:
		for i, val := range t {
			t[i] = stripSelfRefValue(val)
		}
		return t
	default:
		return v
	}
}
