// Package validator wraps go-playground/validator with custom tag messages.
package validator

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	v    *validator.Validate
	once sync.Once
)

// Get returns the singleton validator instance.
func Get() *validator.Validate {
	once.Do(func() {
		v = validator.New()
	})
	return v
}

// Validate runs struct validation and returns a human-readable error string or nil.
func Validate(s interface{}) error {
	if err := Get().Struct(s); err != nil {
		return formatErrors(err.(validator.ValidationErrors))
	}
	return nil
}

// formatErrors converts validator.ValidationErrors to a single readable message.
func formatErrors(errs validator.ValidationErrors) error {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		msgs = append(msgs, fieldErr(e))
	}
	return fmt.Errorf("%s", strings.Join(msgs, "; "))
}

func fieldErr(e validator.FieldError) string {
	field := strings.ToLower(e.Field())
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, e.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	default:
		return fmt.Sprintf("%s is invalid (%s)", field, e.Tag())
	}
}
