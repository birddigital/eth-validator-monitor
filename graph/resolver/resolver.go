package resolver

import (
	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/config"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/storage"
	"github.com/birddigital/eth-validator-monitor/graph/dataloader"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	// Database pool
	DB *pgxpool.Pool

	// Repositories
	ValidatorRepo   *repository.ValidatorRepository
	SnapshotRepo    *repository.SnapshotRepository
	AlertRepo       *repository.AlertRepository
	PerformanceRepo *repository.PerformanceRepository
	UserRepo        *storage.UserRepository

	// Cache
	Cache *cache.RedisCache

	// Authentication
	JWTService *auth.JWTService

	// Config
	Config *config.Config

	// Logger
	Logger *zerolog.Logger

	// DataLoaders (will be populated per-request)
	DataLoaders *dataloader.Loaders
}
