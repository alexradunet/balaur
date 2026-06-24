package nodes

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// PropType is the value-type token for a property definition.
type PropType string

const (
	PropText   PropType = "text"
	PropNumber PropType = "number"
	PropDate   PropType = "date" // RFC3339 string
	PropBool   PropType = "bool"
	PropSelect PropType = "select" // value must be one of Options
)

// PropDef describes one typed property for a node type.
type PropDef struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     PropType `json:"type"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

// ValidateProps checks props against the schema defined by defs.
//
// Rules:
//   - Every Required def must have a non-empty value in props.
//   - Every present value whose key matches a def must satisfy the def's type.
//   - Unknown keys are allowed (forward-compatible; internal code adds extras
//     like use_count/last_used without touching the schema).
//   - An empty defs slice accepts any props.
func ValidateProps(defs []PropDef, props map[string]any) error {
	if len(defs) == 0 {
		return nil
	}
	defByKey := make(map[string]PropDef, len(defs))
	for _, d := range defs {
		defByKey[d.Key] = d
	}

	for _, d := range defs {
		v, present := props[d.Key]
		if d.Required {
			if !present || isEmpty(v) {
				return fmt.Errorf("required property %q is missing or empty", d.Key)
			}
		}
		if present && !isEmpty(v) {
			if err := checkType(d, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// TypeSchema loads the property schema for typ from the node_types registry.
// Returns an empty slice (not nil) when the type has no schema.
func TypeSchema(app core.App, typ string) ([]PropDef, error) {
	rec, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": typ})
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("node_types: loading schema for %q: %w", typ, err)
	}
	raw := rec.GetString("properties")
	if raw == "" {
		return nil, nil
	}
	var defs []PropDef
	if err := json.Unmarshal([]byte(raw), &defs); err != nil {
		return nil, fmt.Errorf("node_types: parsing schema for %q: %w", typ, err)
	}
	return defs, nil
}

// TypeTemplate loads the template map for typ from the node_types registry.
// Returns nil when the type has no template. The reserved key "_body" is a
// default node body; all other keys are default prop values.
func TypeTemplate(app core.App, typ string) (map[string]any, error) {
	rec, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": typ})
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("node_types: loading template for %q: %w", typ, err)
	}
	raw := rec.GetString("template")
	if raw == "" {
		return nil, nil
	}
	var tmpl map[string]any
	if err := json.Unmarshal([]byte(raw), &tmpl); err != nil {
		return nil, fmt.Errorf("node_types: parsing template for %q: %w", typ, err)
	}
	return tmpl, nil
}

// ApplyTemplate fills missing prop keys from tmpl and optionally provides a
// default body. It is a pure function: the input maps are not mutated.
//
//   - If body is empty and tmpl contains "_body" (string), the template body is used.
//   - For every key in tmpl except "_body", if props does not already carry that
//     key, the template value is added.
func ApplyTemplate(tmpl map[string]any, body string, props map[string]any) (string, map[string]any) {
	if len(tmpl) == 0 {
		return body, props
	}
	outProps := make(map[string]any, len(props))
	for k, v := range props {
		outProps[k] = v
	}
	outBody := body
	for k, v := range tmpl {
		if k == "_body" {
			if outBody == "" {
				if s, ok := v.(string); ok {
					outBody = s
				}
			}
			continue
		}
		if _, exists := outProps[k]; !exists {
			outProps[k] = v
		}
	}
	return outBody, outProps
}

// isEmpty reports whether a prop value is absent or zero-ish for the purpose
// of required-field checking: nil, empty string, zero float64/int are empty.
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return x == ""
	case float64:
		return false // 0 is a valid number
	case int:
		return false
	case bool:
		return false
	}
	return false
}

// checkType validates v against the type constraint in d, naming the key in
// any returned error.
func checkType(d PropDef, v any) error {
	switch d.Type {
	case PropText:
		if _, ok := v.(string); !ok {
			return fmt.Errorf("property %q: expected text (string), got %T", d.Key, v)
		}
	case PropNumber:
		switch v.(type) {
		case float64, int, int64, float32:
			// ok
		default:
			return fmt.Errorf("property %q: expected number, got %T", d.Key, v)
		}
	case PropBool:
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("property %q: expected bool, got %T", d.Key, v)
		}
	case PropDate:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("property %q: expected RFC3339 date string, got %T", d.Key, v)
		}
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			return fmt.Errorf("property %q: invalid RFC3339 date %q: %w", d.Key, s, err)
		}
	case PropSelect:
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("property %q: expected string (select), got %T", d.Key, v)
		}
		for _, opt := range d.Options {
			if s == opt {
				return nil
			}
		}
		return fmt.Errorf("property %q: value %q not in allowed options %v", d.Key, s, d.Options)
	}
	return nil
}
