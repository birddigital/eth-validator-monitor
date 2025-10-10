package dataloader

import (
	"context"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/cache"
	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/graph-gophers/dataloader/v7"
)

// Loaders contains all DataLoaders for the application
type Loaders struct {
	ValidatorByIndex    *dataloader.Loader[int, *models.Validator]
	ValidatorByPubkey   *dataloader.Loader[string, *models.Validator]
	SnapshotsByValidator *dataloader.Loader[int, []*models.ValidatorSnapshot]
	AlertsByValidator   *dataloader.Loader[int, []*models.Alert]
	LatestSnapshotByValidator *dataloader.Loader[int, *models.ValidatorSnapshot]
}

// NewLoaders creates a new instance of Loaders
func NewLoaders(
	validatorRepo *repository.ValidatorRepository,
	snapshotRepo *repository.SnapshotRepository,
	alertRepo *repository.AlertRepository,
	cache *cache.RedisCache,
) *Loaders {
	// Configure DataLoader options
	options := []dataloader.Option[int, *models.Validator]{
		dataloader.WithBatchCapacity[int, *models.Validator](100),
		dataloader.WithWait[int, *models.Validator](16 * time.Millisecond),
	}

	return &Loaders{
		ValidatorByIndex: dataloader.NewBatchedLoader(
			NewValidatorByIndexBatchFunc(validatorRepo, cache),
			options...,
		),
		ValidatorByPubkey: dataloader.NewBatchedLoader(
			NewValidatorByPubkeyBatchFunc(validatorRepo, cache),
			dataloader.WithBatchCapacity[string, *models.Validator](100),
			dataloader.WithWait[string, *models.Validator](16*time.Millisecond),
		),
		SnapshotsByValidator: dataloader.NewBatchedLoader(
			NewSnapshotsByValidatorBatchFunc(snapshotRepo, cache),
			dataloader.WithBatchCapacity[int, []*models.ValidatorSnapshot](100),
			dataloader.WithWait[int, []*models.ValidatorSnapshot](16*time.Millisecond),
		),
		AlertsByValidator: dataloader.NewBatchedLoader(
			NewAlertsByValidatorBatchFunc(alertRepo, cache),
			dataloader.WithBatchCapacity[int, []*models.Alert](100),
			dataloader.WithWait[int, []*models.Alert](16*time.Millisecond),
		),
		LatestSnapshotByValidator: dataloader.NewBatchedLoader(
			NewLatestSnapshotByValidatorBatchFunc(snapshotRepo, cache),
			dataloader.WithBatchCapacity[int, *models.ValidatorSnapshot](100),
			dataloader.WithWait[int, *models.ValidatorSnapshot](16*time.Millisecond),
		),
	}
}

// Load validator by index
func (l *Loaders) LoadValidator(ctx context.Context, index int) (*models.Validator, error) {
	return l.ValidatorByIndex.Load(ctx, index)()
}

// Load validator by pubkey
func (l *Loaders) LoadValidatorByPubkey(ctx context.Context, pubkey string) (*models.Validator, error) {
	return l.ValidatorByPubkey.Load(ctx, pubkey)()
}

// Load snapshots for a validator
func (l *Loaders) LoadSnapshots(ctx context.Context, validatorIndex int) ([]*models.ValidatorSnapshot, error) {
	return l.SnapshotsByValidator.Load(ctx, validatorIndex)()
}

// Load alerts for a validator
func (l *Loaders) LoadAlerts(ctx context.Context, validatorIndex int) ([]*models.Alert, error) {
	return l.AlertsByValidator.Load(ctx, validatorIndex)()
}

// Load latest snapshot for a validator
func (l *Loaders) LoadLatestSnapshot(ctx context.Context, validatorIndex int) (*models.ValidatorSnapshot, error) {
	return l.LatestSnapshotByValidator.Load(ctx, validatorIndex)()
}

// contextKey is a unique type for context keys
type contextKey string

const loadersKey = contextKey("dataloaders")

// ContextWithLoaders attaches loaders to the context
func ContextWithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey, loaders)
}

// LoadersFromContext retrieves loaders from context
func LoadersFromContext(ctx context.Context) *Loaders {
	loaders, ok := ctx.Value(loadersKey).(*Loaders)
	if !ok {
		panic("dataloaders not found in context")
	}
	return loaders
}
