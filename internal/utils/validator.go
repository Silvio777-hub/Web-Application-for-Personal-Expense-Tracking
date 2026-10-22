package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
)

// Validator facilitates easy request data validation.
type Validator struct {
	Errors map[string]string
}

func NewValidator() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

func (v *Validator) Error() error {
	if v.Valid() {
		return nil
	}
	var sb strings.Builder
	for k, m := range v.Errors {
		sb.WriteString(fmt.Sprintf("%s: %s; ", k, m))
	}
	return errors.New(strings.TrimSuffix(sb.String(), "; "))
}

// Common Validation Helpers

func (v *Validator) Required(val, key string) {
	v.Check(strings.TrimSpace(val) != "", key, "this field is required")
}

func (v *Validator) MinLength(val, key string, min int) {
	v.Check(len(val) >= min, key, fmt.Sprintf("must be at least %d characters", min))
}

func (v *Validator) MaxLength(val, key string, max int) {
	v.Check(len(val) <= max, key, fmt.Sprintf("must be no more than %d characters", max))
}

func (v *Validator) Email(val, key string) {
	v.Check(emailRegex.MatchString(strings.ToLower(val)), key, "must be a valid email address")
}

func (v *Validator) Range(val float64, key string, min, max float64) {
	v.Check(val >= min && val <= max, key, fmt.Sprintf("must be between %v and %v", min, max))
}
