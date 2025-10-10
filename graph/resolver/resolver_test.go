package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/birddigital/eth-validator-monitor/graph/dataloader"
	"github.com/birddigital/eth-validator-monitor/graph/model"
	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryResolver_Validator(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	// Setup repositories
	validatorRepo := repository.NewValidatorRepository(pool)
	snapshotRepo := repository.NewSnapshotRepository(pool)
	alertRepo := repository.NewAlertRepository(pool)

	// Setup cache
	cacheConfig := cache.Config{
		Host:      "localhost",
		Port:      6379,
		Password:  "",
		DB:        1,
		Strategy:  cache.DefaultTTLStrategy(),
		KeyPrefix: "test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	// Setup dataloaders
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	// Create resolver
	resolver := &Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
	}

	ctx := context.Background()

	// Create test validator
	validator := testutil.ValidatorFixture(123)
	err = validatorRepo.CreateValidator(ctx, validator)
	require.NoError(t, err)

	// Test Query.validator resolver
	queryResolver := resolver.Query()
	result, err := queryResolver.Validator(ctx, 123)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(123), result.ValidatorIndex)
	assert.Equal(t, validator.Pubkey, result.Pubkey)
}

func TestQueryResolver_Validators(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	// Setup repositories
	validatorRepo := repository.NewValidatorRepository(pool)
	snapshotRepo := repository.NewSnapshotRepository(pool)
	alertRepo := repository.NewAlertRepository(pool)

	// Setup cache
	cacheConfig := cache.Config{
		Host:      "localhost",
		Port:      6379,
		Password:  "",
		DB:        1,
		Strategy:  cache.DefaultTTLStrategy(),
		KeyPrefix: "test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	// Setup dataloaders
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	// Create resolver
	resolver := &Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
	}

	ctx := context.Background()

	// Create multiple validators
	validators := testutil.MultipleValidatorFixtures(25)
	err = validatorRepo.BatchCreateValidators(ctx, validators)
	require.NoError(t, err)

	// Test Query.validators resolver
	queryResolver := resolver.Query()

	// Test basic pagination
	limit := 10
	pagination := &model.PaginationInput{
		Limit: &limit,
	}

	result, err := queryResolver.Validators(ctx, nil, nil, pagination)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Edges, 10)
	assert.True(t, result.PageInfo.HasNextPage)
	assert.Equal(t, 25, result.TotalCount)

	// Test with cursor-based pagination
	if len(result.Edges) > 0 {
		cursor := result.PageInfo.EndCursor
		nextPagination := &model.PaginationInput{
			Limit:  &limit,
			Cursor: cursor,
		}

		nextResult, err := queryResolver.Validators(ctx, nil, nil, nextPagination)
		require.NoError(t, err)
		assert.Len(t, nextResult.Edges, 10)
	}

	// Test with filter
	monitoredTrue := true
	filter := &model.ValidatorFilterInput{
		Monitored: &monitoredTrue,
	}

	filteredResult, err := queryResolver.Validators(ctx, filter, nil, pagination)
	require.NoError(t, err)
	assert.NotNil(t, filteredResult)
}

func TestValidatorResolver_LatestSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	// Setup repositories
	validatorRepo := repository.NewValidatorRepository(pool)
	snapshotRepo := repository.NewSnapshotRepository(pool)
	alertRepo := repository.NewAlertRepository(pool)

	// Setup cache
	cacheConfig := cache.Config{
		Host:      "localhost",
		Port:      6379,
		Password:  "",
		DB:        1,
		Strategy:  cache.DefaultTTLStrategy(),
		KeyPrefix: "test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	// Setup dataloaders
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	// Create resolver
	resolver := &Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
	}

	ctx := context.Background()

	// Create validator and snapshots
	validator := testutil.ValidatorFixture(456)
	err = validatorRepo.CreateValidator(ctx, validator)
	require.NoError(t, err)

	baseTime := time.Now().Add(-24 * time.Hour)
	for i := 0; i < 10; i++ {
		snapshot := testutil.ValidatorSnapshotFixture(456, baseTime.Add(time.Duration(i)*time.Hour))
		err := snapshotRepo.InsertSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Test Validator.latestSnapshot resolver
	validatorResolver := resolver.Validator()
	latestSnapshot, err := validatorResolver.LatestSnapshot(ctx, validator)
	require.NoError(t, err)
	require.NotNil(t, latestSnapshot)
	assert.Equal(t, int64(456), latestSnapshot.ValidatorIndex)
}

func TestValidatorResolver_Snapshots(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	// Setup repositories
	validatorRepo := repository.NewValidatorRepository(pool)
	snapshotRepo := repository.NewSnapshotRepository(pool)
	alertRepo := repository.NewAlertRepository(pool)

	// Setup cache
	cacheConfig := cache.Config{
		Host:      "localhost",
		Port:      6379,
		Password:  "",
		DB:        1,
		Strategy:  cache.DefaultTTLStrategy(),
		KeyPrefix: "test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	// Setup dataloaders
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	// Create resolver
	resolver := &Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
	}

	ctx := context.Background()

	// Create validator and snapshots
	validator := testutil.ValidatorFixture(789)
	err = validatorRepo.CreateValidator(ctx, validator)
	require.NoError(t, err)

	snapshots := testutil.MultipleSnapshotFixtures(789, 50)
	err = snapshotRepo.BatchInsertSnapshots(ctx, snapshots)
	require.NoError(t, err)

	// Test Validator.snapshots resolver
	validatorResolver := resolver.Validator()

	limit := 20
	pagination := &model.PaginationInput{
		Limit: &limit,
	}

	result, err := validatorResolver.Snapshots(ctx, validator, nil, nil, pagination)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Edges, 20)
	assert.True(t, result.PageInfo.HasNextPage)
}

func TestQueryResolver_NetworkStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	// Setup repositories
	validatorRepo := repository.NewValidatorRepository(pool)
	snapshotRepo := repository.NewSnapshotRepository(pool)

	// Setup cache
	cacheConfig := cache.Config{
		Host:      "localhost",
		Port:      6379,
		Password:  "",
		DB:        1,
		Strategy:  cache.DefaultTTLStrategy(),
		KeyPrefix: "test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	// Create resolver
	resolver := &Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		Cache:         redisCache,
	}

	ctx := context.Background()

	// Test Query.networkStats resolver
	queryResolver := resolver.Query()
	result, err := queryResolver.NetworkStats(ctx)

	// Network stats will return default/empty values since we don't have real network data
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestDataLoaderCaching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	// Setup repositories
	validatorRepo := repository.NewValidatorRepository(pool)
	snapshotRepo := repository.NewSnapshotRepository(pool)
	alertRepo := repository.NewAlertRepository(pool)

	// Setup cache
	cacheConfig := cache.Config{
		Host:      "localhost",
		Port:      6379,
		Password:  "",
		DB:        1,
		Strategy:  cache.DefaultTTLStrategy(),
		KeyPrefix: "test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	// Flush cache to start fresh
	ctx := context.Background()
	err = redisCache.Flush(ctx)
	require.NoError(t, err)

	// Setup dataloaders
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	// Create test validators
	validators := testutil.MultipleValidatorFixtures(10)
	err = validatorRepo.BatchCreateValidators(ctx, validators)
	require.NoError(t, err)

	// Load validator through dataloader (first load - cache miss)
	thunk1 := loaders.ValidatorByIndex.Load(ctx, 100)
	result1, err1 := thunk1()
	require.NoError(t, err1)
	require.NotNil(t, result1)

	// Load same validator again (should hit cache)
	thunk2 := loaders.ValidatorByIndex.Load(ctx, 100)
	result2, err2 := thunk2()
	require.NoError(t, err2)
	require.NotNil(t, result2)

	// Results should be equivalent
	assert.Equal(t, result1.ValidatorIndex, result2.ValidatorIndex)
	assert.Equal(t, result1.Pubkey, result2.Pubkey)
}

func TestPaginationEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	// Setup repositories
	validatorRepo := repository.NewValidatorRepository(pool)
	snapshotRepo := repository.NewSnapshotRepository(pool)
	alertRepo := repository.NewAlertRepository(pool)

	// Setup cache
	cacheConfig := cache.Config{
		Host:      "localhost",
		Port:      6379,
		Password:  "",
		DB:        1,
		Strategy:  cache.DefaultTTLStrategy(),
		KeyPrefix: "test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	// Setup dataloaders
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	// Create resolver
	resolver := &Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
	}

	ctx := context.Background()

	// Create exactly 10 validators
	validators := testutil.MultipleValidatorFixtures(10)
	err = validatorRepo.BatchCreateValidators(ctx, validators)
	require.NoError(t, err)

	queryResolver := resolver.Query()

	// Test exact page size
	limit := 10
	pagination := &model.PaginationInput{
		Limit: &limit,
	}

	result, err := queryResolver.Validators(ctx, nil, nil, pagination)
	require.NoError(t, err)
	assert.Len(t, result.Edges, 10)
	assert.False(t, result.PageInfo.HasNextPage) // No more pages

	// Test larger limit than available
	largeLimit := 100
	largePagination := &model.PaginationInput{
		Limit: &largeLimit,
	}

	largeResult, err := queryResolver.Validators(ctx, nil, nil, largePagination)
	require.NoError(t, err)
	assert.Len(t, largeResult.Edges, 10)
	assert.False(t, largeResult.PageInfo.HasNextPage)

	// Test zero/nil pagination (should use defaults)
	defaultResult, err := queryResolver.Validators(ctx, nil, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, defaultResult)
}

func TestFilterCombinations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(context.Background(), pool)

	// Setup repositories
	validatorRepo := repository.NewValidatorRepository(pool)
	snapshotRepo := repository.NewSnapshotRepository(pool)
	alertRepo := repository.NewAlertRepository(pool)

	// Setup cache
	cacheConfig := cache.Config{
		Host:      "localhost",
		Port:      6379,
		Password:  "",
		DB:        1,
		Strategy:  cache.DefaultTTLStrategy(),
		KeyPrefix: "test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	// Setup dataloaders
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	// Create resolver
	resolver := &Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
	}

	ctx := context.Background()

	// Create validators with different properties
	for i := 0; i < 20; i++ {
		validator := testutil.ValidatorFixture(int64(1000 + i))
		if i%2 == 0 {
			validator.Monitored = false
		}
		err := validatorRepo.CreateValidator(ctx, validator)
		require.NoError(t, err)
	}

	queryResolver := resolver.Query()

	// Test monitored + indices filter
	monitoredTrue := true
	indices := []int{1000, 1002, 1004}
	filter := &model.ValidatorFilterInput{
		Monitored: &monitoredTrue,
		Indices:   indices,
	}

	limit := 50
	pagination := &model.PaginationInput{
		Limit: &limit,
	}

	result, err := queryResolver.Validators(ctx, filter, nil, pagination)
	require.NoError(t, err)
	// Should only return monitored validators from the indices list
	// Since even indices are not monitored, this should return 0
	assert.Len(t, result.Edges, 0)

	// Test with monitored false
	monitoredFalse := false
	filter2 := &model.ValidatorFilterInput{
		Monitored: &monitoredFalse,
		Indices:   indices,
	}

	result2, err := queryResolver.Validators(ctx, filter2, nil, pagination)
	require.NoError(t, err)
	// Even indices are not monitored, should match
	assert.Len(t, result2.Edges, 3)
}
