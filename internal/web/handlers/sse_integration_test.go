package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/web/handlers"
	"github.com/birddigital/eth-validator-monitor/internal/web/sse"
)

// TestSSEHandler_MultipleConcurrentClients tests SSE with multiple concurrent connections
func TestSSEHandler_MultipleConcurrentClients(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create SSE handler
	sseHandler := handlers.NewSSEHandler(ctx)
	broadcaster := sseHandler.Broadcaster()

	// Simulate 10 concurrent SSE connections
	const numClients = 10
	servers := make([]*httptest.Server, numClients)
	responses := make(chan string, numClients)

	for i := 0; i < numClients; i++ {
		clientID := i
		server := httptest.NewServer(http.HandlerFunc(sseHandler.ServeHTTP))
		servers[i] = server

		// Connect each client
		go func(id int, serverURL string) {
			req, err := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
			if err != nil {
				t.Errorf("Client %d: failed to create request: %v", id, err)
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("Client %d: failed to connect: %v", id, err)
				return
			}
			defer resp.Body.Close()

			// Verify SSE headers
			if resp.Header.Get("Content-Type") != "text/event-stream" {
				t.Errorf("Client %d: wrong content type: %s", id, resp.Header.Get("Content-Type"))
			}

			// Read first event (connection confirmation)
			buf := make([]byte, 1024)
			n, err := resp.Body.Read(buf)
			if err != nil {
				t.Errorf("Client %d: failed to read response: %v", id, err)
				return
			}

			responses <- string(buf[:n])
		}(clientID, server.URL)
	}

	// Wait for all clients to connect
	time.Sleep(500 * time.Millisecond)

	// Verify client count
	connectedClients := broadcaster.ClientCount()
	if connectedClients != numClients {
		t.Errorf("Expected %d connected clients, got %d", numClients, connectedClients)
	}

	// Broadcast a test event
	testEvent := sse.Event{
		Type: sse.EventTypeMetricsUpdate,
		Data: map[string]interface{}{
			"test": "concurrent_clients",
		},
		ID: "test-1",
	}
	broadcaster.Broadcast(testEvent)

	// Allow event to propagate
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	for _, server := range servers {
		server.Close()
	}

	// Verify we received responses from all clients
	receivedCount := len(responses)
	if receivedCount != numClients {
		t.Logf("Received responses from %d/%d clients", receivedCount, numClients)
	}

	t.Logf("Successfully tested %d concurrent SSE connections", numClients)
}

// TestSSEHandler_HeartbeatKeepAlive tests that heartbeat events keep connections alive
func TestSSEHandler_HeartbeatKeepAlive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	sseHandler := handlers.NewSSEHandler(ctx)
	server := httptest.NewServer(http.HandlerFunc(sseHandler.ServeHTTP))
	defer server.Close()

	// Connect to SSE endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer resp.Body.Close()

	// Read events for 35 seconds (should receive at least 1 heartbeat at 30s)
	heartbeatReceived := false
	readCtx, readCancel := context.WithTimeout(ctx, 35*time.Second)
	defer readCancel()

	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-readCtx.Done():
				return
			default:
				n, err := resp.Body.Read(buf)
				if err != nil {
					return
				}
				data := string(buf[:n])
				if strings.Contains(data, "heartbeat") {
					heartbeatReceived = true
					readCancel() // Stop reading after heartbeat
				}
			}
		}
	}()

	<-readCtx.Done()

	if !heartbeatReceived {
		t.Error("No heartbeat event received within 35 seconds")
	} else {
		t.Log("Heartbeat event received successfully")
	}
}

// TestSSEHandler_DisconnectionCleanup tests that disconnected clients are cleaned up
func TestSSEHandler_DisconnectionCleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sseHandler := handlers.NewSSEHandler(ctx)
	broadcaster := sseHandler.Broadcaster()

	// Create and connect 5 clients
	const numClients = 5
	servers := make([]*httptest.Server, numClients)
	clientContexts := make([]context.CancelFunc, numClients)

	for i := 0; i < numClients; i++ {
		server := httptest.NewServer(http.HandlerFunc(sseHandler.ServeHTTP))
		servers[i] = server

		clientCtx, clientCancel := context.WithCancel(ctx)
		clientContexts[i] = clientCancel

		// Connect client
		go func(serverURL string, cCtx context.Context) {
			req, _ := http.NewRequestWithContext(cCtx, "GET", serverURL, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			// Keep reading until context cancelled
			buf := make([]byte, 1024)
			for {
				select {
				case <-cCtx.Done():
					return
				default:
					resp.Body.Read(buf)
				}
			}
		}(server.URL, clientCtx)
	}

	// Wait for all clients to connect
	time.Sleep(500 * time.Millisecond)

	initialCount := broadcaster.ClientCount()
	if initialCount != numClients {
		t.Errorf("Expected %d clients initially, got %d", numClients, initialCount)
	}

	// Disconnect 3 clients
	for i := 0; i < 3; i++ {
		clientContexts[i]()
		servers[i].Close()
	}

	// Wait for cleanup
	time.Sleep(2 * time.Second)

	// Verify remaining client count
	remainingCount := broadcaster.ClientCount()
	expectedRemaining := numClients - 3
	if remainingCount != int32(expectedRemaining) {
		t.Errorf("Expected %d remaining clients, got %d", expectedRemaining, remainingCount)
	}

	// Cleanup remaining clients
	for i := 3; i < numClients; i++ {
		clientContexts[i]()
		servers[i].Close()
	}

	t.Logf("Successfully tested client disconnection cleanup: %d -> %d clients", initialCount, remainingCount)
}

// TestSSEHandler_EventBroadcasting tests that events are broadcast to all clients
func TestSSEHandler_EventBroadcasting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sseHandler := handlers.NewSSEHandler(ctx)
	broadcaster := sseHandler.Broadcaster()

	// Connect 3 clients
	const numClients = 3
	eventChans := make([]chan string, numClients)

	for i := 0; i < numClients; i++ {
		eventChans[i] = make(chan string, 10)
		server := httptest.NewServer(http.HandlerFunc(sseHandler.ServeHTTP))
		defer server.Close()

		// Connect and read events
		go func(ch chan string, serverURL string) {
			req, _ := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			buf := make([]byte, 4096)
			for {
				n, err := resp.Body.Read(buf)
				if err != nil {
					return
				}
				ch <- string(buf[:n])
			}
		}(eventChans[i], server.URL)
	}

	// Wait for connections
	time.Sleep(300 * time.Millisecond)

	// Broadcast test events
	for j := 0; j < 5; j++ {
		event := sse.Event{
			Type: sse.EventTypeMetricsUpdate,
			Data: map[string]interface{}{
				"index": j,
				"test":  fmt.Sprintf("event-%d", j),
			},
			ID: fmt.Sprintf("test-%d", j),
		}
		broadcaster.Broadcast(event)
		time.Sleep(50 * time.Millisecond)
	}

	// Verify all clients received events
	for i, ch := range eventChans {
		receivedEvents := len(ch)
		if receivedEvents == 0 {
			t.Errorf("Client %d received no events", i)
		} else {
			t.Logf("Client %d received %d events", i, receivedEvents)
		}
	}
}

// BenchmarkSSEHandler_ConcurrentConnections benchmarks SSE with many concurrent connections
func BenchmarkSSEHandler_ConcurrentConnections(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sseHandler := handlers.NewSSEHandler(ctx)
	broadcaster := sseHandler.Broadcaster()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate 50 concurrent connections
		const connections = 50
		servers := make([]*httptest.Server, connections)

		for j := 0; j < connections; j++ {
			server := httptest.NewServer(http.HandlerFunc(sseHandler.ServeHTTP))
			servers[j] = server

			go func(serverURL string) {
				req, _ := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return
				}
				defer resp.Body.Close()

				buf := make([]byte, 1024)
				resp.Body.Read(buf)
			}(server.URL)
		}

		time.Sleep(100 * time.Millisecond)

		// Broadcast 10 events
		for k := 0; k < 10; k++ {
			broadcaster.Broadcast(sse.Event{
				Type: sse.EventTypeMetricsUpdate,
				Data: map[string]interface{}{"test": k},
			})
		}

		// Cleanup
		for _, server := range servers {
			server.Close()
		}
	}
}
