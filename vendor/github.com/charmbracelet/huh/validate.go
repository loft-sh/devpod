package huh

import (
	"fmt"
	"unicode/utf8"
)

// ValidateNotEmpty checks if the input is not empty.
func ValidateNotEmpty() func(s string) error {
	return func(s string) error {
		if err := ValidateMinLength(1)(s); err != nil {
			return fmt.Errorf("input cannot be empty")
		}
		return nil
	}
}

// ValidateMinLength checks if the length of the input is at least min.
func ValidateMinLength(v int) func(s string) error {
	return func(s string) error {
		if utf8.RuneCountInString(s) < v {
			return fmt.Errorf("input must be at least %d characters long", v)
		}
		return nil
	}
}

// ValidateMaxLength checks if the length of the input is at most max.
func ValidateMaxLength(v int) func(s string) error {
	return func(s string) error {
		if utf8.RuneCountInString(s) > v {
			return fmt.Errorf("input must be at most %d characters long", v)
		}
		return nil
	}
}

// ValidateLength checks if the length of the input is within the specified range.
func ValidateLength(minl, maxl int) func(s string) error {
	return func(s string) error {
		if err := ValidateMinLength(minl)(s); err != nil {
			return err
		}
		return ValidateMaxLength(maxl)(s)
	}
}

// ValidateOneOf checks if a string is one of the specified options.
func ValidateOneOf(options ...string) func(string) error {
	validOptions := make(map[string]struct{})
	for _, option := range options {
		validOptions[option] = struct{}{}
	}

	return func(value string) error {
		if _, ok := validOptions[value]; !ok {
			return fmt.Errorf("invalid option: %s", value)
		}
		return nil
	}
}
