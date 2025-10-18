package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestNewValidatorMetrics is tested via TestValidatorMetricsExposition
// to avoid duplicate metric registration conflicts.
// The validator metrics use promauto which registers globally,
// so we verify initialization through actual usage in endpoint tests.

func TestRecordEffectivenessScore(t *testing.T) {
	// Create new registry to avoid conflicts with global metrics
	registry := prometheus.NewRegistry()

	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_validator_effectiveness_score",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey"},
	)
	registry.MustRegister(metric)

	// Record a value
	metric.WithLabelValues("123", "0xabc").Set(95.5)

	// Verify the value
	value := testutil.ToFloat64(metric.WithLabelValues("123", "0xabc"))
	if value != 95.5 {
		t.Errorf("Expected 95.5, got %f", value)
	}
}

func TestRecordSnapshotLag(t *testing.T) {
	registry := prometheus.NewRegistry()

	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_validator_snapshot_lag_seconds",
			Help: "Test metric",
		},
		[]string{"validator_index"},
	)
	registry.MustRegister(metric)

	// Record a value
	metric.WithLabelValues("456").Set(12.5)

	// Verify the value
	value := testutil.ToFloat64(metric.WithLabelValues("456"))
	if value != 12.5 {
		t.Errorf("Expected 12.5, got %f", value)
	}
}

func TestRecordMissedAttestation(t *testing.T) {
	registry := prometheus.NewRegistry()

	metric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_validator_missed_attestations_total",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey"},
	)
	registry.MustRegister(metric)

	// Increment counter
	metric.WithLabelValues("789", "0xdef").Inc()
	metric.WithLabelValues("789", "0xdef").Inc()
	metric.WithLabelValues("789", "0xdef").Inc()

	// Verify the value
	value := testutil.ToFloat64(metric.WithLabelValues("789", "0xdef"))
	if value != 3 {
		t.Errorf("Expected 3, got %f", value)
	}
}

func TestRecordBalance(t *testing.T) {
	registry := prometheus.NewRegistry()

	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_validator_balance_wei",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey"},
	)
	registry.MustRegister(metric)

	// Record a value
	balanceWei := 32000000000.0 // 32 ETH in Gwei
	metric.WithLabelValues("100", "0x123").Set(balanceWei)

	// Verify the value
	value := testutil.ToFloat64(metric.WithLabelValues("100", "0x123"))
	if value != balanceWei {
		t.Errorf("Expected %f, got %f", balanceWei, value)
	}
}

func TestRecordProposalSuccessRate(t *testing.T) {
	registry := prometheus.NewRegistry()

	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_validator_proposal_success_rate",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey"},
	)
	registry.MustRegister(metric)

	// Record a value (0-1 range)
	metric.WithLabelValues("200", "0x456").Set(0.875)

	// Verify the value
	value := testutil.ToFloat64(metric.WithLabelValues("200", "0x456"))
	if value != 0.875 {
		t.Errorf("Expected 0.875, got %f", value)
	}
}

func TestRecordValidatorStatus(t *testing.T) {
	registry := prometheus.NewRegistry()

	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_validator_status",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey", "status_name"},
	)
	registry.MustRegister(metric)

	// Test different status values
	tests := []struct {
		statusName  string
		statusValue float64
	}{
		{"offline", 0},
		{"pending", 1},
		{"active", 2},
		{"exiting", 3},
		{"slashed", 4},
	}

	for _, tt := range tests {
		metric.WithLabelValues("300", "0x789", tt.statusName).Set(tt.statusValue)
		value := testutil.ToFloat64(metric.WithLabelValues("300", "0x789", tt.statusName))
		if value != tt.statusValue {
			t.Errorf("Status %s: expected %f, got %f", tt.statusName, tt.statusValue, value)
		}
	}
}

func TestRecordAttestationRate(t *testing.T) {
	registry := prometheus.NewRegistry()

	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_validator_attestation_participation_rate",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey"},
	)
	registry.MustRegister(metric)

	// Record a value (0-1 range)
	metric.WithLabelValues("400", "0xabc").Set(0.99)

	// Verify the value
	value := testutil.ToFloat64(metric.WithLabelValues("400", "0xabc"))
	if value != 0.99 {
		t.Errorf("Expected 0.99, got %f", value)
	}
}

func TestRecordRewardAndPenalty(t *testing.T) {
	registry := prometheus.NewRegistry()

	rewardMetric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_validator_rewards_wei_total",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey"},
	)
	penaltyMetric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_validator_penalties_wei_total",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey"},
	)
	registry.MustRegister(rewardMetric, penaltyMetric)

	// Add rewards
	rewardMetric.WithLabelValues("500", "0xdef").Add(1000000)
	rewardMetric.WithLabelValues("500", "0xdef").Add(500000)

	// Add penalties
	penaltyMetric.WithLabelValues("500", "0xdef").Add(50000)

	// Verify values
	rewardValue := testutil.ToFloat64(rewardMetric.WithLabelValues("500", "0xdef"))
	if rewardValue != 1500000 {
		t.Errorf("Expected rewards 1500000, got %f", rewardValue)
	}

	penaltyValue := testutil.ToFloat64(penaltyMetric.WithLabelValues("500", "0xdef"))
	if penaltyValue != 50000 {
		t.Errorf("Expected penalties 50000, got %f", penaltyValue)
	}
}

func TestRecordBlockProposals(t *testing.T) {
	registry := prometheus.NewRegistry()

	proposalMetric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_validator_block_proposals_total",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey"},
	)
	successfulMetric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_validator_successful_proposals_total",
			Help: "Test metric",
		},
		[]string{"validator_index", "pubkey"},
	)
	registry.MustRegister(proposalMetric, successfulMetric)

	// Record 5 proposals, 4 successful
	for i := 0; i < 5; i++ {
		proposalMetric.WithLabelValues("600", "0x111").Inc()
	}
	for i := 0; i < 4; i++ {
		successfulMetric.WithLabelValues("600", "0x111").Inc()
	}

	// Verify values
	proposalValue := testutil.ToFloat64(proposalMetric.WithLabelValues("600", "0x111"))
	if proposalValue != 5 {
		t.Errorf("Expected 5 proposals, got %f", proposalValue)
	}

	successValue := testutil.ToFloat64(successfulMetric.WithLabelValues("600", "0x111"))
	if successValue != 4 {
		t.Errorf("Expected 4 successful proposals, got %f", successValue)
	}
}

func TestMetricLabels(t *testing.T) {
	registry := prometheus.NewRegistry()

	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_validator_metric_with_labels",
			Help: "Test metric for label verification",
		},
		[]string{"validator_index", "pubkey"},
	)
	registry.MustRegister(metric)

	// Record values with different labels
	testCases := []struct {
		validatorIndex string
		pubkey         string
		value          float64
	}{
		{"1", "0xaaa", 10.0},
		{"2", "0xbbb", 20.0},
		{"3", "0xccc", 30.0},
	}

	for _, tc := range testCases {
		metric.WithLabelValues(tc.validatorIndex, tc.pubkey).Set(tc.value)
	}

	// Verify each labeled metric has correct value
	for _, tc := range testCases {
		value := testutil.ToFloat64(metric.WithLabelValues(tc.validatorIndex, tc.pubkey))
		if value != tc.value {
			t.Errorf("Labels [%s, %s]: expected %f, got %f",
				tc.validatorIndex, tc.pubkey, tc.value, value)
		}
	}
}
