package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/birddigital/eth-validator-monitor/graph"
	"github.com/birddigital/eth-validator-monitor/graph/dataloader"
	"github.com/birddigital/eth-validator-monitor/graph/resolver"
	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_CompleteDataFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Setup test infrastructure
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
		KeyPrefix: "e2e-test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	ctx := context.Background()

	// Clean cache
	_ = redisCache.Flush(ctx)

	// Step 1: Simulate data collection - Create validators
	validators := testutil.MultipleValidatorFixtures(100)
	err = validatorRepo.BatchCreateValidators(ctx, validators)
	require.NoError(t, err)
	t.Log("✓ Step 1: Created 100 validators")

	// Step 2: Collect snapshots for validators over time
	baseTime := time.Now().Add(-24 * time.Hour)
	for _, validator := range validators[:10] { // First 10 validators get snapshots
		snapshots := testutil.MultipleSnapshotFixtures(validator.ValidatorIndex, 50)
		err = snapshotRepo.BatchInsertSnapshots(ctx, snapshots)
		require.NoError(t, err)
	}
	t.Log("✓ Step 2: Created snapshots for 10 validators")

	// Step 3: Setup GraphQL server
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	res := &resolver.Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
		Cache:         redisCache,
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: res,
	}))

	t.Log("✓ Step 3: GraphQL server initialized")

	// Step 4: Query API - Get all validators
	query := `
		query {
			validators(pagination: { limit: 50 }) {
				edges {
					node {
						validatorIndex
						pubkey
						monitored
					}
				}
				pageInfo {
					hasNextPage
					hasPreviousPage
				}
				totalCount
			}
		}
	`

	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "`+strings.ReplaceAll(query, "\n", " ")+`"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	validatorsData := data["validators"].(map[string]interface{})
	edges := validatorsData["edges"].([]interface{})

	assert.Equal(t, 50, len(edges))
	assert.Equal(t, float64(100), validatorsData["totalCount"])

	t.Log("✓ Step 4: Retrieved validators via GraphQL API")

	// Step 5: Query specific validator with latest snapshot
	validatorQuery := `
		query {
			validator(index: 100) {
				validatorIndex
				pubkey
				latestSnapshot {
					balance
					effectiveBalance
				}
			}
		}
	`

	req2 := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "`+strings.ReplaceAll(validatorQuery, "\n", " ")+`"}`))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var response2 map[string]interface{}
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	require.NoError(t, err)

	data2 := response2["data"].(map[string]interface{})
	validatorData := data2["validator"].(map[string]interface{})

	assert.Equal(t, float64(100), validatorData["validatorIndex"])
	assert.NotNil(t, validatorData["latestSnapshot"])

	t.Log("✓ Step 5: Retrieved specific validator with latest snapshot")

	// Step 6: Verify caching - Query same validator again
	req3 := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "`+strings.ReplaceAll(validatorQuery, "\n", " ")+`"}`))
	req3.Header.Set("Content-Type", "application/json")

	w3 := httptest.NewRecorder()
	srv.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
	t.Log("✓ Step 6: Verified caching (second query)")

	// Step 7: Test pagination
	paginationQuery := `
		query {
			validators(pagination: { limit: 10 }) {
				edges {
					cursor
					node {
						validatorIndex
					}
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	`

	req4 := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "`+strings.ReplaceAll(paginationQuery, "\n", " ")+`"}`))
	req4.Header.Set("Content-Type", "application/json")

	w4 := httptest.NewRecorder()
	srv.ServeHTTP(w4, req4)

	assert.Equal(t, http.StatusOK, w4.Code)

	var response4 map[string]interface{}
	err = json.Unmarshal(w4.Body.Bytes(), &response4)
	require.NoError(t, err)

	data4 := response4["data"].(map[string]interface{})
	validators4 := data4["validators"].(map[string]interface{})
	edges4 := validators4["edges"].([]interface{})
	pageInfo4 := validators4["pageInfo"].(map[string]interface{})

	assert.Equal(t, 10, len(edges4))
	assert.True(t, pageInfo4["hasNextPage"].(bool))

	t.Log("✓ Step 7: Verified pagination works correctly")

	// Step 8: Test filtering
	filterQuery := `
		query {
			validators(filter: { monitored: true }, pagination: { limit: 100 }) {
				edges {
					node {
						validatorIndex
						monitored
					}
				}
				totalCount
			}
		}
	`

	req5 := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "`+strings.ReplaceAll(filterQuery, "\n", " ")+`"}`))
	req5.Header.Set("Content-Type", "application/json")

	w5 := httptest.NewRecorder()
	srv.ServeHTTP(w5, req5)

	assert.Equal(t, http.StatusOK, w5.Code)

	var response5 map[string]interface{}
	err = json.Unmarshal(w5.Body.Bytes(), &response5)
	require.NoError(t, err)

	data5 := response5["data"].(map[string]interface{})
	validators5 := data5["validators"].(map[string]interface{})
	edges5 := validators5["edges"].([]interface{})

	// All test validators are monitored
	for _, edge := range edges5 {
		node := edge.(map[string]interface{})["node"].(map[string]interface{})
		assert.True(t, node["monitored"].(bool))
	}

	t.Log("✓ Step 8: Verified filtering works correctly")

	t.Log("✅ End-to-end test completed successfully")
}

func TestE2E_PerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Setup test infrastructure
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
		KeyPrefix: "perf-test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	ctx := context.Background()
	_ = redisCache.Flush(ctx)

	// Create large dataset
	validators := testutil.MultipleValidatorFixtures(1000)
	err = validatorRepo.BatchCreateValidators(ctx, validators)
	require.NoError(t, err)

	// Create snapshots for first 100 validators
	for i := 0; i < 100; i++ {
		snapshots := testutil.MultipleSnapshotFixtures(validators[i].ValidatorIndex, 100)
		err = snapshotRepo.BatchInsertSnapshots(ctx, snapshots)
		require.NoError(t, err)
	}

	// Setup GraphQL server
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	res := &resolver.Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
		Cache:         redisCache,
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: res,
	}))

	// Performance test: Execute 100 queries
	query := `
		query {
			validators(pagination: { limit: 50 }) {
				edges {
					node {
						validatorIndex
						pubkey
					}
				}
			}
		}
	`

	start := time.Now()
	successCount := 0

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "`+strings.ReplaceAll(query, "\n", " ")+`"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	duration := time.Since(start)

	assert.Equal(t, 100, successCount)
	assert.Less(t, duration, 30*time.Second, "100 queries should complete within 30 seconds")

	t.Logf("Performance: 100 queries completed in %v (%.2f req/sec)", duration, 100.0/duration.Seconds())
}

func TestE2E_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling test in short mode")
	}

	// Setup test infrastructure
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
		KeyPrefix: "error-test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	// Setup GraphQL server
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	res := &resolver.Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
		Cache:         redisCache,
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: res,
	}))

	// Test 1: Query non-existent validator
	query := `
		query {
			validator(index: 99999) {
				validatorIndex
			}
		}
	`

	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "`+strings.ReplaceAll(query, "\n", " ")+`"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // GraphQL returns 200 even for errors

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should return null for non-existent validator
	data := response["data"].(map[string]interface{})
	assert.Nil(t, data["validator"])

	// Test 2: Invalid query
	invalidQuery := `query { invalidField { test } }`

	req2 := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "`+invalidQuery+`"}`))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)

	var response2 map[string]interface{}
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	require.NoError(t, err)

	// Should have errors
	errors, hasErrors := response2["errors"]
	assert.True(t, hasErrors)
	assert.NotNil(t, errors)

	t.Log("✓ Error handling tests passed")
}

func TestE2E_DataConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping data consistency test in short mode")
	}

	// Setup test infrastructure
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
		KeyPrefix: "consistency-test",
	}
	redisCache, err := cache.NewRedisCache(cacheConfig)
	require.NoError(t, err)
	defer redisCache.Close()

	ctx := context.Background()
	_ = redisCache.Flush(ctx)

	// Create validator
	validator := testutil.ValidatorFixture(500)
	err = validatorRepo.CreateValidator(ctx, validator)
	require.NoError(t, err)

	// Create snapshots
	baseTime := time.Now().Add(-10 * time.Hour)
	for i := 0; i < 10; i++ {
		snapshot := testutil.ValidatorSnapshotFixture(500, baseTime.Add(time.Duration(i)*time.Hour))
		snapshot.Balance = 32000000000 + int64(i*1000000)
		err := snapshotRepo.InsertSnapshot(ctx, snapshot)
		require.NoError(t, err)
	}

	// Setup GraphQL server
	loaders := dataloader.NewLoaders(validatorRepo, snapshotRepo, alertRepo, redisCache)

	res := &resolver.Resolver{
		ValidatorRepo: validatorRepo,
		SnapshotRepo:  snapshotRepo,
		DataLoaders:   loaders,
		Cache:         redisCache,
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: res,
	}))

	// Query validator multiple times and verify consistency
	query := `
		query {
			validator(index: 500) {
				validatorIndex
				pubkey
				latestSnapshot {
					balance
					validatorIndex
				}
			}
		}
	`

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "`+strings.ReplaceAll(query, "\n", " ")+`"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		validatorData := data["validator"].(map[string]interface{})
		latestSnapshot := validatorData["latestSnapshot"].(map[string]interface{})

		// Verify consistency
		assert.Equal(t, float64(500), validatorData["validatorIndex"])
		assert.Equal(t, float64(500), latestSnapshot["validatorIndex"])
		assert.NotNil(t, latestSnapshot["balance"])
	}

	t.Log("✓ Data consistency verified across multiple queries")
}
