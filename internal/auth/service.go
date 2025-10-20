package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/birddigital/eth-validator-monitor/internal/storage"
)

var (
	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrUserAlreadyExists is returned when attempting to create a duplicate user
	ErrUserAlreadyExists = errors.New("user already exists")
)

// Service handles authentication business logic
type Service struct {
	userRepo  *storage.UserRepository
	validator *Validator
}

// NewService creates a new authentication service
func NewService(userRepo *storage.UserRepository) *Service {
	return &Service{
		userRepo:  userRepo,
		validator: NewValidator(),
	}
}

// Register creates a new user account with hashed password
func (s *Service) Register(ctx context.Context, username, password, confirmPassword, email string, roles []string) (*storage.User, error) {
	// Validate all registration fields
	if err := s.validator.ValidateRegistration(username, email, password, confirmPassword); err != nil {
		return nil, err // Returns *ValidationError with field-level errors
	}

	// Check email uniqueness
	existingUser, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, storage.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Check username uniqueness
	existingUser, err = s.userRepo.GetUserByUsername(ctx, username)
	if err != nil && !errors.Is(err, storage.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check username uniqueness: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Hash password using existing utility
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Default roles if none provided
	if roles == nil {
		roles = []string{"user"}
	}

	// Create user via repository
	user, err := s.userRepo.CreateUser(ctx, username, email, hashedPassword, roles)
	if err != nil {
		if errors.Is(err, storage.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login verifies credentials and returns user if valid
func (s *Service) Login(ctx context.Context, username, password string) (*storage.User, error) {
	// Validate login fields
	if err := s.validator.ValidateLogin(username, password); err != nil {
		return nil, err // Returns *ValidationError with field-level errors
	}

	// Fetch user by username
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Verify password
	if err := VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login timestamp
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Log error but don't fail login
		// In production, use proper logger
		fmt.Printf("Warning: failed to update last login for user %s: %v\n", user.ID, err)
	}

	return user, nil
}

// LoginByEmail verifies credentials using email and returns user if valid
func (s *Service) LoginByEmail(ctx context.Context, email, password string) (*storage.User, error) {
	// Fetch user by email
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Verify password
	if err := VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login timestamp
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Log error but don't fail login
		// In production, use proper logger
		fmt.Printf("Warning: failed to update last login for user %s: %v\n", user.ID, err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*storage.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ChangePassword updates a user's password
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// Fetch user
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify old password
	if err := VerifyPassword(user.PasswordHash, oldPassword); err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password in database
	// Note: UserRepository doesn't have UpdatePassword method yet
	// This would need to be added to storage.UserRepository
	// For now, we'll note this as a TODO
	_ = newHash
	return fmt.Errorf("password update not yet implemented in repository")
}
