package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Simulation Manager State
type simulationControl struct {
	cancelFunc context.CancelFunc
	// We might add more per-user control state here later if needed
}

var (
	activeSimulations      = make(map[string]*simulationControl)
	activeSimulationsMutex sync.Mutex
	currentSimulationCtx   context.Context
	currentUsers           []string
)

const simulationDuration = 30 * time.Second

func handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	stopSimulationsInternal()

	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.UserIDs) == 0 {
		http.Error(w, "User ID list cannot be empty", http.StatusBadRequest)
		return
	}

	log.Printf("Received /start request for %d users", len(req.UserIDs))

	activeSimulationsMutex.Lock()
	// Create a new parent context for this run with a timeout
	currentSimulationCtx, _ = context.WithTimeout(context.Background(), simulationDuration)
	currentUsers = make([]string, len(req.UserIDs))
	copy(currentUsers, req.UserIDs)

	for _, userID := range req.UserIDs {
		if userID == "" {
			log.Println("Skipping empty user ID in start request")
			continue
		}
		userCtx, cancel := context.WithCancel(currentSimulationCtx)
		activeSimulations[userID] = &simulationControl{cancelFunc: cancel}
		go SimulateUserMovement(userCtx, userID)
	}
	activeSimulationsMutex.Unlock()

	go func() {
		<-currentSimulationCtx.Done()
		if currentSimulationCtx.Err() == context.DeadlineExceeded {
			log.Printf("Simulation duration (%s) reached, stopping automatically.", simulationDuration)
			stopSimulationsInternal() // Ensure cleanup if timeout hits
		}
	}()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Simulation started for %d users. Will run for approximately %s.\n", len(req.UserIDs), simulationDuration)
}

// stopSimulationsInternal stops all active simulations. Assumes mutex is handled by caller or not needed.
func stopSimulationsInternal() {
	activeSimulationsMutex.Lock()
	defer activeSimulationsMutex.Unlock()

	if len(activeSimulations) == 0 {
		log.Println("Stop request received, but no simulations are currently active.")
		return
	}

	log.Printf("Stopping %d active simulations...", len(activeSimulations))
	for userID, control := range activeSimulations {
		control.cancelFunc() // Signal the goroutine to stop
		delete(activeSimulations, userID)
	}
	currentUsers = nil
	if currentSimulationCtx != nil {
	}
	log.Println("All simulations stopped.")
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	stopSimulationsInternal()
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "All active simulations stopped.")
}

func handleGetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Expecting path like /user/{user_id}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(pathParts) != 2 || pathParts[0] != "user" || pathParts[1] == "" {
		http.Error(w, "Invalid URL path. Expected /user/{user_id}", http.StatusBadRequest)
		return
	}
	userID := pathParts[1]

	// Parse min and max query parameters
	minPercentStr := r.URL.Query().Get("min")
	maxPercentStr := r.URL.Query().Get("max")

	minPercent := 0.0
	maxPercent := 1.0
	var err error

	if minPercentStr != "" {
		minPercent, err = strconv.ParseFloat(minPercentStr, 64)
		if err != nil || minPercent < 0.0 || minPercent > 1.0 {
			http.Error(w, "Invalid 'min' parameter. Must be a float between 0.0 and 1.0.", http.StatusBadRequest)
			return
		}
	}

	if maxPercentStr != "" {
		maxPercent, err = strconv.ParseFloat(maxPercentStr, 64)
		if err != nil || maxPercent < 0.0 || maxPercent > 1.0 {
			http.Error(w, "Invalid 'max' parameter. Must be a float between 0.0 and 1.0.", http.StatusBadRequest)
			return
		}
	}

	if minPercent > maxPercent {
		http.Error(w, "'min' parameter cannot be greater than 'max' parameter.", http.StatusBadRequest)
		return
	}

	log.Printf("GET /user/%s request with min=%.2f, max=%.2f", userID, minPercent, maxPercent)

	userData, err := ReadLocationData(userID, minPercent, maxPercent)
	if err != nil {
		log.Printf("Error reading data for user %s: %v", userID, err)
		http.Error(w, fmt.Sprintf("Failed to retrieve data for user %s: %v", userID, err), http.StatusInternalServerError)
		return
	}

	if len(userData) == 0 {
		// Check if the original range might have filtered out all data, or if user truly has no data
		// For simplicity, we'll return Not Found. A more sophisticated check could query without percentages first.
		http.Error(w, fmt.Sprintf("No data found for user %s within the specified range", userID), http.StatusNotFound)
		return
	}

	response := UserDataResponse{
		UserID: userID,
		Data:   userData,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding /user/%s response: %v", userID, err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
