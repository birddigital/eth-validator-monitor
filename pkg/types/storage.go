package types

import (
	"context"
	"time"
)

// Storage defines the interface for persistent storage operations
type Storage interface {
	// Validator operations
	AddValidator(ctx context.Context, v *Validator) error
	GetValidator(ctx context.Context, index int) (*Validator, error)
	GetValidatorByPubkey(ctx context.Context, pubkey string) (*Validator, error)
	ListValidators(ctx context.Context, filter ValidatorFilter) ([]*Validator, error)
	UpdateValidator(ctx context.Context, v *Validator) error
	DeleteValidator(ctx context.Context, index int) error

	// Snapshot operations
	SaveSnapshot(ctx context.Context, snapshot *ValidatorSnapshot) error
	GetSnapshots(ctx context.Context, validatorIndex int, from, to time.Time) ([]*ValidatorSnapshot, error)
	GetLatestSnapshot(ctx context.Context, validatorIndex int) (*ValidatorSnapshot, error)
	GetSnapshotsByEpoch(ctx context.Context, epoch int) ([]*ValidatorSnapshot, error)

	// Performance operations
	SavePerformance(ctx context.Context, perf *ValidatorPerformance) error
	GetPerformance(ctx context.Context, validatorIndex int) (*ValidatorPerformance, error)

	// Alert operations
	SaveAlert(ctx context.Context, alert *Alert) error
	GetAlerts(ctx context.Context, filter AlertFilter) ([]*Alert, error)
	GetAlert(ctx context.Context, id int64) (*Alert, error)
	AcknowledgeAlert(ctx context.Context, id int64) error

	// Health check
	Ping(ctx context.Context) error
}

// Cache defines the interface for caching layer operations
type Cache interface {
	// Validator state caching
	GetValidatorState(ctx context.Context, index int) (*ValidatorData, error)
	SetValidatorState(ctx context.Context, index int, state *ValidatorData, ttl time.Duration) error
	Invalidate(ctx context.Context, index int) error

	// Snapshot caching
	GetLatestSnapshot(ctx context.Context, index int) (*ValidatorSnapshot, error)
	SetLatestSnapshot(ctx context.Context, index int, snapshot *ValidatorSnapshot, ttl time.Duration) error

	// Network stats caching
	GetNetworkStats(ctx context.Context) (*NetworkStats, error)
	SetNetworkStats(ctx context.Context, stats *NetworkStats, ttl time.Duration) error

	// Health check
	Ping(ctx context.Context) error
}
