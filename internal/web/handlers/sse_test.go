package handlers

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/birddigital/eth-validator-monitor/internal/web/sse"
)

func TestSSEHandler_Connection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	handler := NewSSEHandler(ctx)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	// Start handler in goroutine
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	// Wait a bit for initial event
	time.Sleep(100 * time.Millisecond)

	// Cancel context to close connection
	cancel()

	// Wait for handler to finish
	<-done

	// Verify headers
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))

	// Verify initial event was sent
	body := rec.Body.String()
	assert.Contains(t, body, "event: health-status")
	assert.Contains(t, body, "Connected to validator monitor")
}

func TestSSEHandler_BroadcastMetrics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	handler := NewSSEHandler(ctx)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req = req.WithContext(ctx)

	// Use flushable recorder
	rec := &flushableRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		body:             &strings.Builder{},
	}

	// Start handler
	go handler.ServeHTTP(rec, req)

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Broadcast metrics update
	handler.BroadcastMetricsUpdate(sse.MetricsUpdateData{
		ValidatorIndex: 42,
		Balance:        32000000000,
		Effectiveness:  0.99,
		Status:         "active",
		LastUpdated:    time.Now().Unix(),
	})

	// Wait for event propagation
	time.Sleep(100 * time.Millisecond)

	// Verify event received
	body := rec.body.String()
	assert.Contains(t, body, "event: metrics-update")
	assert.Contains(t, body, `"validator_index":42`)
	assert.Contains(t, body, `"effectiveness":0.99`)

	cancel()
}

func TestSSEHandler_BroadcastAlert(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	handler := NewSSEHandler(ctx)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req = req.WithContext(ctx)

	rec := &flushableRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		body:             &strings.Builder{},
	}

	go handler.ServeHTTP(rec, req)
	time.Sleep(100 * time.Millisecond)

	// Broadcast alert
	handler.BroadcastAlert(sse.NewAlertData{
		AlertID:     "alert-123",
		Severity:    "critical",
		Message:     "Validator missed attestation",
		ValidatorID: "42",
		Timestamp:   time.Now().Unix(),
	})

	time.Sleep(100 * time.Millisecond)

	body := rec.body.String()
	assert.Contains(t, body, "event: new-alert")
	assert.Contains(t, body, `"alert_id":"alert-123"`)
	assert.Contains(t, body, `"severity":"critical"`)

	cancel()
}

func TestSSEHandler_BroadcastHealthStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	handler := NewSSEHandler(ctx)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req = req.WithContext(ctx)

	rec := &flushableRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		body:             &strings.Builder{},
	}

	go handler.ServeHTTP(rec, req)
	time.Sleep(100 * time.Millisecond)

	// Broadcast health status
	handler.BroadcastHealthStatus(sse.HealthStatusData{
		BeaconNodeStatus: "connected",
		DatabaseStatus:   "healthy",
		LastSync:         time.Now().Unix(),
		ActiveValidators: 10,
	})

	time.Sleep(100 * time.Millisecond)

	body := rec.body.String()
	assert.Contains(t, body, "event: health-status")
	assert.Contains(t, body, `"beacon_node_status":"connected"`)
	assert.Contains(t, body, `"active_validators":10`)

	cancel()
}

func TestSSEHandler_MultipleClients(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	handler := NewSSEHandler(ctx)

	// Create HTTP test server
	server := httptest.NewServer(handler)
	defer server.Close()

	const numClients = 10
	clientsDone := make(chan struct{}, numClients)

	// Connect 10 clients
	for i := 0; i < numClients; i++ {
		go func(clientNum int) {
			defer func() { clientsDone <- struct{}{} }()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Read events
			scanner := bufio.NewScanner(resp.Body)
			eventCount := 0

			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "event:") {
					eventCount++
				}

				// Read at least 2 events then stop
				if eventCount >= 2 {
					break
				}
			}

			assert.GreaterOrEqual(t, eventCount, 2, "Client %d should receive at least 2 events", clientNum)
		}(i)
	}

	// Wait for clients to connect
	time.Sleep(200 * time.Millisecond)

	// Broadcast event to all clients
	handler.BroadcastMetricsUpdate(sse.MetricsUpdateData{
		ValidatorIndex: 1,
		Balance:        32000000000,
		Effectiveness:  0.98,
		LastUpdated:    time.Now().Unix(),
	})

	// Wait for all clients to finish
	for i := 0; i < numClients; i++ {
		select {
		case <-clientsDone:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for clients to finish")
		}
	}
}

func TestSSEHandler_LoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	handler := NewSSEHandler(ctx)

	// Create HTTP test server
	server := httptest.NewServer(handler)
	defer server.Close()

	const numClients = 100
	clientsDone := make(chan struct{}, numClients)

	// Connect 100 clients
	for i := 0; i < numClients; i++ {
		go func(clientNum int) {
			defer func() { clientsDone <- struct{}{} }()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Errorf("Client %d: failed to create request: %v", clientNum, err)
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("Client %d: failed to connect: %v", clientNum, err)
				return
			}
			defer resp.Body.Close()

			// Read events for a short duration
			scanner := bufio.NewScanner(resp.Body)
			eventCount := 0

			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "event:") {
					eventCount++
				}

				// Read at least 3 events then stop
				if eventCount >= 3 {
					break
				}
			}

			if eventCount < 3 {
				t.Errorf("Client %d: only received %d events", clientNum, eventCount)
			}
		}(i)
	}

	// Broadcast events while clients are connected
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				handler.BroadcastMetricsUpdate(sse.MetricsUpdateData{
					ValidatorIndex: 1,
					Balance:        32000000000,
					Effectiveness:  0.98,
					LastUpdated:    time.Now().Unix(),
				})
			}
		}
	}()

	// Wait for all clients to finish
	for i := 0; i < numClients; i++ {
		select {
		case <-clientsDone:
		case <-time.After(15 * time.Second):
			t.Fatal("Timeout waiting for clients to finish")
		}
	}

	// Verify client count
	assert.LessOrEqual(t, handler.Broadcaster().ClientCount(), numClients)
}

// Helper for testing SSE with flush support
type flushableRecorder struct {
	*httptest.ResponseRecorder
	body *strings.Builder
}

func (f *flushableRecorder) Write(b []byte) (int, error) {
	f.body.Write(b)
	return f.ResponseRecorder.Write(b)
}

func (f *flushableRecorder) Flush() {
	// No-op for testing
}
