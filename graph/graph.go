package graph

import (
	"github.com/birddigital/eth-validator-monitor/graph/generated"
	"github.com/birddigital/eth-validator-monitor/graph/resolver"
	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/config"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
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

// NewResolverWithAuth creates a new GraphQL resolver with auth support
func NewResolverWithAuth(
	pool *pgxpool.Pool,
	userRepo *storage.UserRepository,
	jwtService *auth.JWTService,
	cfg *config.Config,
	log *zerolog.Logger,
) *resolver.Resolver {
	return &resolver.Resolver{
		DB:              pool,
		ValidatorRepo:   repository.NewValidatorRepository(pool),
		SnapshotRepo:    repository.NewSnapshotRepository(pool),
		AlertRepo:       repository.NewAlertRepository(pool),
		PerformanceRepo: repository.NewPerformanceRepository(pool),
		UserRepo:        userRepo,
		Cache:           nil, // Cache initialization requires Redis config
		JWTService:      jwtService,
		Config:          cfg,
		Logger:          log,
	}
}
