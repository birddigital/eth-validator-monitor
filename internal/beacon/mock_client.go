package beacon

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/birddigital/eth-validator-monitor/pkg/types"
)

// MockClient is a mock implementation of the BeaconClient interface for development/testing
type MockClient struct {
	epoch int
	slot  int
}

// NewMockClient creates a new mock beacon client
func NewMockClient() *MockClient {
	return &MockClient{
		epoch: int(time.Now().Unix() / 384), // Approximate current epoch
		slot:  int(time.Now().Unix() / 12),  // Approximate current slot
	}
}

// GetValidator retrieves mock validator information by index
func (m *MockClient) GetValidator(ctx context.Context, index int) (*types.ValidatorData, error) {
	return &types.ValidatorData{
		Index:   index,
		Balance: big.NewInt(32_000_000_000 + rand.Int63n(1_000_000_000)), // 32-33 ETH in Gwei
		Status:  types.ValidatorStatusActive,
		Validator: types.ValidatorInfo{
			Pubkey:                     fmt.Sprintf("0x%096d", index),
			WithdrawalCredentials:      "0x00" + fmt.Sprintf("%062d", index),
			EffectiveBalance:           big.NewInt(32_000_000_000),
			Slashed:                    false,
			ActivationEligibilityEpoch: 0,
			ActivationEpoch:            0,
			ExitEpoch:                  2147483647,
			WithdrawableEpoch:          2147483647,
		},
	}, nil
}

// GetValidatorBalance retrieves mock validator balance
func (m *MockClient) GetValidatorBalance(ctx context.Context, index int, epoch int) (*big.Int, error) {
	return big.NewInt(32_000_000_000 + rand.Int63n(1_000_000_000)), nil
}

// GetValidatorByPubkey retrieves mock validator by pubkey
func (m *MockClient) GetValidatorByPubkey(ctx context.Context, pubkey string) (*types.ValidatorData, error) {
	return m.GetValidator(ctx, 0)
}

// GetAttestations returns mock attestations
func (m *MockClient) GetAttestations(ctx context.Context, epoch int) ([]types.Attestation, error) {
	return []types.Attestation{}, nil
}

// GetProposals returns mock proposals
func (m *MockClient) GetProposals(ctx context.Context, epoch int) ([]types.Proposal, error) {
	return []types.Proposal{}, nil
}

// SubscribeToHeadEvents creates a channel that emits mock head events every 12 seconds
func (m *MockClient) SubscribeToHeadEvents(ctx context.Context) (<-chan types.HeadEvent, error) {
	ch := make(chan types.HeadEvent, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(12 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.slot++
				if m.slot%32 == 0 {
					m.epoch++
				}

				ch <- types.HeadEvent{
					Slot:      m.slot,
					Block:     fmt.Sprintf("0x%064d", m.slot),
					State:     fmt.Sprintf("0x%064d", m.slot),
					Timestamp: time.Now(),
				}
			}
		}
	}()

	return ch, nil
}

// SubscribeToHead is an alias for SubscribeToHeadEvents
func (m *MockClient) SubscribeToHead(ctx context.Context) (<-chan types.HeadEvent, error) {
	return m.SubscribeToHeadEvents(ctx)
}

// GetCurrentEpoch returns the mock current epoch
func (m *MockClient) GetCurrentEpoch(ctx context.Context) (int, error) {
	return m.epoch, nil
}

// GetCurrentSlot returns the mock current slot
func (m *MockClient) GetCurrentSlot(ctx context.Context) (int, error) {
	return m.slot, nil
}

// GetNetworkStats returns mock network statistics
func (m *MockClient) GetNetworkStats(ctx context.Context) (*types.NetworkStats, error) {
	return &types.NetworkStats{
		CurrentEpoch:       m.epoch,
		CurrentSlot:        m.slot,
		TotalValidators:    1_000_000,
		ActiveValidators:   950_000,
		PendingValidators:  25_000,
		ExitingValidators:  5_000,
		SlashedValidators:  100,
		AverageBalance:     big.NewInt(32_500_000_000),
		TotalStaked:        big.NewInt(30_400_000_000_000_000),
		ParticipationRate:  0.95,
		Timestamp:          time.Now(),
	}, nil
}
