package main

import (
	"context"
	"log"
	"math"
	"math/rand"
	"time"
)

const (
	// Simulation constraints
	maxDistancePerStep = 0.004 // meters (approx 4mm, as requested)
	// Batching configuration for database writes
	batchSize     = 100                    // Number of data points to batch before writing
	flushInterval = 100 * time.Millisecond // Max time between flushes even if batch isn't full
)

// SimulateUserMovement runs the location simulation for a single user.
// It generates random movement data and writes it to the database in batches.
// It stops when the provided context is cancelled.
func SimulateUserMovement(ctx context.Context, userID string) {
	log.Printf("Starting simulation for user %s", userID)
	defer log.Printf("Stopping simulation for user %s", userID)

	// Seed random number generator for this goroutine
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	dataBuffer := make([]LocationData, 0, batchSize)
	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()

	for {
		select {
		case <-ctx.Done(): // Check if context has been cancelled (by /stop or timeout)
			if len(dataBuffer) > 0 {
				if err := WriteLocationData(userID, dataBuffer); err != nil {
					log.Printf("Error flushing final data for user %s: %v", userID, err)
				}
			}
			return
		case <-flushTicker.C:
			// Flush buffer periodically
			if len(dataBuffer) > 0 {
				if err := WriteLocationData(userID, dataBuffer); err != nil {
					log.Printf("Error flushing data buffer for user %s: %v", userID, err)
				}
				dataBuffer = make([]LocationData, 0, batchSize) // Reset buffer
			}
		default:
			// Generate next data point
			timestamp := time.Now()
			deltaX, deltaY := generateMovement(rng)

			dataPoint := LocationData{
				DeltaX:    deltaX,
				DeltaY:    deltaY,
				Timestamp: timestamp,
			}

			dataBuffer = append(dataBuffer, dataPoint)

			// Write to DB if batch is full
			if len(dataBuffer) >= batchSize {
				if err := WriteLocationData(userID, dataBuffer); err != nil {
					// Log error and continue. Data might be lost for this batch.
					log.Printf("Error writing batch data for user %s: %v", userID, err)
				}
				dataBuffer = make([]LocationData, 0, batchSize) // Reset buffer
				flushTicker.Reset(flushInterval)                // Reset ticker after a full batch write
			}
		}
	}
}

// generateMovement creates a small random displacement (dx, dy).
func generateMovement(rng *rand.Rand) (float64, float64) {
	angle := rng.Float64() * 2 * math.Pi
	magnitude := rng.Float64() * maxDistancePerStep

	dx := magnitude * math.Cos(angle)
	dy := magnitude * math.Sin(angle)

	return dx, dy
}
