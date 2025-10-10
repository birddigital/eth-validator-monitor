package dataloader

import (
	"context"
	"fmt"

	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/graph-gophers/dataloader/v7"
)

// NewAlertsByValidatorBatchFunc creates a batch function for loading alerts by validator
func NewAlertsByValidatorBatchFunc(repo *repository.AlertRepository, c *cache.RedisCache) func(context.Context, []int) []*dataloader.Result[[]*models.Alert] {
	return func(ctx context.Context, validatorIndices []int) []*dataloader.Result[[]*models.Alert] {
		results := make([]*dataloader.Result[[]*models.Alert], len(validatorIndices))

		for i, idx := range validatorIndices {
			idx64 := int64(idx)

			// Try cache
			key := fmt.Sprintf("alerts:validator:%d:active", idx64)
			var alerts []*models.Alert
			if err := c.Get(ctx, key, &alerts); err == nil {
				results[i] = &dataloader.Result[[]*models.Alert]{Data: alerts}
				continue
			}

			// Fetch from database
			activeStatus := models.AlertStatusActive
			filter := &models.AlertFilter{
				ValidatorIndex: &idx64,
				Status:         &activeStatus,
				Limit:          100,
			}

			alerts, err := repo.ListAlerts(ctx, filter)
			if err != nil {
				results[i] = &dataloader.Result[[]*models.Alert]{Error: err}
				continue
			}

			results[i] = &dataloader.Result[[]*models.Alert]{Data: alerts}

			// Cache the results
			_ = c.Set(ctx, key, alerts, cache.GetAlertCacheTTL())
		}

		return results
	}
}
