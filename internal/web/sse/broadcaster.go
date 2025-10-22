package sse

import (
	"context"
	"log"
	"sync"
	"time"
)

// Client represents a connected SSE client
type Client struct {
	ID       string
	Messages chan Event
	ctx      context.Context
	cancel   context.CancelFunc
}

// Broadcaster manages SSE client connections and broadcasts events
type Broadcaster struct {
	clients map[string]*Client
	mu      sync.RWMutex

	// Channels for client management
	register   chan *Client
	unregister chan string
	broadcast  chan Event

	// Heartbeat configuration
	heartbeatInterval time.Duration

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewBroadcaster creates a new SSE broadcaster
func NewBroadcaster(ctx context.Context) *Broadcaster {
	ctx, cancel := context.WithCancel(ctx)

	b := &Broadcaster{
		clients:           make(map[string]*Client),
		register:          make(chan *Client, 10),
		unregister:        make(chan string, 10),
		broadcast:         make(chan Event, 100), // Buffered for burst traffic
		heartbeatInterval: 30 * time.Second,
		ctx:               ctx,
		cancel:            cancel,
	}

	go b.run()
	go b.heartbeat()

	return b
}

// run is the main event loop for the broadcaster
func (b *Broadcaster) run() {
	for {
		select {
		case <-b.ctx.Done():
			log.Println("SSE broadcaster shutting down...")
			b.closeAllClients()
			return

		case client := <-b.register:
			b.mu.Lock()
			b.clients[client.ID] = client
			b.mu.Unlock()
			log.Printf("SSE client connected: %s (total: %d)", client.ID, len(b.clients))

		case clientID := <-b.unregister:
			b.removeClient(clientID)

		case event := <-b.broadcast:
			b.broadcastEvent(event)
		}
	}
}

// heartbeat sends periodic heartbeat events to keep connections alive
func (b *Broadcaster) heartbeat() {
	ticker := time.NewTicker(b.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.Broadcast(Event{
				Type: EventTypeHeartbeat,
				Data: map[string]interface{}{
					"timestamp": time.Now().Unix(),
				},
			})
		}
	}
}

// broadcastEvent sends an event to all connected clients
func (b *Broadcaster) broadcastEvent(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for clientID, client := range b.clients {
		// Check if client context is cancelled
		select {
		case <-client.ctx.Done():
			// Client context cancelled, remove it
			go b.Unregister(clientID)
			continue
		default:
		}

		// Try to send event to client
		select {
		case client.Messages <- event:
			// Event sent successfully
		default:
			// Client's message buffer is full, disconnect slow client
			log.Printf("SSE client %s buffer full, disconnecting", clientID)
			go b.Unregister(clientID)
		}
	}
}

// Register adds a new client connection
func (b *Broadcaster) Register(clientID string, ctx context.Context) *Client {
	clientCtx, cancel := context.WithCancel(ctx)

	client := &Client{
		ID:       clientID,
		Messages: make(chan Event, 10), // Buffer 10 messages per client
		ctx:      clientCtx,
		cancel:   cancel,
	}

	b.register <- client
	return client
}

// Unregister removes a client connection
func (b *Broadcaster) Unregister(clientID string) {
	b.unregister <- clientID
}

// removeClient handles client removal (called from run loop)
func (b *Broadcaster) removeClient(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if client, exists := b.clients[clientID]; exists {
		client.cancel()
		close(client.Messages)
		delete(b.clients, clientID)
		log.Printf("SSE client disconnected: %s (total: %d)", clientID, len(b.clients))
	}
}

// Broadcast sends an event to all connected clients
func (b *Broadcaster) Broadcast(event Event) {
	select {
	case b.broadcast <- event:
	case <-b.ctx.Done():
		log.Println("Cannot broadcast, broadcaster is shut down")
	default:
		log.Println("Broadcast channel full, dropping event")
	}
}

// ClientCount returns the current number of connected clients
func (b *Broadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Shutdown gracefully shuts down the broadcaster
func (b *Broadcaster) Shutdown() {
	b.cancel()
}

// closeAllClients closes all client connections
func (b *Broadcaster) closeAllClients() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for clientID, client := range b.clients {
		client.cancel()
		close(client.Messages)
		log.Printf("SSE client closed: %s", clientID)
	}
	b.clients = make(map[string]*Client)
}
