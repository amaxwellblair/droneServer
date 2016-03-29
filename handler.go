package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

// Handler deals with requests
type Handler struct {
	mu    sync.Mutex
	Queue []*Connection
}

// Connection holds the connection to a drone client
type Connection struct {
	C    chan []*ActionsResponse
	Wait chan struct{}
}

// ActionsResponse contains the actions for the drone client
type ActionsResponse struct {
	ItemID  int
	Actions []*string
}

// ActionsRequest contains the actions from the pilot
type ActionsRequest struct {
	ItemID  int
	Actions []*string
}

// NewHandler creates a new handler instance
func NewHandler() *Handler {
	return &Handler{}
}

// NewConnection creates a new connection instance
func NewConnection() *Connection {
	return &Connection{
		C:    make(chan []*ActionsResponse, 0),
		Wait: make(chan struct{}, 0),
	}
}

// ServeHTTP creates a new router
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/connect":
		if r.Method == "POST" {
			h.postConnectHandler(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	case "/actions":
		if r.Method == "GET" {
			h.getActionsHandler(w, r)
		} else if r.Method == "POST" {
			h.postActionsHandler(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
}

// postConnectHandler creates a connection between droneServer and droneClient
func (h *Handler) postConnectHandler(w http.ResponseWriter, r *http.Request) {
	c := NewConnection()

	// Add new connection to the queue
	h.mu.Lock()
	h.Queue = append(h.Queue, c)
	h.mu.Unlock()

	// Wait until actions are posted then get actions
	<-c.Wait
	http.Redirect(w, r, "/actions", http.StatusFound)
}

// getActionsHandler sends droneClient actions to execute
func (h *Handler) getActionsHandler(w http.ResponseWriter, r *http.Request) {
	// Read and remove the first connection from the queue
	h.mu.Lock()
	c := h.Queue[0]
	h.Queue = h.Queue[1:]
	h.mu.Unlock()

	// Encode action to json and send successful response
	if err := json.NewEncoder(w).Encode(<-c.C); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) postActionsHandler(w http.ResponseWriter, r *http.Request) {
	// Decode the request
	var ar ActionsRequest
	if err := json.NewDecoder(r.Body).Decode(&ar); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
}
