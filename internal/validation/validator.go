package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validators for Ethereum-specific fields
	_ = validate.RegisterValidation("eth_address", validateEthAddress)
	_ = validate.RegisterValidation("validator_index", validateValidatorIndex)
	_ = validate.RegisterValidation("pubkey", validatePubkey)
}

// ValidateStruct validates any struct with validation tags
func ValidateStruct(ctx context.Context, s interface{}) error {
	if err := validate.StructCtx(ctx, s); err != nil {
		return FormatValidationError(err)
	}
	return nil
}

// FormatValidationError converts validator errors to user-friendly messages
func FormatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			messages = append(messages, formatFieldError(e))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(messages, "; "))
	}
	return err
}

func formatFieldError(e validator.FieldError) string {
	field := strings.ToLower(e.Field())

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, e.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "eth_address":
		return fmt.Sprintf("%s must be a valid Ethereum address", field)
	case "validator_index":
		return fmt.Sprintf("%s must be a valid validator index (>= 0)", field)
	case "pubkey":
		return fmt.Sprintf("%s must be a valid BLS public key (0x followed by 96 hex chars)", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "dive":
		return fmt.Sprintf("%s contains invalid items", field)
	case "gtefield":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, e.Param())
	default:
		return fmt.Sprintf("%s failed %s validation", field, e.Tag())
	}
}

// Custom validators for Ethereum-specific fields

// validateEthAddress validates Ethereum addresses (0x + 40 hex chars)
func validateEthAddress(fl validator.FieldLevel) bool {
	addr := fl.Field().String()
	if len(addr) != 42 {
		return false
	}
	if !strings.HasPrefix(addr, "0x") {
		return false
	}
	// Check hex chars
	for _, c := range addr[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// validateValidatorIndex validates validator indices (must be >= 0)
func validateValidatorIndex(fl validator.FieldLevel) bool {
	index := fl.Field().Int()
	return index >= 0
}

// validatePubkey validates BLS public keys (0x + 96 hex chars)
func validatePubkey(fl validator.FieldLevel) bool {
	pubkey := fl.Field().String()
	if len(pubkey) != 98 { // 0x + 96 hex chars
		return false
	}
	if !strings.HasPrefix(pubkey, "0x") {
		return false
	}
	// Check hex chars
	for _, c := range pubkey[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
