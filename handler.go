package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

const (
	// StatusWaiting is a waiting drone status
	StatusWaiting = "Waiting"

	// StatusAssigned is a assigned drone status
	StatusAssigned = "Assigned"
)

// Handler deals with requests
type Handler struct {
	mu    sync.Mutex
	Queue []*Drone
}

// Drone holds the connection to a drone client
type Drone struct {
	DroneID int
	Status  string
	C       chan *ActionsResponse
	Wait    chan *struct{}
}

// NewHandler creates a new handler instance
func NewHandler() *Handler {
	return &Handler{}
}

// NewDrone creates a new connection instance
func NewDrone(droneID int) *Drone {
	return &Drone{
		DroneID: droneID,
		Status:  StatusWaiting,
		C:       make(chan *ActionsResponse, 1),
		Wait:    make(chan *struct{}, 1),
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
	// Decode request and create new drone
	var dr DroneRequest
	if err := json.NewDecoder(r.Body).Decode(&dr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, err := strconv.Atoi(dr.DroneID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c := NewDrone(id)

	// Add new drone to the queue
	h.mu.Lock()
	h.Queue = append(h.Queue, c)
	h.mu.Unlock()

	// Build URL
	u := r.URL
	u.Path = "/actions"
	params := u.Query()
	params.Add("id", dr.DroneID)
	u.RawQuery = params.Encode()

	// Wait until actions are posted then get actions
	<-c.Wait
	fmt.Println(u.String())
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// getActionsHandler sends droneClient actions to execute
func (h *Handler) getActionsHandler(w http.ResponseWriter, r *http.Request) {
	// Decode request
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Read and remove the drone from the queue
	h.mu.Lock()
	var d *Drone
	if len(h.Queue) == 1 {
		if drone := h.Queue[0]; drone.DroneID == id {
			d = drone
			h.Queue = h.Queue[:0]
		}
	} else {
		for index, drone := range h.Queue {
			if drone.DroneID == id {
				d = drone
				h.Queue = append(h.Queue[0:(index)], h.Queue[(index+1):]...)
			}
		}
	}
	h.mu.Unlock()
	if d == nil {
		http.Error(w, "no drone found with this ID", http.StatusInternalServerError)
		return
	}

	// Encode action to json and send successful response
	if err := json.NewEncoder(w).Encode(<-d.C); err != nil {
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

	// Find a drone that is waiting and assign them
	h.mu.Lock()
	defer h.mu.Unlock()
	var d *Drone
	for _, drone := range h.Queue {
		if drone.Status == StatusWaiting {
			d = drone
			break
		}
	}
	if d == nil {
		http.Error(w, "no available drones", http.StatusInternalServerError)
		return
	}
	d.Status = StatusAssigned

	// Send actions to the corresponding drone
	itemID, err := strconv.Atoi(ar.ItemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d.C <- &ActionsResponse{
		ItemID:  itemID,
		Actions: ar.Actions,
	}
	d.Wait <- new(struct{})
}

// DroneRequest contains drone specific information of the client
type DroneRequest struct {
	DroneID string `json:"droneID"`
}

// ActionsResponse contains the actions for the drone client
type ActionsResponse struct {
	ItemID  int
	Actions []*string
}

// ActionsRequest contains the actions from the pilot
type ActionsRequest struct {
	ItemID  string    `json:"itemID"`
	Actions []*string `json:"actions"`
}
