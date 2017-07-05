package minions

import (
	"encoding/json"
)

// BindingResult holds validation errors of the binding process from a HTML
// form to a Go struct.
type BindingResult map[string]string

// Valid returns whether the binding was successfull or not.
func (br BindingResult) Valid() bool {
	return len(br) == 0
}

// Fail marks the binding as failed and stores an error for the given field
// that caused the form binding to fail.
func (br BindingResult) Fail(field, err string) {
	br[field] = err
}

// Include copies all errors and state of a binding result
func (br BindingResult) Include(other BindingResult) {
	for field, err := range other {
		br.Fail(field, err)
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (br BindingResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string(br))
}

// V is a helper type to quickly build variable maps for templates.
type V map[string]interface{}

// MarshalJSON implements the json.Marshaler interface.
func (v V) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}(v))
}
