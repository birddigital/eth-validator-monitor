package main

import (
	"context"

	"github.com/birddigital/eth-validator-monitor/internal/services/health"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxPoolAdapter adapts *pgxpool.Pool to implement health.DBPinger interface
type pgxPoolAdapter struct {
	pool *pgxpool.Pool
}

// newPgxPoolAdapter creates a new adapter for pgxpool.Pool
func newPgxPoolAdapter(pool *pgxpool.Pool) health.DBPinger {
	return &pgxPoolAdapter{pool: pool}
}

// Ping implements health.DBPinger.Ping
func (a *pgxPoolAdapter) Ping(ctx context.Context) error {
	return a.pool.Ping(ctx)
}

// Stat implements health.DBPinger.Stat
func (a *pgxPoolAdapter) Stat() health.PoolStats {
	return &pgxPoolStatsAdapter{stat: a.pool.Stat()}
}

// pgxPoolStatsAdapter adapts pgxpool.Stat to implement health.PoolStats interface
type pgxPoolStatsAdapter struct {
	stat *pgxpool.Stat
}

// TotalConns implements health.PoolStats.TotalConns
func (s *pgxPoolStatsAdapter) TotalConns() int32 {
	return s.stat.TotalConns()
}

// IdleConns implements health.PoolStats.IdleConns
func (s *pgxPoolStatsAdapter) IdleConns() int32 {
	return s.stat.IdleConns()
}
