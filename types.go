package main

import "time"

// TODO: Define shared data structures

// LocationData represents a single coordinate change record.
// Stored as JSON in unitdb message payload.
type LocationData struct {
	DeltaX    float64   `json:"dx"`
	DeltaY    float64   `json:"dy"`
	Timestamp time.Time `json:"ts"` // Store timestamp explicitly for easier retrieval/sorting
}

// StartRequest is the expected JSON body for the /start endpoint.
type StartRequest struct {
	UserIDs []string `json:"user_ids"`
}

// UserDataResponse is the structure for returning data for a user.
type UserDataResponse struct {
	UserID string         `json:"user_id"`
	Data   []LocationData `json:"data"`
}
