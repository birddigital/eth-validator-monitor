package resolver

import (
	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/graph/dataloader"
	"github.com/jackc/pgx/v5/pgxpool"
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

	// Cache
	Cache *cache.RedisCache

	// DataLoaders (will be populated per-request)
	DataLoaders *dataloader.Loaders
}
