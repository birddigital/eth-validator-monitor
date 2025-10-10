package models

import (
	"database/sql/driver"
	"strings"
	"time"
)

// Validator represents an Ethereum validator
type Validator struct {
	ID                         int32     `db:"id"`
	ValidatorIndex             int64     `db:"validator_index"`
	Pubkey                     string    `db:"pubkey"`
	WithdrawalCredentials      *string   `db:"withdrawal_credentials"`
	EffectiveBalance           int64     `db:"effective_balance"`
	Slashed                    bool      `db:"slashed"`
	ActivationEpoch            *int64    `db:"activation_epoch"`
	ActivationEligibilityEpoch *int64    `db:"activation_eligibility_epoch"`
	ExitEpoch                  *int64    `db:"exit_epoch"`
	WithdrawableEpoch          *int64    `db:"withdrawable_epoch"`
	Name                       *string   `db:"name"`
	Tags                       Tags      `db:"tags"`
	Monitored                  bool      `db:"monitored"`
	CreatedAt                  time.Time `db:"created_at"`
	UpdatedAt                  time.Time `db:"updated_at"`
}

// ValidatorSnapshot represents a point-in-time validator state
type ValidatorSnapshot struct {
	Time                       time.Time `db:"time"`
	ValidatorIndex             int64     `db:"validator_index"`
	Balance                    int64     `db:"balance"`
	EffectiveBalance           int64     `db:"effective_balance"`
	AttestationEffectiveness   *float64  `db:"attestation_effectiveness"`
	AttestationInclusionDelay  *int32    `db:"attestation_inclusion_delay"`
	AttestationHeadVote        *bool     `db:"attestation_head_vote"`
	AttestationSourceVote      *bool     `db:"attestation_source_vote"`
	AttestationTargetVote      *bool     `db:"attestation_target_vote"`
	ProposalsScheduled         int32     `db:"proposals_scheduled"`
	ProposalsExecuted          int32     `db:"proposals_executed"`
	ProposalsMissed            int32     `db:"proposals_missed"`
	SyncCommitteeParticipation bool      `db:"sync_committee_participation"`
	Slashed                    bool      `db:"slashed"`
	IsOnline                   bool      `db:"is_online"`
	ConsecutiveMissedAttestations int32  `db:"consecutive_missed_attestations"`
	DailyIncome                *int64    `db:"daily_income"`
	APR                        *float64  `db:"apr"`
}

// Alert represents a validator alert
type Alert struct {
	ID             int32      `db:"id"`
	ValidatorIndex *int64     `db:"validator_index"`
	AlertType      string     `db:"alert_type"`
	Severity       Severity   `db:"severity"`
	Title          string     `db:"title"`
	Message        string     `db:"message"`
	Details        JSONB      `db:"details"`
	Status         AlertStatus `db:"status"`
	AcknowledgedAt *time.Time `db:"acknowledged_at"`
	ResolvedAt     *time.Time `db:"resolved_at"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// AggregatedMetrics represents pre-computed metrics
type AggregatedMetrics struct {
	Time               time.Time     `db:"time"`
	ValidatorIndex     int64         `db:"validator_index"`
	IntervalType       IntervalType  `db:"interval_type"`
	AvgBalance         *int64        `db:"avg_balance"`
	MinBalance         *int64        `db:"min_balance"`
	MaxBalance         *int64        `db:"max_balance"`
	AvgEffectiveness   *float64      `db:"avg_effectiveness"`
	MinEffectiveness   *float64      `db:"min_effectiveness"`
	MaxEffectiveness   *float64      `db:"max_effectiveness"`
	TotalAttestations  *int32        `db:"total_attestations"`
	MissedAttestations *int32        `db:"missed_attestations"`
	ParticipationRate  *float64      `db:"participation_rate"`
	TotalIncome        *int64        `db:"total_income"`
	AvgAPR             *float64      `db:"avg_apr"`
	UptimePercentage   *float64      `db:"uptime_percentage"`
}

// Tags represents an array of strings for PostgreSQL
type Tags []string

// Value implements driver.Valuer interface
func (t Tags) Value() (driver.Value, error) {
	if len(t) == 0 {
		return "{}", nil
	}
	return "{" + strings.Join(t, ",") + "}", nil
}

// Scan implements sql.Scanner interface
func (t *Tags) Scan(value interface{}) error {
	if value == nil {
		*t = Tags{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		str := string(v)
		str = strings.Trim(str, "{}")
		if str == "" {
			*t = Tags{}
		} else {
			*t = strings.Split(str, ",")
		}
	case string:
		str := strings.Trim(v, "{}")
		if str == "" {
			*t = Tags{}
		} else {
			*t = strings.Split(str, ",")
		}
	default:
		*t = Tags{}
	}
	return nil
}

// JSONB represents a JSONB database column
type JSONB map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return j, nil
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	// The pgx driver will handle JSONB parsing
	*j = value.(map[string]interface{})
	return nil
}

// Severity represents alert severity levels
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// AlertStatus represents alert status
type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
	AlertStatusIgnored      AlertStatus = "ignored"
)

// IntervalType represents aggregation interval types
type IntervalType string

const (
	Interval1Hour  IntervalType = "1h"
	Interval24Hour IntervalType = "24h"
	Interval7Day   IntervalType = "7d"
	Interval30Day  IntervalType = "30d"
)

// ValidatorFilter contains filter criteria for querying validators
type ValidatorFilter struct {
	ValidatorIndices []int64
	Pubkeys          []string
	Tags             []string
	Monitored        *bool
	Slashed          *bool
	Limit            int
	Offset           int
}

// SnapshotFilter contains filter criteria for querying snapshots
type SnapshotFilter struct {
	ValidatorIndex int64
	StartTime      *time.Time
	EndTime        *time.Time
	Limit          int
	Offset         int
}

// AlertFilter contains filter criteria for querying alerts
type AlertFilter struct {
	ValidatorIndex *int64
	AlertType      *string
	Severity       *Severity
	Status         *AlertStatus
	StartTime      *time.Time
	EndTime        *time.Time
	Limit          int
	Offset         int
}