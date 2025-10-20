package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStruct_RegisterInput(t *testing.T) {
	tests := []struct {
		name      string
		input     ValidatedRegisterInput
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid registration input",
			input: ValidatedRegisterInput{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "securepassword123",
			},
			wantError: false,
		},
		{
			name: "username too short",
			input: ValidatedRegisterInput{
				Username: "ab",
				Email:    "test@example.com",
				Password: "securepassword123",
			},
			wantError: true,
			errorMsg:  "username must be at least 3",
		},
		{
			name: "username with special chars",
			input: ValidatedRegisterInput{
				Username: "user@name",
				Email:    "test@example.com",
				Password: "securepassword123",
			},
			wantError: true,
			errorMsg:  "alphanum",
		},
		{
			name: "invalid email",
			input: ValidatedRegisterInput{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "securepassword123",
			},
			wantError: true,
			errorMsg:  "email must be a valid email address",
		},
		{
			name: "password too short",
			input: ValidatedRegisterInput{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "short",
			},
			wantError: true,
			errorMsg:  "password must be at least 8",
		},
		{
			name: "missing required fields",
			input: ValidatedRegisterInput{
				Username: "",
				Email:    "",
				Password: "",
			},
			wantError: true,
			errorMsg:  "is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(context.Background(), tt.input)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStruct_LoginInput(t *testing.T) {
	tests := []struct {
		name      string
		input     ValidatedLoginInput
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid login input",
			input: ValidatedLoginInput{
				Username: "testuser",
				Password: "password",
			},
			wantError: false,
		},
		{
			name: "empty username",
			input: ValidatedLoginInput{
				Username: "",
				Password: "password",
			},
			wantError: true,
			errorMsg:  "username is required",
		},
		{
			name: "empty password",
			input: ValidatedLoginInput{
				Username: "testuser",
				Password: "",
			},
			wantError: true,
			errorMsg:  "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(context.Background(), tt.input)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEthAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		valid   bool
	}{
		{"valid lowercase address", "0x1234567890123456789012345678901234567890", true},
		{"valid uppercase address", "0xABCDEF1234567890ABCDEF1234567890ABCDEF12", true},
		{"valid mixed case", "0xAbCdEf1234567890aBcDeF1234567890AbCdEf12", true},
		{"too short", "0x12345", false},
		{"too long", "0x12345678901234567890123456789012345678901", false},
		{"missing 0x prefix", "1234567890123456789012345678901234567890", false},
		{"invalid hex chars", "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type addr struct {
				Address string `validate:"eth_address"`
			}

			err := ValidateStruct(context.Background(), addr{Address: tt.address})

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidatePubkey(t *testing.T) {
	// Create valid 96-character hex strings (BLS pubkeys are 48 bytes = 96 hex chars)
	// 0x + 96 hex chars = 98 total
	validPubkey := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	validUppercase := "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	invalidPubkeyWithG := "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG"

	tests := []struct {
		name   string
		pubkey string
		valid  bool
	}{
		{"valid pubkey", validPubkey, true},
		{"valid uppercase", validUppercase, true},
		{"too short", "0xabc", false},
		{"missing 0x", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"invalid chars", invalidPubkeyWithG, false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type pk struct {
				Pubkey string `validate:"pubkey"`
			}

			err := ValidateStruct(context.Background(), pk{Pubkey: tt.pubkey})

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateValidatorIndex(t *testing.T) {
	tests := []struct {
		name  string
		index int64
		valid bool
	}{
		{"zero index", 0, true},
		{"positive index", 12345, true},
		{"large index", 1000000, true},
		{"negative index", -1, false},
		{"large negative", -999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type vi struct {
				Index int64 `validate:"validator_index"`
			}

			err := ValidateStruct(context.Background(), vi{Index: tt.index})

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateStruct_ValidatorQueryInput(t *testing.T) {
	validIndex := 12345
	invalidIndex := -1
	// Create a valid 96-char hex pubkey (BLS pubkeys are 48 bytes = 96 hex chars)
	// 0x + 96 hex chars = 98 total
	validPubkey := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	invalidPubkey := "invalid"
	validLimit := 50
	invalidLimit := 200
	validOffset := 10
	invalidOffset := -5

	tests := []struct {
		name      string
		input     ValidatedValidatorQueryInput
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid with validator index",
			input: ValidatedValidatorQueryInput{
				ValidatorIndex: &validIndex,
				Limit:          &validLimit,
				Offset:         &validOffset,
			},
			wantError: false,
		},
		{
			name: "valid with pubkey",
			input: ValidatedValidatorQueryInput{
				PublicKey: &validPubkey,
				Limit:     &validLimit,
			},
			wantError: false,
		},
		{
			name: "invalid validator index",
			input: ValidatedValidatorQueryInput{
				ValidatorIndex: &invalidIndex,
			},
			wantError: true,
			errorMsg:  "validatorindex must be a valid validator index",
		},
		{
			name: "invalid pubkey",
			input: ValidatedValidatorQueryInput{
				PublicKey: &invalidPubkey,
			},
			wantError: true,
			errorMsg:  "publickey must be a valid BLS public key",
		},
		{
			name: "limit too large",
			input: ValidatedValidatorQueryInput{
				ValidatorIndex: &validIndex,
				Limit:          &invalidLimit,
			},
			wantError: true,
			errorMsg:  "limit must be at most 100",
		},
		{
			name: "negative offset",
			input: ValidatedValidatorQueryInput{
				ValidatorIndex: &validIndex,
				Offset:         &invalidOffset,
			},
			wantError: true,
			errorMsg:  "offset must be at least 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(context.Background(), tt.input)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatFieldError(t *testing.T) {
	// This tests the error message formatting
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{
			name: "required field",
			input: struct {
				Field string `validate:"required"`
			}{},
			want: "field is required",
		},
		{
			name: "min length",
			input: struct {
				Field string `validate:"min=5"`
			}{Field: "abc"},
			want: "field must be at least 5",
		},
		{
			name: "max length",
			input: struct {
				Field string `validate:"max=10"`
			}{Field: "this is way too long"},
			want: "field must be at most 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(context.Background(), tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}
