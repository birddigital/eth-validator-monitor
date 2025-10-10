package dataloader

import (
	"context"
	"fmt"

	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/graph-gophers/dataloader/v7"
)

// NewValidatorByIndexBatchFunc creates a batch function for loading validators by index
func NewValidatorByIndexBatchFunc(repo *repository.ValidatorRepository, c *cache.RedisCache) func(context.Context, []int) []*dataloader.Result[*models.Validator] {
	return func(ctx context.Context, indices []int) []*dataloader.Result[*models.Validator] {
		// Convert []int to []int64 for repository
		indices64 := make([]int64, len(indices))
		for i, idx := range indices {
			indices64[i] = int64(idx)
		}

		// Try cache first for each validator
		results := make([]*dataloader.Result[*models.Validator], len(indices))
		uncachedIndices := make([]int64, 0)
		uncachedPositions := make(map[int64]int)

		for i, idx := range indices64 {
			// Try to get from cache
			key := fmt.Sprintf("validator:%d", idx)
			var validator models.Validator
			if err := c.Get(ctx, key, &validator); err == nil {
				results[i] = &dataloader.Result[*models.Validator]{Data: &validator}
			} else {
				uncachedIndices = append(uncachedIndices, idx)
				uncachedPositions[idx] = i
				results[i] = &dataloader.Result[*models.Validator]{} // Placeholder
			}
		}

		// Fetch uncached validators from database
		if len(uncachedIndices) > 0 {
			filter := &models.ValidatorFilter{
				ValidatorIndices: uncachedIndices,
			}

			validators, err := repo.ListValidators(ctx, filter)
			if err != nil {
				// Set error for all uncached positions
				for _, pos := range uncachedPositions {
					results[pos] = &dataloader.Result[*models.Validator]{Error: err}
				}
				return results
			}

			// Map validators to their positions and cache them
			for _, v := range validators {
				if pos, ok := uncachedPositions[v.ValidatorIndex]; ok {
					results[pos] = &dataloader.Result[*models.Validator]{Data: v}

					// Cache the validator
					key := fmt.Sprintf("validator:%d", v.ValidatorIndex)
					_ = c.Set(ctx, key, v, cache.GetValidatorMetadataTTL())
				}
			}

			// Set nil for any that weren't found
			for idx, pos := range uncachedPositions {
				if results[pos].Data == nil && results[pos].Error == nil {
					results[pos] = &dataloader.Result[*models.Validator]{
						Error: fmt.Errorf("validator %d not found", idx),
					}
				}
			}
		}

		return results
	}
}

// NewValidatorByPubkeyBatchFunc creates a batch function for loading validators by pubkey
func NewValidatorByPubkeyBatchFunc(repo *repository.ValidatorRepository, c *cache.RedisCache) func(context.Context, []string) []*dataloader.Result[*models.Validator] {
	return func(ctx context.Context, pubkeys []string) []*dataloader.Result[*models.Validator] {
		results := make([]*dataloader.Result[*models.Validator], len(pubkeys))
		uncachedPubkeys := make([]string, 0)
		uncachedPositions := make(map[string]int)

		for i, pubkey := range pubkeys {
			// Try to get from cache
			key := fmt.Sprintf("validator:pubkey:%s", pubkey)
			var validator models.Validator
			if err := c.Get(ctx, key, &validator); err == nil {
				results[i] = &dataloader.Result[*models.Validator]{Data: &validator}
			} else {
				uncachedPubkeys = append(uncachedPubkeys, pubkey)
				uncachedPositions[pubkey] = i
				results[i] = &dataloader.Result[*models.Validator]{} // Placeholder
			}
		}

		// Fetch uncached validators from database
		if len(uncachedPubkeys) > 0 {
			filter := &models.ValidatorFilter{
				Pubkeys: uncachedPubkeys,
			}

			validators, err := repo.ListValidators(ctx, filter)
			if err != nil {
				// Set error for all uncached positions
				for _, pos := range uncachedPositions {
					results[pos] = &dataloader.Result[*models.Validator]{Error: err}
				}
				return results
			}

			// Map validators to their positions and cache them
			for _, v := range validators {
				if pos, ok := uncachedPositions[v.Pubkey]; ok {
					results[pos] = &dataloader.Result[*models.Validator]{Data: v}

					// Cache the validator by both index and pubkey
					keyByIndex := fmt.Sprintf("validator:%d", v.ValidatorIndex)
					keyByPubkey := fmt.Sprintf("validator:pubkey:%s", v.Pubkey)
					_ = c.Set(ctx, keyByIndex, v, cache.GetValidatorMetadataTTL())
					_ = c.Set(ctx, keyByPubkey, v, cache.GetValidatorMetadataTTL())
				}
			}

			// Set nil for any that weren't found
			for pubkey, pos := range uncachedPositions {
				if results[pos].Data == nil && results[pos].Error == nil {
					results[pos] = &dataloader.Result[*models.Validator]{
						Error: fmt.Errorf("validator with pubkey %s not found", pubkey),
					}
				}
			}
		}

		return results
	}
}
