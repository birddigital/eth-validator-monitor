package validators

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
)

var (
	validatorListCacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "validator_list_cache_hits_total",
		Help: "Total number of validator list cache hits",
	})

	validatorListCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "validator_list_cache_misses_total",
		Help: "Total number of validator list cache misses",
	})

	validatorListQueryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "validator_list_query_duration_seconds",
		Help:    "Duration of validator list queries",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
	})
)

type ListService struct {
	repo  *repository.ValidatorListRepository
	cache *cache.ValidatorListCache
}

func NewListService(repo *repository.ValidatorListRepository, cache *cache.ValidatorListCache) *ListService {
	return &ListService{
		repo:  repo,
		cache: cache,
	}
}

func (s *ListService) List(ctx context.Context, filter repository.ValidatorListFilter) (*repository.ValidatorListResult, error) {
	start := time.Now()
	defer func() {
		validatorListQueryDuration.Observe(time.Since(start).Seconds())
	}()

	// Try cache first
	if result, err := s.cache.Get(ctx, filter); err != nil {
		log.Printf("cache get error: %v", err)
		// Continue to database on cache error
	} else if result != nil {
		validatorListCacheHits.Inc()
		return result, nil
	}

	validatorListCacheMisses.Inc()

	// Query database
	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("repository list: %w", err)
	}

	// Cache the result (fire and forget)
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := s.cache.Set(cacheCtx, filter, result); err != nil {
			log.Printf("cache set error: %v", err)
		}
	}()

	return result, nil
}

func (s *ListService) InvalidateCache(ctx context.Context) error {
	return s.cache.InvalidateAll(ctx)
}
