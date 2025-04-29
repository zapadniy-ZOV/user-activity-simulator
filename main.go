package main

import (
	"flag"
	// "fmt" // Removed unused import
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Command-line flags
	dbPath := flag.String("dbpath", "./user_movement_db", "Path to the unitdb database directory")
	listenAddr := flag.String("addr", ":8080", "Address and port to listen on")
	flag.Parse()

	// Setup logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Starting User Movement Simulator...")

	// Initialize Database
	if err := InitDB(*dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer CloseDB()

	// Setup signal handling for graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stopChan
		log.Println("Received shutdown signal. Stopping simulations and closing database...")
		stopSimulationsInternal() // Stop any running simulations
		CloseDB()                 // Close DB connection
		log.Println("Shutdown complete.")
		os.Exit(0)
	}()

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/start", handleStart)
	mux.HandleFunc("/stop", handleStop)
	mux.HandleFunc("/user/", handleGetUser) // Register a prefix handler

	server := &http.Server{
		Addr:    *listenAddr,
		Handler: mux,
	}

	log.Printf("Server listening on %s", *listenAddr)
	log.Printf("Database stored at %s", *dbPath)
	log.Println("Endpoints:")
	log.Println("  POST /start   - Body: {\"user_ids\": [\"id1\", \"id2\"]}")
	log.Println("  POST /stop")
	log.Println("  GET  /user/{user_id}")

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}
