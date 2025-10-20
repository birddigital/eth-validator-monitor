package resolver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/birddigital/eth-validator-monitor/graph/model"
	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/storage"
	"github.com/birddigital/eth-validator-monitor/internal/validation"
	"github.com/birddigital/eth-validator-monitor/pkg/types"
	"github.com/google/uuid"
)

// Register handles user registration
func (r *mutationResolver) Register(ctx context.Context, input model.RegisterInput) (*model.AuthPayload, error) {
	// Validate input using validation package
	validatedInput := validation.ValidatedRegisterInput{
		Username: input.Username,
		Email:    input.Email,
		Password: input.Password,
	}
	if err := validation.ValidateStruct(ctx, validatedInput); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user with default role
	user, err := r.UserRepo.CreateUser(ctx, input.Username, input.Email, hashedPassword, []string{"user"})
	if err != nil {
		if errors.Is(err, storage.ErrUserAlreadyExists) {
			return nil, errors.New("username or email already exists")
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, err := r.JWTService.GenerateAccessToken(user.ID.String(), user.Username, user.Roles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := r.JWTService.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Calculate expiration
	expiresAt := time.Now().Add(r.Config.JWT.AccessTokenDuration).Unix()

	return &model.AuthPayload{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         mapUserToModel(user),
		ExpiresAt:    int(expiresAt),
	}, nil
}

// Login handles user authentication
func (r *mutationResolver) Login(ctx context.Context, input model.LoginInput) (*model.AuthPayload, error) {
	// Validate input using validation package
	validatedInput := validation.ValidatedLoginInput{
		Username: input.Username,
		Password: input.Password,
	}
	if err := validation.ValidateStruct(ctx, validatedInput); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Retrieve user
	user, err := r.UserRepo.GetUserByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, errors.New("invalid username or password")
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Verify password
	if err := auth.VerifyPassword(user.PasswordHash, input.Password); err != nil {
		return nil, errors.New("invalid username or password")
	}

	// Update last login
	if err := r.UserRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Log error but don't fail the login
		r.Logger.Warn().Str("user_id", user.ID.String()).Err(err).Msg("failed to update last login")
	}

	// Generate tokens
	accessToken, err := r.JWTService.GenerateAccessToken(user.ID.String(), user.Username, user.Roles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := r.JWTService.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Calculate expiration
	expiresAt := time.Now().Add(r.Config.JWT.AccessTokenDuration).Unix()

	return &model.AuthPayload{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         mapUserToModel(user),
		ExpiresAt:    int(expiresAt),
	}, nil
}

// RefreshToken handles token refresh
func (r *mutationResolver) RefreshToken(ctx context.Context, refreshToken string) (*model.AuthPayload, error) {
	// Validate refresh token
	claims, err := r.JWTService.ValidateToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	// Parse user ID
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, errors.New("invalid user ID in token")
	}

	// Retrieve user to get latest roles
	user, err := r.UserRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Generate new tokens
	newAccessToken, err := r.JWTService.GenerateAccessToken(user.ID.String(), user.Username, user.Roles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := r.JWTService.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Calculate expiration
	expiresAt := time.Now().Add(r.Config.JWT.AccessTokenDuration).Unix()

	return &model.AuthPayload{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User:         mapUserToModel(user),
		ExpiresAt:    int(expiresAt),
	}, nil
}

// Me returns the current authenticated user
func (r *queryResolver) Me(ctx context.Context) (*model.User, error) {
	claims, err := auth.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	user, err := r.UserRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return mapUserToModel(user), nil
}

// Helper function to map storage.User to model.User
func mapUserToModel(user *storage.User) *model.User {
	var lastLogin *types.Time
	if user.LastLogin != nil {
		t := types.Time(*user.LastLogin)
		lastLogin = &t
	}

	return &model.User{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		Roles:     user.Roles,
		CreatedAt: types.Time(user.CreatedAt),
		LastLogin: lastLogin,
	}
}
