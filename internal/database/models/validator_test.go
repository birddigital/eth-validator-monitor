package models

import (
	"database/sql/driver"
	"testing"
	"time"
)

// TestAlertModelValidation tests the Alert model structure and field types
func TestAlertModelValidation(t *testing.T) {
	tests := []struct {
		name  string
		alert Alert
		valid bool
	}{
		{
			name: "valid alert with all fields",
			alert: Alert{
				ID:             1,
				ValidatorIndex: ptrInt64(123456),
				AlertType:      "missed_attestation",
				Severity:       SeverityCritical,
				Title:          "Missed Attestation",
				Message:        "Validator 123456 missed an attestation",
				Source:         "validator_collector",
				Details:        JSONB{"epoch": 1000, "slot": 32000},
				Status:         AlertStatusNew,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			valid: true,
		},
		{
			name: "valid alert with minimal fields",
			alert: Alert{
				ID:        2,
				AlertType: "network_alert",
				Severity:  SeverityWarning,
				Title:     "Network Issue",
				Message:   "Network latency detected",
				Source:    "system",
				Status:    AlertStatusNew,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			valid: true,
		},
		{
			name: "valid alert with legacy status",
			alert: Alert{
				ID:        3,
				AlertType: "slashing_event",
				Severity:  SeverityCritical,
				Title:     "Slashing Detected",
				Message:   "Validator slashed",
				Source:    "beacon_chain_monitor",
				Status:    AlertStatusActive,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify required fields are set
			if tt.alert.AlertType == "" {
				t.Error("AlertType should not be empty")
			}
			if tt.alert.Severity == "" {
				t.Error("Severity should not be empty")
			}
			if tt.alert.Title == "" {
				t.Error("Title should not be empty")
			}
			if tt.alert.Message == "" {
				t.Error("Message should not be empty")
			}
			if tt.alert.Source == "" {
				t.Error("Source should not be empty")
			}
			if tt.alert.Status == "" {
				t.Error("Status should not be empty")
			}
		})
	}
}

// TestSeverityEnum tests Severity enum values
func TestSeverityEnum(t *testing.T) {
	tests := []struct {
		severity Severity
		valid    bool
	}{
		{SeverityInfo, true},
		{SeverityWarning, true},
		{SeverityError, true},
		{SeverityCritical, true},
		{Severity("invalid"), false},
	}

	validSeverities := map[Severity]bool{
		SeverityInfo:     true,
		SeverityWarning:  true,
		SeverityError:    true,
		SeverityCritical: true,
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			_, exists := validSeverities[tt.severity]
			if exists != tt.valid {
				t.Errorf("Severity %s validity = %v, want %v", tt.severity, exists, tt.valid)
			}
		})
	}
}

// TestAlertStatusEnum tests AlertStatus enum values
func TestAlertStatusEnum(t *testing.T) {
	tests := []struct {
		status AlertStatus
		valid  bool
	}{
		// Legacy statuses
		{AlertStatusActive, true},
		{AlertStatusAcknowledged, true},
		{AlertStatusResolved, true},
		{AlertStatusIgnored, true},
		// New statuses for alerts management page
		{AlertStatusNew, true},
		{AlertStatusRead, true},
		{AlertStatusDismissed, true},
		// Invalid
		{AlertStatus("invalid"), false},
	}

	validStatuses := map[AlertStatus]bool{
		AlertStatusActive:       true,
		AlertStatusAcknowledged: true,
		AlertStatusResolved:     true,
		AlertStatusIgnored:      true,
		AlertStatusNew:          true,
		AlertStatusRead:         true,
		AlertStatusDismissed:    true,
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			_, exists := validStatuses[tt.status]
			if exists != tt.valid {
				t.Errorf("AlertStatus %s validity = %v, want %v", tt.status, exists, tt.valid)
			}
		})
	}
}

// TestJSONBType tests JSONB custom type
func TestJSONBType(t *testing.T) {
	tests := []struct {
		name  string
		jsonb JSONB
	}{
		{
			name:  "nil JSONB",
			jsonb: nil,
		},
		{
			name:  "empty JSONB",
			jsonb: JSONB{},
		},
		{
			name: "JSONB with data",
			jsonb: JSONB{
				"epoch":           1000,
				"slot":            32000,
				"validator_index": 123456,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Value() method
			val, err := tt.jsonb.Value()
			if err != nil {
				t.Errorf("JSONB.Value() error = %v", err)
			}

			if tt.jsonb == nil && val != nil {
				t.Error("Expected nil value for nil JSONB")
			}
		})
	}
}

// TestTagsType tests Tags custom type (PostgreSQL array)
func TestTagsType(t *testing.T) {
	tests := []struct {
		name     string
		tags     Tags
		expected driver.Value
	}{
		{
			name:     "empty tags",
			tags:     Tags{},
			expected: "{}",
		},
		{
			name:     "single tag",
			tags:     Tags{"production"},
			expected: "{production}",
		},
		{
			name:     "multiple tags",
			tags:     Tags{"production", "critical", "monitored"},
			expected: "{production,critical,monitored}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Value() method
			val, err := tt.tags.Value()
			if err != nil {
				t.Errorf("Tags.Value() error = %v", err)
			}
			if val != tt.expected {
				t.Errorf("Tags.Value() = %v, want %v", val, tt.expected)
			}

			// Test Scan() method
			var scanned Tags
			err = scanned.Scan(val)
			if err != nil {
				t.Errorf("Tags.Scan() error = %v", err)
			}

			// Verify scanned tags match original
			if len(scanned) != len(tt.tags) {
				t.Errorf("Tags.Scan() length = %d, want %d", len(scanned), len(tt.tags))
			}
		})
	}
}

// Helper function for creating pointer to int64
func ptrInt64(i int64) *int64 {
	return &i
}
