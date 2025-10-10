package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// InvalidationEvent represents a cache invalidation event
type InvalidationEvent struct {
	Type      InvalidationType `json:"type"`
	EntityID  string           `json:"entity_id"`
	Timestamp time.Time        `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// InvalidationType defines the type of invalidation
type InvalidationType string

const (
	InvalidationTypeValidator        InvalidationType = "validator"
	InvalidationTypeSnapshot         InvalidationType = "snapshot"
	InvalidationTypePerformance      InvalidationType = "performance"
	InvalidationTypeNetworkStats     InvalidationType = "network_stats"
	InvalidationTypeAlert            InvalidationType = "alert"
	InvalidationTypeBulk             InvalidationType = "bulk"
)

// InvalidationManager handles cache invalidation
type InvalidationManager struct {
	cache           *RedisCache
	pubsub          *redis.PubSub
	listeners       map[InvalidationType][]InvalidationListener
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// InvalidationListener is called when an invalidation event occurs
type InvalidationListener func(event InvalidationEvent) error

// NewInvalidationManager creates a new invalidation manager
func NewInvalidationManager(cache *RedisCache) *InvalidationManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &InvalidationManager{
		cache:     cache,
		listeners: make(map[InvalidationType][]InvalidationListener),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins listening for invalidation events
func (im *InvalidationManager) Start() error {
	// Subscribe to invalidation channel
	im.pubsub = im.cache.client.Subscribe(im.ctx, "cache:invalidation")

	// Start event processor
	im.wg.Add(1)
	go im.processInvalidationEvents()

	log.Println("Cache invalidation manager started")
	return nil
}

// Stop stops the invalidation manager
func (im *InvalidationManager) Stop() error {
	im.cancel()

	if im.pubsub != nil {
		if err := im.pubsub.Close(); err != nil {
			log.Printf("Error closing pubsub: %v", err)
		}
	}

	im.wg.Wait()
	log.Println("Cache invalidation manager stopped")
	return nil
}

// processInvalidationEvents processes incoming invalidation events
func (im *InvalidationManager) processInvalidationEvents() {
	defer im.wg.Done()

	ch := im.pubsub.Channel()

	for {
		select {
		case <-im.ctx.Done():
			return
		case msg := <-ch:
			var event InvalidationEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("Failed to unmarshal invalidation event: %v", err)
				continue
			}

			// Process event
			im.handleInvalidationEvent(event)
		}
	}
}

// handleInvalidationEvent handles a single invalidation event
func (im *InvalidationManager) handleInvalidationEvent(event InvalidationEvent) {
	im.mu.RLock()
	listeners := im.listeners[event.Type]
	im.mu.RUnlock()

	for _, listener := range listeners {
		if err := listener(event); err != nil {
			log.Printf("Error in invalidation listener for %s: %v", event.Type, err)
		}
	}
}

// RegisterListener registers a listener for a specific invalidation type
func (im *InvalidationManager) RegisterListener(eventType InvalidationType, listener InvalidationListener) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.listeners[eventType] = append(im.listeners[eventType], listener)
}

// PublishInvalidation publishes an invalidation event
func (im *InvalidationManager) PublishInvalidation(ctx context.Context, event InvalidationEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal invalidation event: %w", err)
	}

	return im.cache.client.Publish(ctx, "cache:invalidation", data).Err()
}

// InvalidateValidator invalidates all cached data for a validator
func (im *InvalidationManager) InvalidateValidator(ctx context.Context, validatorIndex int) error {
	// Remove specific cache keys
	if err := im.cache.InvalidateAll(ctx, validatorIndex); err != nil {
		return err
	}

	// Publish invalidation event
	event := InvalidationEvent{
		Type:     InvalidationTypeValidator,
		EntityID: fmt.Sprintf("%d", validatorIndex),
		Metadata: map[string]interface{}{
			"validator_index": validatorIndex,
		},
	}

	return im.PublishInvalidation(ctx, event)
}

// InvalidateSnapshot invalidates a specific validator snapshot
func (im *InvalidationManager) InvalidateSnapshot(ctx context.Context, validatorIndex int) error {
	if err := im.cache.InvalidateValidatorSnapshot(ctx, validatorIndex); err != nil {
		return err
	}

	event := InvalidationEvent{
		Type:     InvalidationTypeSnapshot,
		EntityID: fmt.Sprintf("%d", validatorIndex),
		Metadata: map[string]interface{}{
			"validator_index": validatorIndex,
		},
	}

	return im.PublishInvalidation(ctx, event)
}

// InvalidatePerformance invalidates performance metrics
func (im *InvalidationManager) InvalidatePerformance(ctx context.Context, validatorIndex int, epoch int) error {
	if err := im.cache.InvalidatePerformance(ctx, validatorIndex, epoch); err != nil {
		return err
	}

	event := InvalidationEvent{
		Type:     InvalidationTypePerformance,
		EntityID: fmt.Sprintf("%d:%d", validatorIndex, epoch),
		Metadata: map[string]interface{}{
			"validator_index": validatorIndex,
			"epoch":          epoch,
		},
	}

	return im.PublishInvalidation(ctx, event)
}

// InvalidateNetworkStats invalidates network statistics
func (im *InvalidationManager) InvalidateNetworkStats(ctx context.Context) error {
	if err := im.cache.InvalidateNetworkStats(ctx); err != nil {
		return err
	}

	event := InvalidationEvent{
		Type:     InvalidationTypeNetworkStats,
		EntityID: "network",
	}

	return im.PublishInvalidation(ctx, event)
}

// BulkInvalidate invalidates multiple entries by pattern
func (im *InvalidationManager) BulkInvalidate(ctx context.Context, pattern string) error {
	iter := im.cache.client.Scan(ctx, 0, pattern, 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("error scanning keys: %w", err)
	}

	if len(keys) > 0 {
		if err := im.cache.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("error deleting keys: %w", err)
		}
		log.Printf("Bulk invalidated %d cache entries matching pattern: %s", len(keys), pattern)
	}

	event := InvalidationEvent{
		Type:     InvalidationTypeBulk,
		EntityID: pattern,
		Metadata: map[string]interface{}{
			"pattern":    pattern,
			"keys_count": len(keys),
		},
	}

	return im.PublishInvalidation(ctx, event)
}

// CacheVersion tracks versioned cache entries
type CacheVersion struct {
	cache  *RedisCache
	prefix string
}

// NewCacheVersion creates a new versioned cache manager
func NewCacheVersion(cache *RedisCache) *CacheVersion {
	return &CacheVersion{
		cache:  cache,
		prefix: "version",
	}
}

// versionKey generates a version tracking key
func (cv *CacheVersion) versionKey(key string) string {
	return fmt.Sprintf("%s:%s:%s", cv.cache.prefix, cv.prefix, key)
}

// GetVersion retrieves the current version for a key
func (cv *CacheVersion) GetVersion(ctx context.Context, key string) (int64, error) {
	vKey := cv.versionKey(key)

	val, err := cv.cache.client.Get(ctx, vKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return val, nil
}

// IncrementVersion increments the version for a key
func (cv *CacheVersion) IncrementVersion(ctx context.Context, key string) (int64, error) {
	vKey := cv.versionKey(key)
	return cv.cache.client.Incr(ctx, vKey).Result()
}

// SetWithVersion sets a cache entry with version tracking
func (cv *CacheVersion) SetWithVersion(ctx context.Context, key string, value interface{}, ttl time.Duration) (int64, error) {
	// Increment version
	version, err := cv.IncrementVersion(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("failed to increment version: %w", err)
	}

	// Set value
	data, err := json.Marshal(value)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal value: %w", err)
	}

	fullKey := fmt.Sprintf("%s:%s", cv.cache.prefix, key)
	if err := cv.cache.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return 0, fmt.Errorf("failed to set cache value: %w", err)
	}

	return version, nil
}

// GetWithVersion retrieves a cache entry and its version
func (cv *CacheVersion) GetWithVersion(ctx context.Context, key string) (interface{}, int64, error) {
	version, err := cv.GetVersion(ctx, key)
	if err != nil {
		return nil, 0, err
	}

	fullKey := fmt.Sprintf("%s:%s", cv.cache.prefix, key)
	data, err := cv.cache.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		return nil, version, err
	}

	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, version, err
	}

	return value, version, nil
}

// CleanupManager handles periodic cleanup of stale cache entries
type CleanupManager struct {
	cache       *RedisCache
	interval    time.Duration
	maxAge      time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(cache *RedisCache, interval, maxAge time.Duration) *CleanupManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &CleanupManager{
		cache:    cache,
		interval: interval,
		maxAge:   maxAge,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins periodic cleanup
func (cm *CleanupManager) Start() {
	cm.wg.Add(1)
	go cm.runCleanup()
	log.Printf("Cache cleanup manager started (interval: %v, max age: %v)", cm.interval, cm.maxAge)
}

// Stop stops the cleanup manager
func (cm *CleanupManager) Stop() {
	cm.cancel()
	cm.wg.Wait()
	log.Println("Cache cleanup manager stopped")
}

// runCleanup runs the periodic cleanup process
func (cm *CleanupManager) runCleanup() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.cleanup()
		}
	}
}

// cleanup performs the actual cleanup
func (cm *CleanupManager) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Scan for keys with the cache prefix
	pattern := fmt.Sprintf("%s:*", cm.cache.prefix)
	iter := cm.cache.client.Scan(ctx, 0, pattern, 100).Iterator()

	cleaned := 0
	for iter.Next(ctx) {
		key := iter.Val()

		// Check TTL
		ttl, err := cm.cache.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}

		// If no TTL or expired, remove
		if ttl < 0 {
			if err := cm.cache.client.Del(ctx, key).Err(); err == nil {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		log.Printf("Cache cleanup removed %d stale entries", cleaned)
	}
}

// AtomicUpdate performs an atomic cache update with consistency guarantee
func AtomicUpdate(ctx context.Context, cache *RedisCache, key string, updateFn func(current interface{}) (interface{}, error), ttl time.Duration) error {
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Watch the key for changes
		err := cache.client.Watch(ctx, func(tx *redis.Tx) error {
			// Get current value
			fullKey := fmt.Sprintf("%s:%s", cache.prefix, key)
			currentData, err := tx.Get(ctx, fullKey).Bytes()

			var current interface{}
			if err == nil {
				if err := json.Unmarshal(currentData, &current); err != nil {
					return err
				}
			}

			// Apply update function
			updated, err := updateFn(current)
			if err != nil {
				return err
			}

			// Marshal updated value
			updatedData, err := json.Marshal(updated)
			if err != nil {
				return err
			}

			// Execute update in transaction
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, fullKey, updatedData, ttl)
				return nil
			})

			return err
		}, key)

		if err == nil {
			return nil
		}

		if err == redis.TxFailedErr {
			// Transaction failed due to key modification, retry
			continue
		}

		return err
	}

	return fmt.Errorf("atomic update failed after %d attempts", maxRetries)
}
