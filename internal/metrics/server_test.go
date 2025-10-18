package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewMetricsServer(t *testing.T) {
	apiMetrics := NewAPIMetrics()
	server := NewMetricsServer(9090, apiMetrics)

	if server == nil {
		t.Fatal("Expected server to be initialized")
	}
	if server.port != 9090 {
		t.Errorf("Expected port 9090, got %d", server.port)
	}
	if server.apiMetrics != apiMetrics {
		t.Error("API metrics not set correctly")
	}
	if server.server == nil {
		t.Error("HTTP server not initialized")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	// Use a unique port for this test
	port := 19090
	// Pass nil since API metrics are registered globally and already exist
	server := NewMetricsServer(port, nil)

	// Start server in background
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Ensure cleanup
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Test /metrics endpoint
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	if err != nil {
		t.Fatalf("Failed to get /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	bodyStr := string(body)

	// Verify Prometheus format - should have HELP and TYPE lines
	if !strings.Contains(bodyStr, "# HELP") {
		t.Error("Response missing Prometheus HELP lines")
	}
	if !strings.Contains(bodyStr, "# TYPE") {
		t.Error("Response missing Prometheus TYPE lines")
	}

	// Verify Prometheus Go metrics are present (these are always registered)
	expectedMetrics := []string{
		"go_goroutines",
		"go_memstats_alloc_bytes",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(bodyStr, metric) {
			t.Errorf("Expected metric '%s' not found in /metrics output", metric)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Use a unique port for this test
	port := 19091
	server := NewMetricsServer(port, nil)

	// Start server in background
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Ensure cleanup
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Test /health endpoint
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		t.Fatalf("Failed to get /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if string(body) != "OK" {
		t.Errorf("Expected 'OK', got '%s'", string(body))
	}
}

func TestServerShutdown(t *testing.T) {
	port := 19092
	server := NewMetricsServer(port, nil)

	// Start server
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	_, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		t.Fatalf("Server not running: %v", err)
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}

	// Wait for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	// Verify server is stopped
	_, err = http.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err == nil {
		t.Error("Expected server to be stopped, but connection succeeded")
	}
}

func TestValidatorMetricsExposition(t *testing.T) {
	// Create validator metrics
	validatorMetrics := NewValidatorMetrics()

	// Record some test data
	validatorMetrics.RecordEffectivenessScore(123, "0xabc", 95.5)
	validatorMetrics.RecordSnapshotLag(123, 12.5)
	validatorMetrics.RecordMissedAttestation(123, "0xabc")
	validatorMetrics.RecordBalance(123, "0xabc", 32000000000)
	validatorMetrics.RecordProposalSuccessRate(123, "0xabc", 0.875)
	validatorMetrics.RecordValidatorStatus(123, "0xabc", "active", 2)
	validatorMetrics.RecordAttestationRate(123, "0xabc", 0.99)
	validatorMetrics.RecordReward(123, "0xabc", 1000000)
	validatorMetrics.RecordPenalty(123, "0xabc", 50000)
	validatorMetrics.RecordBlockProposal(123, "0xabc")
	validatorMetrics.RecordSuccessfulProposal(123, "0xabc")

	// Start a server to expose the metrics
	port := 19093
	server := NewMetricsServer(port, nil)

	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Fetch metrics
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	if err != nil {
		t.Fatalf("Failed to get /metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	bodyStr := string(body)

	// Verify validator metrics are present in exposition format
	expectedMetrics := []string{
		"validator_effectiveness_score",
		"validator_snapshot_lag_seconds",
		"validator_missed_attestations_total",
		"validator_balance_wei",
		"validator_proposal_success_rate",
		"validator_status",
		"validator_attestation_participation_rate",
		"validator_rewards_wei_total",
		"validator_penalties_wei_total",
		"validator_block_proposals_total",
		"validator_successful_proposals_total",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(bodyStr, metric) {
			t.Errorf("Expected validator metric '%s' not found in /metrics output", metric)
		}
	}

	// Verify labels are present
	expectedLabels := []string{
		`validator_index="123"`,
		`pubkey="0xabc"`,
	}

	for _, label := range expectedLabels {
		if !strings.Contains(bodyStr, label) {
			t.Errorf("Expected label '%s' not found in /metrics output", label)
		}
	}
}
