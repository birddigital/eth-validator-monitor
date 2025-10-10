package dataloader

import (
	"context"
	"fmt"

	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/graph-gophers/dataloader/v7"
)

// NewSnapshotsByValidatorBatchFunc creates a batch function for loading snapshots by validator
func NewSnapshotsByValidatorBatchFunc(repo *repository.SnapshotRepository, c *cache.RedisCache) func(context.Context, []int) []*dataloader.Result[[]*models.ValidatorSnapshot] {
	return func(ctx context.Context, validatorIndices []int) []*dataloader.Result[[]*models.ValidatorSnapshot] {
		results := make([]*dataloader.Result[[]*models.ValidatorSnapshot], len(validatorIndices))

		for i, idx := range validatorIndices {
			idx64 := int64(idx)

			// Try cache first
			key := fmt.Sprintf("snapshots:validator:%d:recent", idx64)
			var snapshots []*models.ValidatorSnapshot
			if err := c.Get(ctx, key, &snapshots); err == nil {
				results[i] = &dataloader.Result[[]*models.ValidatorSnapshot]{Data: snapshots}
				continue
			}

			// Fetch from database
			snapshots, err := repo.GetRecentSnapshots(ctx, idx64, 50)
			if err != nil {
				results[i] = &dataloader.Result[[]*models.ValidatorSnapshot]{Error: err}
				continue
			}

			results[i] = &dataloader.Result[[]*models.ValidatorSnapshot]{Data: snapshots}

			// Cache the results
			_ = c.Set(ctx, key, snapshots, cache.GetValidatorSnapshotTTL())
		}

		return results
	}
}

// NewLatestSnapshotByValidatorBatchFunc creates a batch function for loading latest snapshots
func NewLatestSnapshotByValidatorBatchFunc(repo *repository.SnapshotRepository, c *cache.RedisCache) func(context.Context, []int) []*dataloader.Result[*models.ValidatorSnapshot] {
	return func(ctx context.Context, validatorIndices []int) []*dataloader.Result[*models.ValidatorSnapshot] {
		results := make([]*dataloader.Result[*models.ValidatorSnapshot], len(validatorIndices))

		for i, idx := range validatorIndices {
			idx64 := int64(idx)

			// Try cache
			key := c.LatestSnapshotKey(idx64)
			var snapshot models.ValidatorSnapshot
			if err := c.Get(ctx, key, &snapshot); err == nil {
				results[i] = &dataloader.Result[*models.ValidatorSnapshot]{Data: &snapshot}
				continue
			}

			// Fetch from database
			snapshot_ptr, err := repo.GetLatestSnapshot(ctx, idx64)
			if err != nil {
				results[i] = &dataloader.Result[*models.ValidatorSnapshot]{Error: err}
				continue
			}

			results[i] = &dataloader.Result[*models.ValidatorSnapshot]{Data: snapshot_ptr}

			// Cache the result
			_ = c.Set(ctx, key, snapshot_ptr, cache.GetLatestSnapshotTTL())
		}

		return results
	}
}
