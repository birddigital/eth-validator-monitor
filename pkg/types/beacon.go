package types

import (
	"context"
	"math/big"
	"time"
)

// BeaconClient defines the interface for interacting with an Ethereum beacon chain
type BeaconClient interface {
	// GetValidator retrieves validator information by index
	GetValidator(ctx context.Context, index int) (*ValidatorData, error)

	// GetValidatorBalance retrieves the balance for a validator at a specific epoch
	GetValidatorBalance(ctx context.Context, index int, epoch int) (*big.Int, error)

	// GetValidatorByPubkey retrieves validator information by public key
	GetValidatorByPubkey(ctx context.Context, pubkey string) (*ValidatorData, error)

	// GetAttestations retrieves attestations for a specific epoch
	GetAttestations(ctx context.Context, epoch int) ([]Attestation, error)

	// GetProposals retrieves block proposals for a specific epoch
	GetProposals(ctx context.Context, epoch int) ([]Proposal, error)

	// SubscribeToHeadEvents subscribes to new beacon chain head events
	SubscribeToHeadEvents(ctx context.Context) (<-chan HeadEvent, error)

	// SubscribeToHead is an alias for SubscribeToHeadEvents (for compatibility)
	SubscribeToHead(ctx context.Context) (<-chan HeadEvent, error)

	// GetCurrentEpoch retrieves the current epoch number
	GetCurrentEpoch(ctx context.Context) (int, error)

	// GetCurrentSlot retrieves the current slot number
	GetCurrentSlot(ctx context.Context) (int, error)

	// GetNetworkStats retrieves network-wide statistics
	GetNetworkStats(ctx context.Context) (*NetworkStats, error)
}

// ValidatorData represents data from the beacon chain about a validator
type ValidatorData struct {
	Index                int             `json:"index"`
	Balance              *big.Int        `json:"balance"`
	Status               ValidatorStatus `json:"status"`
	Validator            ValidatorInfo   `json:"validator"`
}

// ValidatorInfo contains the core validator information
type ValidatorInfo struct {
	Pubkey                     string   `json:"pubkey"`
	WithdrawalCredentials      string   `json:"withdrawal_credentials"`
	EffectiveBalance           *big.Int `json:"effective_balance"`
	Slashed                    bool     `json:"slashed"`
	ActivationEligibilityEpoch int      `json:"activation_eligibility_epoch"`
	ActivationEpoch            int      `json:"activation_epoch"`
	ExitEpoch                  int      `json:"exit_epoch"`
	WithdrawableEpoch          int      `json:"withdrawable_epoch"`
}

// Attestation represents a validator attestation
type Attestation struct {
	AggregationBits string          `json:"aggregation_bits"`
	Data            AttestationData `json:"data"`
	Signature       string          `json:"signature"`
}

// AttestationData contains the attestation details
type AttestationData struct {
	Slot            int    `json:"slot"`
	Index           int    `json:"index"`
	BeaconBlockRoot string `json:"beacon_block_root"`
	Source          Checkpoint `json:"source"`
	Target          Checkpoint `json:"target"`
}

// Checkpoint represents a checkpoint in the beacon chain
type Checkpoint struct {
	Epoch int    `json:"epoch"`
	Root  string `json:"root"`
}

// Proposal represents a block proposal
type Proposal struct {
	Slot      int    `json:"slot"`
	Proposer  int    `json:"proposer"`
	BlockRoot string `json:"block_root"`
	Timestamp time.Time `json:"timestamp"`
}

// HeadEvent represents a beacon chain head update event
type HeadEvent struct {
	Slot  int       `json:"slot"`
	Block string    `json:"block"`
	State string    `json:"state"`
	Timestamp time.Time `json:"timestamp"`
}

// NetworkStats represents network-wide statistics
type NetworkStats struct {
	CurrentEpoch         int       `json:"current_epoch"`
	CurrentSlot          int       `json:"current_slot"`
	TotalValidators      int       `json:"total_validators"`
	ActiveValidators     int       `json:"active_validators"`
	PendingValidators    int       `json:"pending_validators"`
	ExitingValidators    int       `json:"exiting_validators"`
	SlashedValidators    int       `json:"slashed_validators"`
	AverageBalance       *big.Int  `json:"average_balance"`
	TotalStaked          *big.Int  `json:"total_staked"`
	ParticipationRate    float64   `json:"participation_rate"`
	Timestamp            time.Time `json:"timestamp"`
}
