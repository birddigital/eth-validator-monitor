package types

import (
	"math/big"
	"time"
)

// ValidatorStatus represents the current state of a validator
type ValidatorStatus string

const (
	StatusPending  ValidatorStatus = "pending"
	StatusActive   ValidatorStatus = "active"
	StatusExiting  ValidatorStatus = "exiting"
	StatusExited   ValidatorStatus = "exited"
	StatusSlashed  ValidatorStatus = "slashed"
	StatusUnknown  ValidatorStatus = "unknown"
)

// Validator represents an Ethereum validator
type Validator struct {
	Index           int             `json:"index"`
	Pubkey          string          `json:"pubkey"`
	Name            string          `json:"name,omitempty"`
	Status          ValidatorStatus `json:"status"`
	ActivationEpoch *int            `json:"activation_epoch,omitempty"`
	ExitEpoch       *int            `json:"exit_epoch,omitempty"`
	Slashed         bool            `json:"slashed"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ValidatorBalance represents balance information for a validator
type ValidatorBalance struct {
	Current      *big.Int `json:"current"`       // Current balance in Gwei
	Effective    *big.Int `json:"effective"`     // Effective balance in Gwei
	Withdrawable *big.Int `json:"withdrawable"`  // Withdrawable balance in Gwei
}

// ValidatorSnapshot represents a point-in-time snapshot of validator state
type ValidatorSnapshot struct {
	ID                   int64     `json:"id"`
	ValidatorIndex       int       `json:"validator_index"`
	Epoch                int       `json:"epoch"`
	Slot                 int       `json:"slot"`
	Timestamp            time.Time `json:"timestamp"`
	Balance              *big.Int  `json:"balance"`
	EffectiveBalance     *big.Int  `json:"effective_balance"`
	AttestationSuccess   *bool     `json:"attestation_success,omitempty"`
	InclusionDelay       *int      `json:"inclusion_delay,omitempty"`
	ProposalSuccess      *bool     `json:"proposal_success,omitempty"`
	PerformanceScore     *float64  `json:"performance_score,omitempty"`
	NetworkPercentile    *float64  `json:"network_percentile,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

// ValidatorPerformance represents performance metrics
type ValidatorPerformance struct {
	ValidatorIndex      int       `json:"validator_index"`
	Timestamp           time.Time `json:"timestamp"`

	// Uptime metrics
	UptimePercentage    float64   `json:"uptime_percentage"`
	ConsecutiveMisses   int       `json:"consecutive_misses"`
	TotalMissed         int       `json:"total_missed"`

	// Effectiveness metrics
	AttestationScore    float64   `json:"attestation_score"`
	ProposalSuccess     int       `json:"proposal_success"`
	ProposalMissed      int       `json:"proposal_missed"`

	// Rewards metrics
	ExpectedRewards     *big.Int  `json:"expected_rewards"`
	ActualRewards       *big.Int  `json:"actual_rewards"`
	Effectiveness       float64   `json:"effectiveness"`

	// Comparative metrics
	NetworkAverage      float64   `json:"network_average"`
	Percentile          float64   `json:"percentile"`

	// Risk indicators
	SlashingRisk        RiskLevel `json:"slashing_risk"`
	InactivityScore     int       `json:"inactivity_score"`
}

// RiskLevel represents the level of risk
type RiskLevel string

const (
	RiskNone   RiskLevel = "none"
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// ValidatorFilter is used for querying validators
type ValidatorFilter struct {
	Status   *ValidatorStatus `json:"status,omitempty"`
	Slashed  *bool            `json:"slashed,omitempty"`
	Indices  []int            `json:"indices,omitempty"`
	Pubkeys  []string         `json:"pubkeys,omitempty"`
	Limit    int              `json:"limit,omitempty"`
	Offset   int              `json:"offset,omitempty"`
}

// PerformanceMetrics represents validator performance metrics for storage
type PerformanceMetrics struct {
	ValidatorIndex       int       `json:"validator_index"`
	Epoch                int       `json:"epoch"`
	AttestationScore     float64   `json:"attestation_score"`
	ProposalScore        float64   `json:"proposal_score"`
	SyncCommitteeScore   float64   `json:"sync_committee_score"`
	OverallScore         float64   `json:"overall_score"`
	Timestamp            time.Time `json:"timestamp"`
}
