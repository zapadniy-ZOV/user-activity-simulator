package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/unit-io/unitdb"
)

var db *unitdb.DB

// InitDB initializes the unitdb database connection.
func InitDB(dbPath string) error {
	var err error
	// Open DB with Mutable flag to allow potential future delete operations if needed,
	// though the current spec doesn't require deletes.
	db, err = unitdb.Open(dbPath, unitdb.WithDefaultOptions(), unitdb.WithMutable())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	log.Println("Database opened successfully at", dbPath)
	return nil
}

// CloseDB closes the database connection.
func CloseDB() {
	if db != nil {
		db.Close()
		log.Println("Database closed.")
	}
}

// GetTopicForUser formats the database topic string for a given user ID.
func GetTopicForUser(userID string) []byte {
	return []byte(fmt.Sprintf("user.%s.location", userID))
}

// WriteLocationData writes a batch of location data points for a specific user.
func WriteLocationData(userID string, dataPoints []LocationData) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	topic := GetTopicForUser(userID)

	return db.Batch(func(b *unitdb.Batch, completed <-chan struct{}) error {
		entry := unitdb.NewEntry(topic, nil)
		for _, data := range dataPoints {
			payload, err := json.Marshal(data)
			if err != nil {
				log.Printf("Error marshalling location data for user %s: %v", userID, err)
				continue
			}
			// Using WithPayload reuses the parsed topic, improving efficiency
			entry.WithPayload(payload)
			if err := b.PutEntry(entry); err != nil {
				log.Printf("Error putting entry in batch for user %s: %v", userID, err)
			}
		}
		return nil // Signal batch completion attempt
	})
}

// ReadLocationData retrieves location data for a specific user, with optional percentage-based slicing.
// minPercent and maxPercent range from 0.0 to 1.0.
func ReadLocationData(userID string, minPercent, maxPercent float64) ([]LocationData, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	topic := GetTopicForUser(userID)
	query := unitdb.NewQuery(topic)

	rawMessages, err := db.Get(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get data for user %s from DB: %w", userID, err)
	}

	if len(rawMessages) == 0 {
		return []LocationData{}, nil
	}

	locationDataList := make([]LocationData, 0, len(rawMessages))
	for _, rawMsg := range rawMessages {
		var data LocationData
		if err := json.Unmarshal(rawMsg, &data); err != nil {
			log.Printf("Error unmarshalling data for user %s: %v. Data: %s", userID, err, string(rawMsg))
			continue // Skip corrupted data
		}
		locationDataList = append(locationDataList, data)
	}

	// Sort data by timestamp (oldest to newest)
	sort.Slice(locationDataList, func(i, j int) bool {
		return locationDataList[i].Timestamp.Before(locationDataList[j].Timestamp)
	})

	// Apply percentage-based slicing
	totalEntries := len(locationDataList)
	if totalEntries == 0 {
		return []LocationData{}, nil
	}

	// Validate and clamp percentages
	if minPercent < 0.0 {
		minPercent = 0.0
	}
	if maxPercent > 1.0 {
		maxPercent = 1.0
	}
	if minPercent > maxPercent {
		minPercent = maxPercent // Or handle as an error, for now, clamp
	}

	startIndex := int(minPercent * float64(totalEntries))
	endIndex := int(maxPercent * float64(totalEntries))

	// Ensure indices are within bounds
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > totalEntries {
		endIndex = totalEntries
	}
	if startIndex > endIndex {
		startIndex = endIndex
	}

	return locationDataList[startIndex:endIndex], nil
}
