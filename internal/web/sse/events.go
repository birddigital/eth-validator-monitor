package sse

import (
	"encoding/json"
	"fmt"
)

// EventType represents different SSE event types
type EventType string

const (
	EventTypeMetricsUpdate EventType = "metrics-update"
	EventTypeNewAlert      EventType = "new-alert"
	EventTypeHealthStatus  EventType = "health-status"
	EventTypeHeartbeat     EventType = "heartbeat"
)

// Event represents an SSE event with typed data
type Event struct {
	Type EventType   `json:"type"`
	Data interface{} `json:"data"`
	ID   string      `json:"id,omitempty"` // Optional event ID for resume
}

// MetricsUpdateData represents validator metrics update payload
type MetricsUpdateData struct {
	ValidatorIndex uint64  `json:"validator_index"`
	Balance        uint64  `json:"balance"`
	Effectiveness  float64 `json:"effectiveness"`
	Status         string  `json:"status"`
	LastUpdated    int64   `json:"last_updated"` // Unix timestamp
}

// NewAlertData represents alert notification payload
type NewAlertData struct {
	AlertID     string `json:"alert_id"`
	Severity    string `json:"severity"` // critical, warning, info
	Message     string `json:"message"`
	ValidatorID string `json:"validator_id,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

// HealthStatusData represents system health status
type HealthStatusData struct {
	BeaconNodeStatus string `json:"beacon_node_status"` // connected, disconnected
	DatabaseStatus   string `json:"database_status"`    // healthy, degraded
	LastSync         int64  `json:"last_sync"`
	ActiveValidators int    `json:"active_validators"`
}

// Format formats the event for SSE transmission
func (e *Event) Format() (string, error) {
	data, err := json.Marshal(e.Data)
	if err != nil {
		return "", fmt.Errorf("marshal event data: %w", err)
	}

	var output string
	if e.ID != "" {
		output += fmt.Sprintf("id: %s\n", e.ID)
	}
	output += fmt.Sprintf("event: %s\n", e.Type)
	output += fmt.Sprintf("data: %s\n\n", data)

	return output, nil
}
