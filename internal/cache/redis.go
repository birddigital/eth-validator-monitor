package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/birddigital/eth-validator-monitor/pkg/types"
)

// TTLStrategy defines time-to-live durations for different data types
type TTLStrategy struct {
	ValidatorMetadata   time.Duration // Long-lived, rarely changes
	ValidatorSnapshot   time.Duration // Medium-lived, updates per epoch
	NetworkStats        time.Duration // Short-lived, updates frequently
	PerformanceMetrics  time.Duration // Medium-lived, updates per epoch
	AlertCache          time.Duration // Short-lived, time-sensitive
	BeaconHeadEvent     time.Duration // Very short-lived, real-time data
}

// DefaultTTLStrategy returns production-ready TTL values
func DefaultTTLStrategy() TTLStrategy {
	return TTLStrategy{
		ValidatorMetadata:   1 * time.Hour,     // Validator status changes infrequently
		ValidatorSnapshot:   15 * time.Minute,  // Balance updates ~every epoch (6.4 min)
		NetworkStats:        5 * time.Minute,   // Network-wide stats change often
		PerformanceMetrics:  30 * time.Minute,  // Performance scores per epoch
		AlertCache:          2 * time.Minute,   // Alerts are time-critical
		BeaconHeadEvent:     30 * time.Second,  // Real-time head updates
	}
}

// AggressiveTTLStrategy returns lower TTL values for high-load scenarios
func AggressiveTTLStrategy() TTLStrategy {
	return TTLStrategy{
		ValidatorMetadata:   30 * time.Minute,
		ValidatorSnapshot:   5 * time.Minute,
		NetworkStats:        1 * time.Minute,
		PerformanceMetrics:  10 * time.Minute,
		AlertCache:          1 * time.Minute,
		BeaconHeadEvent:     10 * time.Second,
	}
}

// ConservativeTTLStrategy returns higher TTL values for development/low-load
func ConservativeTTLStrategy() TTLStrategy {
	return TTLStrategy{
		ValidatorMetadata:   2 * time.Hour,
		ValidatorSnapshot:   30 * time.Minute,
		NetworkStats:        10 * time.Minute,
		PerformanceMetrics:  1 * time.Hour,
		AlertCache:          5 * time.Minute,
		BeaconHeadEvent:     1 * time.Minute,
	}
}

// RedisCache implements the Cache interface with TTL strategies
type RedisCache struct {
	client   *redis.Client
	strategy TTLStrategy
	prefix   string
}

// Config holds Redis cache configuration
type Config struct {
	Host        string
	Port        int
	Password    string
	DB          int
	MaxRetries  int
	PoolSize    int
	MinIdleConns int
	Strategy    TTLStrategy
	KeyPrefix   string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg Config) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:   client,
		strategy: cfg.Strategy,
		prefix:   cfg.KeyPrefix,
	}, nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Key generation helpers with namespacing
func (c *RedisCache) validatorKey(index int) string {
	return fmt.Sprintf("%s:validator:%d", c.prefix, index)
}

func (c *RedisCache) validatorSnapshotKey(index int) string {
	return fmt.Sprintf("%s:snapshot:%d", c.prefix, index)
}

func (c *RedisCache) networkStatsKey() string {
	return fmt.Sprintf("%s:network:stats", c.prefix)
}

func (c *RedisCache) performanceKey(index int, epoch int) string {
	return fmt.Sprintf("%s:performance:%d:%d", c.prefix, index, epoch)
}

func (c *RedisCache) alertsKey(index int) string {
	return fmt.Sprintf("%s:alerts:%d", c.prefix, index)
}

func (c *RedisCache) headEventKey() string {
	return fmt.Sprintf("%s:head:event", c.prefix)
}

// GetValidator retrieves a cached validator
func (c *RedisCache) GetValidator(ctx context.Context, index int) (*types.Validator, error) {
	key := c.validatorKey(index)
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("validator %d not in cache", index)
	}
	if err != nil {
		return nil, err
	}

	var validator types.Validator
	if err := json.Unmarshal(data, &validator); err != nil {
		return nil, err
	}

	return &validator, nil
}

// SetValidator caches a validator with appropriate TTL
func (c *RedisCache) SetValidator(ctx context.Context, validator *types.Validator) error {
	key := c.validatorKey(validator.Index)
	data, err := json.Marshal(validator)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.strategy.ValidatorMetadata).Err()
}

// ExtendValidatorTTL extends the TTL for a frequently accessed validator (hot data)
func (c *RedisCache) ExtendValidatorTTL(ctx context.Context, index int) error {
	key := c.validatorKey(index)
	return c.client.Expire(ctx, key, c.strategy.ValidatorMetadata).Err()
}

// GetValidatorSnapshot retrieves a cached validator snapshot
func (c *RedisCache) GetValidatorSnapshot(ctx context.Context, index int) (*types.ValidatorSnapshot, error) {
	key := c.validatorSnapshotKey(index)
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("snapshot for validator %d not in cache", index)
	}
	if err != nil {
		return nil, err
	}

	var snapshot types.ValidatorSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// SetValidatorSnapshot caches a validator snapshot
func (c *RedisCache) SetValidatorSnapshot(ctx context.Context, snapshot *types.ValidatorSnapshot) error {
	key := c.validatorSnapshotKey(snapshot.ValidatorIndex)
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.strategy.ValidatorSnapshot).Err()
}

// GetNetworkStats retrieves cached network statistics
func (c *RedisCache) GetNetworkStats(ctx context.Context) (*types.NetworkStats, error) {
	key := c.networkStatsKey()
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("network stats not in cache")
	}
	if err != nil {
		return nil, err
	}

	var stats types.NetworkStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// SetNetworkStats caches network statistics
func (c *RedisCache) SetNetworkStats(ctx context.Context, stats *types.NetworkStats) error {
	key := c.networkStatsKey()
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.strategy.NetworkStats).Err()
}

// GetPerformance retrieves cached performance metrics
func (c *RedisCache) GetPerformance(ctx context.Context, index int, epoch int) (*types.PerformanceMetrics, error) {
	key := c.performanceKey(index, epoch)
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("performance for validator %d epoch %d not in cache", index, epoch)
	}
	if err != nil {
		return nil, err
	}

	var perf types.PerformanceMetrics
	if err := json.Unmarshal(data, &perf); err != nil {
		return nil, err
	}

	return &perf, nil
}

// SetPerformance caches performance metrics
func (c *RedisCache) SetPerformance(ctx context.Context, perf *types.PerformanceMetrics) error {
	key := c.performanceKey(perf.ValidatorIndex, perf.Epoch)
	data, err := json.Marshal(perf)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.strategy.PerformanceMetrics).Err()
}

// GetAlerts retrieves cached alerts for a validator
func (c *RedisCache) GetAlerts(ctx context.Context, index int) ([]*types.Alert, error) {
	key := c.alertsKey(index)
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("alerts for validator %d not in cache", index)
	}
	if err != nil {
		return nil, err
	}

	var alerts []*types.Alert
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, err
	}

	return alerts, nil
}

// SetAlerts caches alerts for a validator
func (c *RedisCache) SetAlerts(ctx context.Context, index int, alerts []*types.Alert) error {
	key := c.alertsKey(index)
	data, err := json.Marshal(alerts)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.strategy.AlertCache).Err()
}

// InvalidateValidator removes a validator from cache
func (c *RedisCache) InvalidateValidator(ctx context.Context, index int) error {
	return c.client.Del(ctx, c.validatorKey(index)).Err()
}

// InvalidateValidatorSnapshot removes a validator snapshot from cache
func (c *RedisCache) InvalidateValidatorSnapshot(ctx context.Context, index int) error {
	return c.client.Del(ctx, c.validatorSnapshotKey(index)).Err()
}

// InvalidateNetworkStats removes network stats from cache
func (c *RedisCache) InvalidateNetworkStats(ctx context.Context) error {
	return c.client.Del(ctx, c.networkStatsKey()).Err()
}

// InvalidatePerformance removes performance metrics from cache
func (c *RedisCache) InvalidatePerformance(ctx context.Context, index int, epoch int) error {
	return c.client.Del(ctx, c.performanceKey(index, epoch)).Err()
}

// InvalidateAll removes all cached data for a validator
func (c *RedisCache) InvalidateAll(ctx context.Context, index int) error {
	keys := []string{
		c.validatorKey(index),
		c.validatorSnapshotKey(index),
		c.alertsKey(index),
	}

	return c.client.Del(ctx, keys...).Err()
}

// GetHeadEvent retrieves the latest cached head event
func (c *RedisCache) GetHeadEvent(ctx context.Context) (*types.HeadEvent, error) {
	key := c.headEventKey()
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("head event not in cache")
	}
	if err != nil {
		return nil, err
	}

	var event types.HeadEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}

	return &event, nil
}

// SetHeadEvent caches the latest head event
func (c *RedisCache) SetHeadEvent(ctx context.Context, event *types.HeadEvent) error {
	key := c.headEventKey()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.strategy.BeaconHeadEvent).Err()
}

// GetCacheStats retrieves cache statistics
func (c *RedisCache) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	dbSize, err := c.client.DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"info":    info,
		"db_size": dbSize,
		"strategy": map[string]string{
			"validator_metadata":  c.strategy.ValidatorMetadata.String(),
			"validator_snapshot":  c.strategy.ValidatorSnapshot.String(),
			"network_stats":       c.strategy.NetworkStats.String(),
			"performance_metrics": c.strategy.PerformanceMetrics.String(),
			"alert_cache":         c.strategy.AlertCache.String(),
			"head_event":          c.strategy.BeaconHeadEvent.String(),
		},
	}, nil
}

// UpdateTTLStrategy dynamically updates the TTL strategy
func (c *RedisCache) UpdateTTLStrategy(strategy TTLStrategy) {
	c.strategy = strategy
}

// Flush removes all keys with the configured prefix
func (c *RedisCache) Flush(ctx context.Context) error {
	iter := c.client.Scan(ctx, 0, c.prefix+":*", 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// GetTTL returns the remaining TTL for a specific validator
func (c *RedisCache) GetTTL(ctx context.Context, index int) (time.Duration, error) {
	key := c.validatorKey(index)
	return c.client.TTL(ctx, key).Result()
}

// SetWithCustomTTL allows setting a cache entry with a custom TTL
func (c *RedisCache) SetWithCustomTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	fullKey := fmt.Sprintf("%s:%s", c.prefix, key)
	return c.client.Set(ctx, fullKey, data, ttl).Err()
}

// GetWithExtend retrieves a value and extends its TTL (for hot data)
func (c *RedisCache) GetWithExtend(ctx context.Context, index int) (*types.Validator, error) {
	validator, err := c.GetValidator(ctx, index)
	if err != nil {
		return nil, err
	}

	// Extend TTL for frequently accessed data
	_ = c.ExtendValidatorTTL(ctx, index)

	return validator, nil
}

// LatestSnapshotKey generates a cache key for the latest validator snapshot
func (c *RedisCache) LatestSnapshotKey(index int64) string {
	return fmt.Sprintf("snapshot:%d:latest", index)
}

// ValidatorMetadataKey generates a cache key for validator metadata
func (c *RedisCache) ValidatorMetadataKey(index int64) string {
	return fmt.Sprintf("validator:%d", index)
}

// PerformanceKey generates a cache key for performance metrics
func (c *RedisCache) PerformanceKey(index int64, epochFrom int64, epochTo int64) string {
	return fmt.Sprintf("performance:%d:%d:%d", index, epochFrom, epochTo)
}

// NetworkStatsKey generates a cache key for network statistics
func (c *RedisCache) NetworkStatsKey() string {
	return "network:stats"
}

// SetValidatorMetadata caches validator metadata
func (c *RedisCache) SetValidatorMetadata(ctx context.Context, index int64, metadata map[string]interface{}) error {
	key := c.ValidatorMetadataKey(index)
	return c.Set(ctx, key, metadata, GetValidatorMetadataTTL())
}

// GetValidatorMetadata retrieves validator metadata from cache
func (c *RedisCache) GetValidatorMetadata(ctx context.Context, index int64) (map[string]interface{}, error) {
	key := c.ValidatorMetadataKey(index)
	var metadata map[string]interface{}
	err := c.Get(ctx, key, &metadata)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

// BatchSet sets multiple cache entries with the same TTL
func (c *RedisCache) BatchSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipe := c.client.Pipeline()

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
		}
		pipe.Set(ctx, key, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// GetLatestSnapshotTTL returns the TTL for latest snapshot cache entries
func GetLatestSnapshotTTL() time.Duration {
	return DefaultTTLStrategy().ValidatorSnapshot
}

// Get retrieves a generic cached value
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return fmt.Errorf("key %s not in cache", key)
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Set caches a generic value with a specific TTL
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

// GetValidatorMetadataTTL returns the TTL for validator metadata
func GetValidatorMetadataTTL() time.Duration {
	return DefaultTTLStrategy().ValidatorMetadata
}

// GetValidatorSnapshotTTL returns the TTL for validator snapshots
func GetValidatorSnapshotTTL() time.Duration {
	return DefaultTTLStrategy().ValidatorSnapshot
}

// GetPerformanceMetricsTTL returns the TTL for performance metrics
func GetPerformanceMetricsTTL() time.Duration {
	return DefaultTTLStrategy().PerformanceMetrics
}

// GetAlertCacheTTL returns the TTL for alerts
func GetAlertCacheTTL() time.Duration {
	return DefaultTTLStrategy().AlertCache
}
