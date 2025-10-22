package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/logger"
	"github.com/birddigital/eth-validator-monitor/internal/web/sse"
	"github.com/birddigital/eth-validator-monitor/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/birddigital/eth-validator-monitor/internal/cache"
)

// ValidatorCollector manages the collection of validator data
type ValidatorCollector struct {
	beaconClient    types.BeaconClient
	pool            *pgxpool.Pool
	cache           *cache.RedisCache
	workerPool      *WorkerPool
	broadcaster     *sse.Broadcaster

	// Repositories
	validatorRepo   *repository.ValidatorRepository
	snapshotRepo    *repository.SnapshotRepository

	// Configuration
	collectionInterval time.Duration
	batchSize         int
	validators        []int64 // List of validator indices to monitor

	// Control
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup

	// Metrics
	lastCollectionTime time.Time
	collectionsCount   uint64
	errorsCount        uint64
	mu                 sync.RWMutex
}

// CollectorConfig contains configuration for the validator collector
type CollectorConfig struct {
	CollectionInterval time.Duration
	BatchSize          int
	WorkerPoolConfig   *WorkerPoolConfig
}

// DefaultCollectorConfig returns default collector configuration
func DefaultCollectorConfig() *CollectorConfig {
	return &CollectorConfig{
		CollectionInterval: time.Second * 12, // Ethereum epoch time
		BatchSize:          100,
		WorkerPoolConfig:   DefaultWorkerPoolConfig(),
	}
}

// NewValidatorCollector creates a new validator collector
func NewValidatorCollector(
	ctx context.Context,
	beaconClient types.BeaconClient,
	pool *pgxpool.Pool,
	redisCache *cache.RedisCache,
	broadcaster *sse.Broadcaster,
	config *CollectorConfig,
) *ValidatorCollector {
	collectorCtx, cancel := context.WithCancel(ctx)

	return &ValidatorCollector{
		beaconClient:       beaconClient,
		pool:              pool,
		cache:             redisCache,
		broadcaster:       broadcaster,
		workerPool:        NewWorkerPool(collectorCtx, config.WorkerPoolConfig),
		validatorRepo:     repository.NewValidatorRepository(pool),
		snapshotRepo:      repository.NewSnapshotRepository(pool),
		collectionInterval: config.CollectionInterval,
		batchSize:         config.BatchSize,
		ctx:               collectorCtx,
		cancel:            cancel,
	}
}

// Start begins the collection process
func (c *ValidatorCollector) Start() error {
	// Load validators to monitor
	if err := c.loadValidators(); err != nil {
		return fmt.Errorf("failed to load validators: %w", err)
	}

	// Start worker pool
	c.workerPool.Start()

	// Start result processor
	c.wg.Add(1)
	go c.processResults()

	// Start collection ticker
	c.wg.Add(1)
	go c.runCollectionLoop()

	// Start head event subscriber
	c.wg.Add(1)
	go c.subscribeToHeadEvents()

	logger.FromContext(c.ctx).Info().
		Int("validator_count", len(c.validators)).
		Msg("Validator collector started monitoring validators")
	return nil
}

// loadValidators loads the list of validators to monitor
func (c *ValidatorCollector) loadValidators() error {
	filter := &models.ValidatorFilter{
		Monitored: &[]bool{true}[0],
	}

	validators, err := c.validatorRepo.ListValidators(c.ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to load validators: %w", err)
	}

	c.validators = make([]int64, len(validators))
	for i, v := range validators {
		c.validators[i] = v.ValidatorIndex
	}

	return nil
}

// runCollectionLoop runs the main collection loop
func (c *ValidatorCollector) runCollectionLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.collectionInterval)
	defer ticker.Stop()

	// Perform initial collection
	c.collectAllValidators()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectAllValidators()
		}
	}
}

// collectAllValidators initiates collection for all monitored validators
func (c *ValidatorCollector) collectAllValidators() {
	c.mu.Lock()
	c.lastCollectionTime = time.Now()
	c.collectionsCount++
	c.mu.Unlock()

	// Submit tasks in batches to avoid overwhelming the queue
	for i := 0; i < len(c.validators); i += c.batchSize {
		end := i + c.batchSize
		if end > len(c.validators) {
			end = len(c.validators)
		}

		batch := c.validators[i:end]
		for _, validatorIndex := range batch {
			task := Task{
				ID:             fmt.Sprintf("snapshot-%d-%d", validatorIndex, time.Now().Unix()),
				ValidatorIndex: validatorIndex,
				Type:           TaskTypeSnapshot,
				Deadline:       time.Now().Add(c.collectionInterval),
			}

			if err := c.workerPool.Submit(task); err != nil {
				logger.FromContext(c.ctx).Error().
					Err(err).
					Int64("validator_index", validatorIndex).
					Msg("Failed to submit collection task")
				c.mu.Lock()
				c.errorsCount++
				c.mu.Unlock()
			}
		}

		// Small delay between batches to prevent queue overflow
		time.Sleep(time.Millisecond * 10)
	}
}

// processResults processes collection results from the worker pool
func (c *ValidatorCollector) processResults() {
	defer c.wg.Done()

	resultChan := c.workerPool.Results()
	batchResults := make([]*models.ValidatorSnapshot, 0, c.batchSize)
	batchTimer := time.NewTicker(time.Second * 2)
	defer batchTimer.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Flush remaining batch
			if len(batchResults) > 0 {
				c.storeBatch(batchResults)
			}
			return

		case result, ok := <-resultChan:
			if !ok {
				return
			}

			if result.Error != nil {
				logger.FromContext(c.ctx).Error().
					Err(result.Error).
					Int64("validator_index", result.ValidatorIndex).
					Msg("Collection error for validator")
				c.mu.Lock()
				c.errorsCount++
				c.mu.Unlock()
				continue
			}

			// Convert result to snapshot
			snapshot, err := c.resultToSnapshot(result)
			if err != nil {
				logger.FromContext(c.ctx).Error().
					Err(err).
					Msg("Failed to convert result to snapshot")
				continue
			}

			batchResults = append(batchResults, snapshot)

			// Store batch when it reaches the size limit
			if len(batchResults) >= c.batchSize {
				c.storeBatch(batchResults)
				batchResults = make([]*models.ValidatorSnapshot, 0, c.batchSize)
			}

		case <-batchTimer.C:
			// Periodic flush of partial batches
			if len(batchResults) > 0 {
				c.storeBatch(batchResults)
				batchResults = make([]*models.ValidatorSnapshot, 0, c.batchSize)
			}
		}
	}
}

// storeBatch stores a batch of snapshots to the database and cache
func (c *ValidatorCollector) storeBatch(snapshots []*models.ValidatorSnapshot) {
	if len(snapshots) == 0 {
		return
	}

	// Store in database
	if err := c.snapshotRepo.BatchInsertSnapshots(c.ctx, snapshots); err != nil {
		logger.FromContext(c.ctx).Error().
			Err(err).
			Int("batch_size", len(snapshots)).
			Msg("Failed to store snapshot batch")
		return
	}

	// Update cache for latest snapshots
	cacheItems := make(map[string]interface{})
	for _, snapshot := range snapshots {
		key := c.cache.LatestSnapshotKey(snapshot.ValidatorIndex)
		cacheItems[key] = snapshot
	}

	if err := c.cache.BatchSet(c.ctx, cacheItems, cache.GetLatestSnapshotTTL()); err != nil {
		logger.FromContext(c.ctx).Warn().
			Err(err).
			Int("cache_item_count", len(cacheItems)).
			Msg("Failed to update cache")
	}

	// Broadcast SSE events for real-time updates
	if c.broadcaster != nil {
		for _, snapshot := range snapshots {
			c.broadcastMetricsUpdate(snapshot)
		}
	}

	logger.FromContext(c.ctx).Debug().
		Int("snapshot_count", len(snapshots)).
		Msg("Stored batch of snapshots")
}

// resultToSnapshot converts a collection result to a validator snapshot
func (c *ValidatorCollector) resultToSnapshot(result Result) (*models.ValidatorSnapshot, error) {
	data, ok := result.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid result data type")
	}

	// Extract data from result (this is simplified - real implementation would parse beacon data)
	snapshot := &models.ValidatorSnapshot{
		Time:           result.CollectedAt,
		ValidatorIndex: result.ValidatorIndex,
		Balance:        extractInt64(data, "balance"),
		EffectiveBalance: extractInt64(data, "effective_balance"),
		IsOnline:       true,
	}

	// Calculate attestation effectiveness
	if headVote, ok := data["head_vote"].(bool); ok {
		snapshot.AttestationHeadVote = &headVote
	}
	if sourceVote, ok := data["source_vote"].(bool); ok {
		snapshot.AttestationSourceVote = &sourceVote
	}
	if targetVote, ok := data["target_vote"].(bool); ok {
		snapshot.AttestationTargetVote = &targetVote
	}
	if inclusionDelay, ok := data["inclusion_delay"].(int32); ok {
		snapshot.AttestationInclusionDelay = &inclusionDelay

		// Calculate effectiveness score
		effectiveness := repository.CalculateEffectivenessScore(
			snapshot.AttestationHeadVote != nil && *snapshot.AttestationHeadVote,
			snapshot.AttestationSourceVote != nil && *snapshot.AttestationSourceVote,
			snapshot.AttestationTargetVote != nil && *snapshot.AttestationTargetVote,
			inclusionDelay,
		)
		snapshot.AttestationEffectiveness = &effectiveness
	}

	return snapshot, nil
}

// subscribeToHeadEvents subscribes to beacon chain head events
func (c *ValidatorCollector) subscribeToHeadEvents() {
	defer c.wg.Done()

	headChan, err := c.beaconClient.SubscribeToHead(c.ctx)
	if err != nil {
		logger.FromContext(c.ctx).Error().
			Err(err).
			Msg("Failed to subscribe to head events")
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		case head, ok := <-headChan:
			if !ok {
				logger.FromContext(c.ctx).Warn().
					Msg("Head event channel closed, attempting to reconnect")
				time.Sleep(time.Second * 5)

				// Try to reconnect
				headChan, err = c.beaconClient.SubscribeToHead(c.ctx)
				if err != nil {
					logger.FromContext(c.ctx).Error().
						Err(err).
						Msg("Failed to reconnect to head events")
					continue
				}
			} else {
				// Process head event
				epoch := head.Slot / 32 // Calculate epoch from slot
				logger.FromContext(c.ctx).Debug().
					Int64("slot", int64(head.Slot)).
					Int64("epoch", int64(epoch)).
					Msg("New head event received")
				// Could trigger immediate collection for critical validators here
			}
		}
	}
}

// Stop gracefully stops the collector
func (c *ValidatorCollector) Stop() error {
	logger.FromContext(c.ctx).Info().Msg("Stopping validator collector")

	// Cancel context to stop all goroutines
	c.cancel()

	// Shutdown worker pool
	if err := c.workerPool.Shutdown(time.Second * 30); err != nil {
		logger.FromContext(c.ctx).Error().
			Err(err).
			Msg("Error shutting down worker pool")
	}

	// Wait for all goroutines to finish
	c.wg.Wait()

	logger.FromContext(c.ctx).Info().Msg("Validator collector stopped successfully")
	return nil
}

// Stats returns collector statistics
func (c *ValidatorCollector) Stats() CollectorStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	poolStats := c.workerPool.Stats()

	return CollectorStats{
		ValidatorsMonitored: len(c.validators),
		LastCollectionTime:  c.lastCollectionTime,
		CollectionsCount:    c.collectionsCount,
		ErrorsCount:         c.errorsCount,
		PoolStats:          poolStats,
	}
}

// CollectorStats contains collector statistics
type CollectorStats struct {
	ValidatorsMonitored int
	LastCollectionTime  time.Time
	CollectionsCount    uint64
	ErrorsCount         uint64
	PoolStats           PoolStats
}

// AddValidator adds a validator to the monitoring list
func (c *ValidatorCollector) AddValidator(validatorIndex int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already monitoring
	for _, v := range c.validators {
		if v == validatorIndex {
			return
		}
	}

	c.validators = append(c.validators, validatorIndex)
	logger.FromContext(c.ctx).Info().
		Int64("validator_index", validatorIndex).
		Int("total_validators", len(c.validators)).
		Msg("Added validator to monitoring list")
}

// RemoveValidator removes a validator from the monitoring list
func (c *ValidatorCollector) RemoveValidator(validatorIndex int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, v := range c.validators {
		if v == validatorIndex {
			c.validators = append(c.validators[:i], c.validators[i+1:]...)
			logger.FromContext(c.ctx).Info().
				Int64("validator_index", validatorIndex).
				Int("total_validators", len(c.validators)).
				Msg("Removed validator from monitoring list")
			return
		}
	}
}

// Helper function to extract int64 from interface{}
func extractInt64(data map[string]interface{}, key string) int64 {
	if val, ok := data[key].(float64); ok {
		return int64(val)
	}
	if val, ok := data[key].(int64); ok {
		return val
	}
	return 0
}

// broadcastMetricsUpdate broadcasts a metrics update event via SSE
func (c *ValidatorCollector) broadcastMetricsUpdate(snapshot *models.ValidatorSnapshot) {
	if c.broadcaster == nil {
		return
	}

	// Convert snapshot to SSE metrics data
	var effectiveness float64
	if snapshot.AttestationEffectiveness != nil {
		effectiveness = *snapshot.AttestationEffectiveness
	}

	status := "active"
	if !snapshot.IsOnline {
		status = "offline"
	}

	data := sse.MetricsUpdateData{
		ValidatorIndex: uint64(snapshot.ValidatorIndex),
		Balance:        uint64(snapshot.Balance),
		Effectiveness:  effectiveness,
		Status:         status,
		LastUpdated:    snapshot.Time.Unix(),
	}

	// Broadcast the event
	c.broadcaster.Broadcast(sse.Event{
		Type: sse.EventTypeMetricsUpdate,
		Data: data,
		ID:   fmt.Sprintf("metrics-%d-%d", snapshot.ValidatorIndex, snapshot.Time.Unix()),
	})
}