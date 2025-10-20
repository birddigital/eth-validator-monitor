package auth

import (
	"regexp"
	"strings"
	"unicode"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidationError represents validation failures with field-specific errors
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

// NewValidationError creates a new validation error
func NewValidationError() *ValidationError {
	return &ValidationError{
		Fields: make(map[string]string),
	}
}

// AddError adds a field error to the validation error
func (e *ValidationError) AddError(field, message string) {
	e.Fields[field] = message
}

// HasErrors returns true if there are any validation errors
func (e *ValidationError) HasErrors() bool {
	return len(e.Fields) > 0
}

// Validator handles authentication-related validations
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateEmail checks if email format is valid
func (v *Validator) ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		verr := NewValidationError()
		verr.AddError("email", "email is required")
		return verr
	}
	if !emailRegex.MatchString(email) {
		verr := NewValidationError()
		verr.AddError("email", "invalid email format")
		return verr
	}
	return nil
}

// ValidatePassword checks password strength
// Requirements: min 8 chars, at least 1 uppercase, 1 lowercase, 1 number, 1 special char
func (v *Validator) ValidatePassword(password string) error {
	verr := NewValidationError()

	if len(password) < MinPasswordLength {
		verr.AddError("password", "password must be at least 8 characters long")
		return verr
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		verr.AddError("password", "password must contain uppercase, lowercase, number, and special character")
		return verr
	}

	return nil
}

// ValidatePasswordMatch checks if passwords match
func (v *Validator) ValidatePasswordMatch(password, confirm string) error {
	if password != confirm {
		verr := NewValidationError()
		verr.AddError("confirmPassword", "passwords do not match")
		return verr
	}
	return nil
}

// ValidateUsername checks if username is valid
func (v *Validator) ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		verr := NewValidationError()
		verr.AddError("username", "username is required")
		return verr
	}
	if len(username) < 3 {
		verr := NewValidationError()
		verr.AddError("username", "username must be at least 3 characters long")
		return verr
	}
	if len(username) > 255 {
		verr := NewValidationError()
		verr.AddError("username", "username must not exceed 255 characters")
		return verr
	}
	return nil
}

// ValidateRegistration validates all registration fields
func (v *Validator) ValidateRegistration(username, email, password, confirmPassword string) error {
	verr := NewValidationError()

	// Validate username
	if err := v.ValidateUsername(username); err != nil {
		if ve, ok := err.(*ValidationError); ok {
			for k, msg := range ve.Fields {
				verr.AddError(k, msg)
			}
		}
	}

	// Validate email
	if err := v.ValidateEmail(email); err != nil {
		if ve, ok := err.(*ValidationError); ok {
			for k, msg := range ve.Fields {
				verr.AddError(k, msg)
			}
		}
	}

	// Validate password strength
	if err := v.ValidatePassword(password); err != nil {
		if ve, ok := err.(*ValidationError); ok {
			for k, msg := range ve.Fields {
				verr.AddError(k, msg)
			}
		}
	}

	// Validate password match
	if err := v.ValidatePasswordMatch(password, confirmPassword); err != nil {
		if ve, ok := err.(*ValidationError); ok {
			for k, msg := range ve.Fields {
				verr.AddError(k, msg)
			}
		}
	}

	if verr.HasErrors() {
		return verr
	}

	return nil
}

// ValidateLogin validates login fields
func (v *Validator) ValidateLogin(username, password string) error {
	verr := NewValidationError()

	username = strings.TrimSpace(username)
	if username == "" {
		verr.AddError("username", "username is required")
	}

	if password == "" {
		verr.AddError("password", "password is required")
	}

	if verr.HasErrors() {
		return verr
	}

	return nil
}
