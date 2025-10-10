package types

import (
	"context"
	"time"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityWarning  AlertSeverity = "warning"
	SeverityInfo     AlertSeverity = "info"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeOffline              AlertType = "offline"
	AlertTypeSlashed              AlertType = "slashed"
	AlertTypePerformanceDegr      AlertType = "performance_degraded"
	AlertTypeMissedAttestation    AlertType = "missed_attestation"
	AlertTypeMissedProposal       AlertType = "missed_proposal"
	AlertTypeBalanceDecrease      AlertType = "balance_decreased"
	AlertTypeLowPeerCount         AlertType = "low_peer_count"
	AlertTypeValidatorActivated   AlertType = "validator_activated"
	AlertTypeRewardsMilestone     AlertType = "rewards_milestone"
)

// Alert represents a system alert
type Alert struct {
	ID              int64         `json:"id"`
	ValidatorIndex  int           `json:"validator_index"`
	Severity        AlertSeverity `json:"severity"`
	Type            AlertType     `json:"type"`
	Message         string        `json:"message"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Acknowledged    bool          `json:"acknowledged"`
	CreatedAt       time.Time     `json:"created_at"`
}

// AlertFilter is used for querying alerts
type AlertFilter struct {
	ValidatorIndex *int           `json:"validator_index,omitempty"`
	Severity       *AlertSeverity `json:"severity,omitempty"`
	Type           *AlertType     `json:"type,omitempty"`
	Acknowledged   *bool          `json:"acknowledged,omitempty"`
	From           *time.Time     `json:"from,omitempty"`
	To             *time.Time     `json:"to,omitempty"`
	Limit          int            `json:"limit,omitempty"`
	Offset         int            `json:"offset,omitempty"`
}

// Alerter defines the interface for alert notification systems
type Alerter interface {
	// SendAlert sends an alert through configured channels
	SendAlert(ctx context.Context, alert *Alert) error

	// GetAlerts retrieves alerts based on filter criteria
	GetAlerts(ctx context.Context, filter AlertFilter) ([]*Alert, error)

	// AcknowledgeAlert marks an alert as acknowledged
	AcknowledgeAlert(ctx context.Context, alertID int64) error
}

// AlertChannel defines the interface for individual alert channels
type AlertChannel interface {
	// Name returns the channel name
	Name() string

	// Send sends an alert through this channel
	Send(ctx context.Context, alert *Alert) error

	// Enabled returns whether this channel is enabled
	Enabled() bool
}
