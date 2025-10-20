package validation

// ValidatedRegisterInput wraps RegisterInput with validation tags
type ValidatedRegisterInput struct {
	Username string `validate:"required,min=3,max=50,alphanum"`
	Email    string `validate:"required,email,max=255"`
	Password string `validate:"required,min=8,max=72"` // bcrypt max is 72
}

// ValidatedLoginInput wraps LoginInput with validation tags
type ValidatedLoginInput struct {
	Username string `validate:"required,min=1"`
	Password string `validate:"required,min=1"` // Don't leak password requirements on login
}

// ValidatedValidatorQueryInput wraps ValidatorQuery with validation tags
type ValidatedValidatorQueryInput struct {
	ValidatorIndex *int    `validate:"omitempty,validator_index"`
	PublicKey      *string `validate:"omitempty,pubkey"`
	Limit          *int    `validate:"omitempty,min=1,max=100"`
	Offset         *int    `validate:"omitempty,min=0"`
}

// ValidatedAddValidatorInput wraps AddValidatorInput with validation tags
type ValidatedAddValidatorInput struct {
	Pubkey *string `validate:"omitempty,pubkey"`
	Index  *int    `validate:"omitempty,validator_index"`
	Name   *string `validate:"omitempty,min=1,max=100"`
}

// ValidatedPerformanceQueryInput wraps performance query inputs
type ValidatedPerformanceQueryInput struct {
	ValidatorIndex int    `validate:"required,validator_index"`
	EpochFrom      *int64 `validate:"omitempty,min=0"`
	EpochTo        *int64 `validate:"omitempty,min=0,gtefield=EpochFrom"`
}

// ValidatedPaginationInput wraps pagination inputs
type ValidatedPaginationInput struct {
	Limit  *int    `validate:"omitempty,min=1,max=100"`
	Cursor *string `validate:"omitempty"`
}
