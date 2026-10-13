package schema

import (
	"encoding/json"
	"reflect"
	"strings"
)

// FromStruct builds a JSON Schema object for a struct type.
func FromStruct(v interface{}) json.RawMessage {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	obj := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
	props := obj["properties"].(map[string]interface{})
	var required []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name := f.Tag.Get("json")
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		if idx := strings.Index(name, ","); idx >= 0 {
			name = name[:idx]
		}
		if name == "-" {
			continue
		}
		props[name] = fieldSchema(f.Type)
		if tag := f.Tag.Get("jsonschema"); strings.Contains(tag, "required") {
			required = append(required, name)
		}
	}
	if len(required) > 0 {
		obj["required"] = required
	}
	b, _ := json.Marshal(obj)
	return b
}

func fieldSchema(t reflect.Type) map[string]interface{} {
	switch t.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Int, reflect.Int64, reflect.Int32:
		return map[string]interface{}{"type": "integer"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Slice:
		return map[string]interface{}{
			"type":  "array",
			"items": fieldSchema(t.Elem()),
		}
	case reflect.Map, reflect.Struct:
		return map[string]interface{}{"type": "object"}
	default:
		return map[string]interface{}{}
	}
}

// ValidateRequired checks top-level required keys exist in raw JSON.
func ValidateRequired(params json.RawMessage, schema json.RawMessage) error {
	var sm map[string]interface{}
	if err := json.Unmarshal(schema, &sm); err != nil {
		return err
	}
	req, _ := sm["required"].([]interface{})
	var args map[string]interface{}
	if err := json.Unmarshal(params, &args); err != nil {
		return err
	}
	for _, r := range req {
		key, _ := r.(string)
		if _, ok := args[key]; !ok {
			return &ValidationError{Field: key, Msg: "required field missing"}
		}
	}
	return nil
}

// ValidationError is a schema validation failure.
type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Msg
	}
	return e.Msg
}
