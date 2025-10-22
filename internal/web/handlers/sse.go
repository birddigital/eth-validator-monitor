package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/birddigital/eth-validator-monitor/internal/web/sse"
)

// SSEHandler handles Server-Sent Events connections
type SSEHandler struct {
	broadcaster *sse.Broadcaster
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(ctx context.Context) *SSEHandler {
	return &SSEHandler{
		broadcaster: sse.NewBroadcaster(ctx),
	}
}

// ServeHTTP implements http.Handler for SSE endpoint
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Verify SSE support
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust for production

	// Generate unique client ID
	clientID := uuid.New().String()

	// Register client
	client := h.broadcaster.Register(clientID, r.Context())
	defer h.broadcaster.Unregister(clientID)

	// Send initial connection event
	initialEvent := sse.Event{
		Type: sse.EventTypeHealthStatus,
		Data: map[string]interface{}{
			"message":   "Connected to validator monitor",
			"client_id": clientID,
			"timestamp": time.Now().Unix(),
		},
	}

	if err := h.sendEvent(w, flusher, initialEvent); err != nil {
		log.Printf("Failed to send initial event to %s: %v", clientID, err)
		return
	}

	// Stream events to client
	for {
		select {
		case <-r.Context().Done():
			log.Printf("Client %s disconnected (context done)", clientID)
			return

		case event, ok := <-client.Messages:
			if !ok {
				log.Printf("Client %s message channel closed", clientID)
				return
			}

			if err := h.sendEvent(w, flusher, event); err != nil {
				log.Printf("Failed to send event to %s: %v", clientID, err)
				return
			}
		}
	}
}

// sendEvent sends a single event to the client
func (h *SSEHandler) sendEvent(w http.ResponseWriter, flusher http.Flusher, event sse.Event) error {
	formatted, err := event.Format()
	if err != nil {
		return fmt.Errorf("format event: %w", err)
	}

	if _, err := fmt.Fprint(w, formatted); err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	flusher.Flush()
	return nil
}

// Broadcaster returns the underlying broadcaster for publishing events
func (h *SSEHandler) Broadcaster() *sse.Broadcaster {
	return h.broadcaster
}

// BroadcastMetricsUpdate publishes a metrics update event
func (h *SSEHandler) BroadcastMetricsUpdate(data sse.MetricsUpdateData) {
	h.broadcaster.Broadcast(sse.Event{
		Type: sse.EventTypeMetricsUpdate,
		Data: data,
		ID:   fmt.Sprintf("metrics-%d-%d", data.ValidatorIndex, data.LastUpdated),
	})
}

// BroadcastAlert publishes a new alert event
func (h *SSEHandler) BroadcastAlert(data sse.NewAlertData) {
	h.broadcaster.Broadcast(sse.Event{
		Type: sse.EventTypeNewAlert,
		Data: data,
		ID:   data.AlertID,
	})
}

// BroadcastHealthStatus publishes a health status event
func (h *SSEHandler) BroadcastHealthStatus(data sse.HealthStatusData) {
	h.broadcaster.Broadcast(sse.Event{
		Type: sse.EventTypeHealthStatus,
		Data: data,
		ID:   fmt.Sprintf("health-%d", data.LastSync),
	})
}
