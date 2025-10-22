package sse

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBroadcaster_RegisterUnregister(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := NewBroadcaster(ctx)
	defer b.Shutdown()

	clientCtx, clientCancel := context.WithCancel(ctx)
	defer clientCancel()

	// Register client
	client := b.Register("test-client-1", clientCtx)
	require.NotNil(t, client)

	// Wait for registration
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, b.ClientCount())

	// Unregister client
	b.Unregister("test-client-1")

	// Wait for unregistration
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, b.ClientCount())
}

func TestBroadcaster_Broadcast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := NewBroadcaster(ctx)
	defer b.Shutdown()

	clientCtx, clientCancel := context.WithCancel(ctx)
	defer clientCancel()

	// Register client
	client := b.Register("test-client-1", clientCtx)

	// Broadcast event
	testEvent := Event{
		Type: EventTypeMetricsUpdate,
		Data: MetricsUpdateData{
			ValidatorIndex: 12345,
			Balance:        32000000000,
			Effectiveness:  0.98,
		},
	}

	b.Broadcast(testEvent)

	// Receive event
	select {
	case receivedEvent := <-client.Messages:
		assert.Equal(t, EventTypeMetricsUpdate, receivedEvent.Type)
		data, ok := receivedEvent.Data.(MetricsUpdateData)
		require.True(t, ok)
		assert.Equal(t, uint64(12345), data.ValidatorIndex)
		assert.Equal(t, 0.98, data.Effectiveness)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestBroadcaster_MultipleClients(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := NewBroadcaster(ctx)
	defer b.Shutdown()

	// Register 3 clients
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clientCtx, _ := context.WithCancel(ctx)
		clients[i] = b.Register(fmt.Sprintf("client-%d", i), clientCtx)
	}

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 3, b.ClientCount())

	// Broadcast event
	testEvent := Event{
		Type: EventTypeNewAlert,
		Data: NewAlertData{
			AlertID:  "alert-123",
			Severity: "warning",
			Message:  "Test alert",
		},
	}

	b.Broadcast(testEvent)

	// All clients should receive the event
	for i, client := range clients {
		select {
		case receivedEvent := <-client.Messages:
			assert.Equal(t, EventTypeNewAlert, receivedEvent.Type)
			data, ok := receivedEvent.Data.(NewAlertData)
			require.True(t, ok)
			assert.Equal(t, "alert-123", data.AlertID)
		case <-time.After(1 * time.Second):
			t.Fatalf("Client %d did not receive event", i)
		}
	}
}

func TestBroadcaster_ClientDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := NewBroadcaster(ctx)
	defer b.Shutdown()

	clientCtx, clientCancel := context.WithCancel(ctx)

	// Register client
	client := b.Register("test-client-1", clientCtx)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, b.ClientCount())

	// Cancel client context (simulates disconnect)
	clientCancel()

	// Broadcast event - should trigger cleanup
	b.Broadcast(Event{
		Type: EventTypeHeartbeat,
		Data: map[string]interface{}{"timestamp": time.Now().Unix()},
	})

	// Wait for cleanup
	time.Sleep(50 * time.Millisecond)

	// Client should be removed
	_, ok := <-client.Messages
	assert.False(t, ok, "Client channel should be closed")
}

func TestBroadcaster_Heartbeat(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create broadcaster with shorter heartbeat for testing
	b := NewBroadcaster(ctx)
	b.heartbeatInterval = 100 * time.Millisecond
	defer b.Shutdown()

	clientCtx, clientCancel := context.WithCancel(ctx)
	defer clientCancel()

	// Register client
	client := b.Register("test-client-1", clientCtx)

	// Wait for heartbeat
	timeout := time.After(200 * time.Millisecond)
	heartbeatReceived := false

	for !heartbeatReceived {
		select {
		case event := <-client.Messages:
			if event.Type == EventTypeHeartbeat {
				heartbeatReceived = true
			}
		case <-timeout:
			t.Fatal("Did not receive heartbeat within expected time")
		}
	}

	assert.True(t, heartbeatReceived, "Should receive heartbeat event")
}

func TestEvent_Format(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected string
	}{
		{
			name: "event with ID",
			event: Event{
				Type: EventTypeMetricsUpdate,
				ID:   "123",
				Data: map[string]interface{}{"test": "data"},
			},
			expected: "id: 123\nevent: metrics-update\ndata: {\"test\":\"data\"}\n\n",
		},
		{
			name: "event without ID",
			event: Event{
				Type: EventTypeHeartbeat,
				Data: map[string]interface{}{"timestamp": float64(1234567890)},
			},
			expected: "event: heartbeat\ndata: {\"timestamp\":1234567890}\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted, err := tt.event.Format()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, formatted)
		})
	}
}

func TestBroadcaster_SlowClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := NewBroadcaster(ctx)
	defer b.Shutdown()

	clientCtx, clientCancel := context.WithCancel(ctx)
	defer clientCancel()

	// Register client
	client := b.Register("slow-client", clientCtx)
	time.Sleep(10 * time.Millisecond)

	// Fill client buffer (capacity is 10)
	for i := 0; i < 15; i++ {
		b.Broadcast(Event{
			Type: EventTypeMetricsUpdate,
			Data: MetricsUpdateData{ValidatorIndex: uint64(i)},
		})
	}

	// Wait for slow client to be disconnected
	time.Sleep(100 * time.Millisecond)

	// Drain any remaining messages and check if channel is closed
	timeout := time.After(1 * time.Second)
	channelClosed := false
	for !channelClosed {
		select {
		case _, ok := <-client.Messages:
			if !ok {
				channelClosed = true
			}
		case <-timeout:
			t.Fatal("Timeout waiting for client channel to close")
		}
	}

	assert.True(t, channelClosed, "Slow client channel should be closed")
}

func TestBroadcaster_Shutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := NewBroadcaster(ctx)

	// Register multiple clients
	for i := 0; i < 3; i++ {
		clientCtx, _ := context.WithCancel(ctx)
		b.Register(fmt.Sprintf("client-%d", i), clientCtx)
	}

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 3, b.ClientCount())

	// Shutdown broadcaster
	b.Shutdown()

	// Wait for shutdown
	time.Sleep(50 * time.Millisecond)

	// All clients should be removed
	assert.Equal(t, 0, b.ClientCount())
}
