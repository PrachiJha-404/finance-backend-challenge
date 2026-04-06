package validator

import (
	"fmt"
	"net/mail"
	"strings"
)

type Validator struct {
	errors []string
}

func New() *Validator {
	return &Validator{}
}

func (v *Validator) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, fmt.Sprintf("%s is required", field))
	}
}

func (v *Validator) MinLength(field, value string, min int) {
	if len(value) < min {
		v.errors = append(v.errors, fmt.Sprintf("%s must be at least %d characters", field, min))
	}
}

func (v *Validator) IsEmail(field, value string) {
	_, err := mail.ParseAddress(value)
	if err != nil {
		v.errors = append(v.errors, fmt.Sprintf("%s must be a valid email", field))
	}
}

func (v *Validator) OneOf(field, value string, options ...string) {
	for _, opt := range options {
		if value == opt {
			return
		}
	}
	v.errors = append(v.errors, fmt.Sprintf("%s must be one of: %s", field, strings.Join(options, ", ")))
}

func (v *Validator) GreaterThan(field string, value, min float64) {
	if value <= min {
		v.errors = append(v.errors, fmt.Sprintf("%s must be greater than %v", field, min))
	}
}

func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *Validator) Error() string {
	return strings.Join(v.errors, "; ")
}
