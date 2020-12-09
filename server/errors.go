package server

import (
	"fmt"
	"strings"
)

const (
	// PropertyMissing should be used when a required request body property is
	// missing
	PropertyMissing = "MISSING"

	// PropertyForbidden should be used when a request property must not be specified
	PropertyForbidden = "FORBIDDEN"

	// PropertyTaken should be used when the value of a request property should be unique
	// it a given scope but is already used
	PropertyTaken = "TAKEN"

	// PropertyInvalid should be used if the format of a request body property is invalid
	PropertyInvalid = "INVALID_FORMAT"
)

type (

	// PropertyError represents an error condition that encountered
	// on a request body property
	PropertyError struct {
		// Field is the name of the property that caused the error
		Field string `json:"field"`

		// Description is the description of the error
		Description string `json:"message"`
	}

	// ValidationError is a common request body validation error
	ValidationError struct {
		// Fields holds all field/property errors
		Fields []*PropertyError `json:"errors"`
	}
)

// Error implemenets the error interface
func (p *PropertyError) Error() string {
	return fmt.Sprintf("%s: %s", p.Field, p.Description)
}

// Add adds a new property validation error to v
func (v *ValidationError) Add(field, message string) {
	v.Fields = append(v.Fields, &PropertyError{
		Field:       field,
		Description: message,
	})
}

// Error implements the error interface
func (v *ValidationError) Error() string {
	s := make([]string, len(v.Fields))
	for i, e := range v.Fields {
		s[i] = e.Error()
	}

	return strings.Join(s, "; ")
}

// Build returns v itself if it contains property errors.
// If no errors have been added nil is returned
func (v *ValidationError) Build() error {
	if len(v.Fields) == 0 {
		return nil
	}

	return v
}

// AddMissing adds a PropertyMissing error for p
func (v *ValidationError) AddMissing(p string) {
	v.Add(p, PropertyMissing)
}

// AddTaken adds a PropertyTaken error for p
func (v *ValidationError) AddTaken(p string) {
	v.Add(p, PropertyTaken)
}

// AddInvalid adds a PropertyInvalid error for p
func (v *ValidationError) AddInvalid(p string) {
	v.Add(p, PropertyInvalid)
}

// AddForbidden adds a PropertyForbidden error for p
func (v *ValidationError) AddForbidden(p string) {
	v.Add(p, PropertyForbidden)
}
