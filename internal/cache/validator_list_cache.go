package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
)

type ValidatorListCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewValidatorListCache(client *redis.Client, ttl time.Duration) *ValidatorListCache {
	return &ValidatorListCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *ValidatorListCache) Get(ctx context.Context, filter repository.ValidatorListFilter) (*repository.ValidatorListResult, error) {
	key := c.buildCacheKey(filter)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var result repository.ValidatorListResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal cached data: %w", err)
	}

	return &result, nil
}

func (c *ValidatorListCache) Set(ctx context.Context, filter repository.ValidatorListFilter, result *repository.ValidatorListResult) error {
	key := c.buildCacheKey(filter)

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}

	return nil
}

func (c *ValidatorListCache) Invalidate(ctx context.Context, filter repository.ValidatorListFilter) error {
	key := c.buildCacheKey(filter)
	return c.client.Del(ctx, key).Err()
}

func (c *ValidatorListCache) InvalidateAll(ctx context.Context) error {
	// Use scan to find all validator list cache keys
	iter := c.client.Scan(ctx, 0, "validator_list:*", 0).Iterator()
	pipe := c.client.Pipeline()

	count := 0
	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
		count++

		// Execute pipeline in batches of 100
		if count%100 == 0 {
			if _, err := pipe.Exec(ctx); err != nil {
				return fmt.Errorf("pipeline exec: %w", err)
			}
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan iterator: %w", err)
	}

	// Execute remaining commands
	if count%100 != 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("pipeline exec final: %w", err)
		}
	}

	return nil
}

func (c *ValidatorListCache) buildCacheKey(filter repository.ValidatorListFilter) string {
	// Create deterministic hash of filter params
	data := fmt.Sprintf("%s:%s:%s:%s:%d:%d",
		filter.Search,
		filter.Status,
		filter.SortBy,
		filter.SortOrder,
		filter.Limit,
		filter.Offset,
	)

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("validator_list:%x", hash[:8]) // Use first 8 bytes for shorter key
}
