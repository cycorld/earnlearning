package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub maintains active WebSocket clients and broadcasts messages.
type Hub struct {
	// clients maps user_id to their active Client connections.
	clients map[int]map[*Client]bool

	// register channel for new clients.
	register chan *Client

	// unregister channel for disconnecting clients.
	unregister chan *Client

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop. Should be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
			h.mu.Unlock()
			log.Printf("ws: user %d connected (total connections: %d)", client.userID, h.countConnections(client.userID))

		case client := <-h.unregister:
			h.mu.Lock()
			if conns, ok := h.clients[client.userID]; ok {
				if _, exists := conns[client]; exists {
					delete(conns, client)
					close(client.send)
					if len(conns) == 0 {
						delete(h.clients, client.userID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("ws: user %d disconnected", client.userID)
		}
	}
}

// SendToUser sends a message to all connections of a specific user.
func (h *Hub) SendToUser(userID int, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("ws: marshal message: %v", err)
		return
	}

	h.mu.RLock()
	conns, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for client := range conns {
		select {
		case client.send <- data:
		default:
			// Client send buffer full, close the connection
			h.mu.Lock()
			delete(conns, client)
			close(client.send)
			if len(conns) == 0 {
				delete(h.clients, userID)
			}
			h.mu.Unlock()
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("ws: marshal broadcast: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, conns := range h.clients {
		for client := range conns {
			select {
			case client.send <- data:
			default:
				// skip slow clients
			}
		}
	}
}

func (h *Hub) countConnections(userID int) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}
