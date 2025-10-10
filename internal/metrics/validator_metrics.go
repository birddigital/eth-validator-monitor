package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ValidatorMetrics provides Prometheus metrics for validator monitoring
type ValidatorMetrics struct {
	// Validator effectiveness score (0-100)
	EffectivenessScore *prometheus.GaugeVec

	// Snapshot lag time in seconds
	SnapshotLag *prometheus.GaugeVec

	// Missed attestations counter
	MissedAttestations *prometheus.CounterVec

	// Validator balance in Wei
	ValidatorBalance *prometheus.GaugeVec

	// Proposal success rate (0-1)
	ProposalSuccessRate *prometheus.GaugeVec

	// Validator status (0=offline, 1=pending, 2=active, 3=exiting, 4=slashed)
	ValidatorStatus *prometheus.GaugeVec

	// Attestation participation rate (0-1)
	AttestationRate *prometheus.GaugeVec

	// Validator rewards in Wei (cumulative)
	ValidatorRewards *prometheus.CounterVec

	// Validator penalties in Wei (cumulative)
	ValidatorPenalties *prometheus.CounterVec

	// Block proposals counter
	BlockProposals *prometheus.CounterVec

	// Successful block proposals counter
	SuccessfulProposals *prometheus.CounterVec
}

// NewValidatorMetrics creates and registers validator performance metrics
func NewValidatorMetrics() *ValidatorMetrics {
	return &ValidatorMetrics{
		EffectivenessScore: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "validator_effectiveness_score",
				Help: "Validator effectiveness score (0-100) based on attestation and proposal performance",
			},
			[]string{"validator_index", "pubkey"},
		),

		SnapshotLag: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "validator_snapshot_lag_seconds",
				Help: "Time lag between current time and last snapshot update in seconds",
			},
			[]string{"validator_index"},
		),

		MissedAttestations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "validator_missed_attestations_total",
				Help: "Total number of missed attestations for each validator",
			},
			[]string{"validator_index", "pubkey"},
		),

		ValidatorBalance: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "validator_balance_wei",
				Help: "Current validator balance in Wei",
			},
			[]string{"validator_index", "pubkey"},
		),

		ProposalSuccessRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "validator_proposal_success_rate",
				Help: "Validator block proposal success rate (0-1)",
			},
			[]string{"validator_index", "pubkey"},
		),

		ValidatorStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "validator_status",
				Help: "Validator status: 0=offline, 1=pending, 2=active, 3=exiting, 4=slashed",
			},
			[]string{"validator_index", "pubkey", "status_name"},
		),

		AttestationRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "validator_attestation_participation_rate",
				Help: "Validator attestation participation rate (0-1) over recent epochs",
			},
			[]string{"validator_index", "pubkey"},
		),

		ValidatorRewards: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "validator_rewards_wei_total",
				Help: "Cumulative validator rewards in Wei",
			},
			[]string{"validator_index", "pubkey"},
		),

		ValidatorPenalties: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "validator_penalties_wei_total",
				Help: "Cumulative validator penalties in Wei",
			},
			[]string{"validator_index", "pubkey"},
		),

		BlockProposals: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "validator_block_proposals_total",
				Help: "Total number of block proposals assigned to validator",
			},
			[]string{"validator_index", "pubkey"},
		),

		SuccessfulProposals: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "validator_successful_proposals_total",
				Help: "Total number of successful block proposals by validator",
			},
			[]string{"validator_index", "pubkey"},
		),
	}
}

// RecordEffectivenessScore records a validator's effectiveness score
func (m *ValidatorMetrics) RecordEffectivenessScore(validatorIndex int, pubkey string, score float64) {
	m.EffectivenessScore.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
	).Set(score)
}

// RecordSnapshotLag records the snapshot lag time
func (m *ValidatorMetrics) RecordSnapshotLag(validatorIndex int, lagSeconds float64) {
	m.SnapshotLag.WithLabelValues(
		fmt.Sprint(validatorIndex),
	).Set(lagSeconds)
}

// RecordMissedAttestation increments missed attestation counter
func (m *ValidatorMetrics) RecordMissedAttestation(validatorIndex int, pubkey string) {
	m.MissedAttestations.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
	).Inc()
}

// RecordBalance records validator balance
func (m *ValidatorMetrics) RecordBalance(validatorIndex int, pubkey string, balanceWei float64) {
	m.ValidatorBalance.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
	).Set(balanceWei)
}

// RecordProposalSuccessRate records proposal success rate
func (m *ValidatorMetrics) RecordProposalSuccessRate(validatorIndex int, pubkey string, rate float64) {
	m.ProposalSuccessRate.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
	).Set(rate)
}

// RecordValidatorStatus records validator status
func (m *ValidatorMetrics) RecordValidatorStatus(validatorIndex int, pubkey string, statusName string, statusValue float64) {
	m.ValidatorStatus.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
		statusName,
	).Set(statusValue)
}

// RecordAttestationRate records attestation participation rate
func (m *ValidatorMetrics) RecordAttestationRate(validatorIndex int, pubkey string, rate float64) {
	m.AttestationRate.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
	).Set(rate)
}

// RecordReward increments validator rewards
func (m *ValidatorMetrics) RecordReward(validatorIndex int, pubkey string, rewardWei float64) {
	m.ValidatorRewards.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
	).Add(rewardWei)
}

// RecordPenalty increments validator penalties
func (m *ValidatorMetrics) RecordPenalty(validatorIndex int, pubkey string, penaltyWei float64) {
	m.ValidatorPenalties.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
	).Add(penaltyWei)
}

// RecordBlockProposal increments block proposal counter
func (m *ValidatorMetrics) RecordBlockProposal(validatorIndex int, pubkey string) {
	m.BlockProposals.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
	).Inc()
}

// RecordSuccessfulProposal increments successful proposal counter
func (m *ValidatorMetrics) RecordSuccessfulProposal(validatorIndex int, pubkey string) {
	m.SuccessfulProposals.WithLabelValues(
		fmt.Sprint(validatorIndex),
		pubkey,
	).Inc()
}
