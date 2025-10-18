package fixtures

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Validator represents a simplified validator for benchmarking
type Validator struct {
	Index      uint64
	Pubkey     []byte
	Status     string
	Balance    uint64
	Activation uint64
}

// ValidatorSnapshot represents a performance snapshot for benchmarking
type ValidatorSnapshot struct {
	ValidatorIndex     uint64
	Timestamp          time.Time
	Balance            uint64
	EffectiveBalance   uint64
	Effectiveness      float64
	MissedAttestations uint64
	ProposalSuccess    bool
	Epoch              uint64
	Slot               uint64
}

// GenerateValidators creates realistic validator data for benchmarking
func GenerateValidators(count int) []*Validator {
	validators := make([]*Validator, count)

	for i := 0; i < count; i++ {
		pubkey := make([]byte, 48)
		rand.Read(pubkey)

		validators[i] = &Validator{
			Index:      uint64(i),
			Pubkey:     pubkey,
			Status:     "active_ongoing",
			Balance:    32000000000 + uint64(i*1000), // 32 ETH in Gwei + variation
			Activation: 0,
		}
	}

	return validators
}

// GenerateSnapshot creates a single realistic snapshot
func GenerateSnapshot() *ValidatorSnapshot {
	pubkey := make([]byte, 48)
	rand.Read(pubkey)

	return &ValidatorSnapshot{
		ValidatorIndex:     1,
		Timestamp:          time.Now(),
		Balance:            32000000000,
		EffectiveBalance:   32000000000,
		Effectiveness:      0.98,
		MissedAttestations: 2,
		ProposalSuccess:    true,
		Epoch:              1000,
		Slot:               32000,
	}
}

// GenerateSnapshots creates realistic snapshot data for benchmarking
func GenerateSnapshots(count int) []*ValidatorSnapshot {
	snapshots := make([]*ValidatorSnapshot, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		// Create realistic variance in effectiveness scores
		effectiveness := 0.95 + (float64(i%100) / 1000)
		if effectiveness > 1.0 {
			effectiveness = 1.0
		}

		snapshots[i] = &ValidatorSnapshot{
			ValidatorIndex:     uint64(i),
			Timestamp:          now.Add(-time.Duration(i) * time.Minute),
			Balance:            32000000000 + uint64(i*1000),
			EffectiveBalance:   32000000000,
			Effectiveness:      effectiveness,
			MissedAttestations: uint64(i % 10),
			ProposalSuccess:    i%3 == 0,
			Epoch:              uint64(1000 + i/32),
			Slot:               uint64(32000 + i),
		}
	}

	return snapshots
}

// GenerateSnapshotsForTimeRange creates snapshots across a time range
func GenerateSnapshotsForTimeRange(validatorCount int, startTime, endTime time.Time, interval time.Duration) []*ValidatorSnapshot {
	var snapshots []*ValidatorSnapshot

	for t := startTime; t.Before(endTime); t = t.Add(interval) {
		for i := 0; i < validatorCount; i++ {
			effectiveness := 0.95 + (float64(i%100) / 1000)
			if effectiveness > 1.0 {
				effectiveness = 1.0
			}

			snapshots = append(snapshots, &ValidatorSnapshot{
				ValidatorIndex:     uint64(i),
				Timestamp:          t,
				Balance:            32000000000 + uint64(i*1000),
				EffectiveBalance:   32000000000,
				Effectiveness:      effectiveness,
				MissedAttestations: uint64(i % 10),
				ProposalSuccess:    i%3 == 0,
				Epoch:              uint64(1000 + (t.Unix() / 384)), // ~6.4 min epoch
				Slot:               uint64(32000 + (t.Unix() / 12)), // 12 sec slots
			})
		}
	}

	return snapshots
}

// MockBeaconClient simulates a Beacon Chain client for benchmarking
type MockBeaconClient struct {
	validators     []*Validator
	failureRate    float64
	requestCount   int
	requestLatency time.Duration
}

// NewMockBeaconClient creates a mock Beacon client with validators
func NewMockBeaconClient(validators []*Validator) *MockBeaconClient {
	return &MockBeaconClient{
		validators:     validators,
		failureRate:    0.0,
		requestLatency: 50 * time.Millisecond,
	}
}

// NewMockBeaconClientWithFailures creates a mock client with configurable failure rate
func NewMockBeaconClientWithFailures(failureRate float64) *MockBeaconClient {
	return &MockBeaconClient{
		validators:     GenerateValidators(1000),
		failureRate:    failureRate,
		requestLatency: 50 * time.Millisecond,
	}
}

// GetValidatorStatus simulates fetching validator status
func (m *MockBeaconClient) GetValidatorStatus(index uint64) (*Validator, error) {
	m.requestCount++

	// Simulate network latency
	time.Sleep(m.requestLatency)

	// Simulate failures
	if m.failureRate > 0 {
		randBytes := make([]byte, 1)
		rand.Read(randBytes)
		if float64(randBytes[0])/255.0 < m.failureRate {
			return nil, fmt.Errorf("simulated network error")
		}
	}

	if index < uint64(len(m.validators)) {
		return m.validators[index], nil
	}

	return nil, fmt.Errorf("validator not found")
}

// GetValidatorBalance simulates fetching validator balance
func (m *MockBeaconClient) GetValidatorBalance(index uint64) (uint64, error) {
	m.requestCount++
	time.Sleep(m.requestLatency)

	if index < uint64(len(m.validators)) {
		return m.validators[index].Balance, nil
	}

	return 0, fmt.Errorf("validator not found")
}
