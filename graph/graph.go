package graph

import (
	"github.com/birddigital/eth-validator-monitor/graph/generated"
	"github.com/birddigital/eth-validator-monitor/graph/resolver"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Re-export types and functions from generated package
type (
	Config = generated.Config
)

var (
	NewExecutableSchema = generated.NewExecutableSchema
)

// NewResolver creates a new GraphQL resolver with database pool
// Note: Cache is optional and will be nil if Redis is not configured
func NewResolver(pool *pgxpool.Pool) *resolver.Resolver {
	return &resolver.Resolver{
		DB:              pool,
		ValidatorRepo:   repository.NewValidatorRepository(pool),
		SnapshotRepo:    repository.NewSnapshotRepository(pool),
		AlertRepo:       repository.NewAlertRepository(pool),
		PerformanceRepo: repository.NewPerformanceRepository(pool),
		Cache:           nil, // Cache initialization requires Redis config
	}
}
