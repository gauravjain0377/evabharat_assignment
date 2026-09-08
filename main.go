// Ticket System - A REST API for managing support tickets.
// This is the entry point that wires everything together and starts the server.
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/gauravjain0377/ticket-system/config"
	"github.com/gauravjain0377/ticket-system/handlers"
	"github.com/gauravjain0377/ticket-system/middleware"
	"github.com/gauravjain0377/ticket-system/store"
)

func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	// Initialize the SQLite database
	db, err := store.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create handler instances with their dependencies
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret)
	ticketHandler := handlers.NewTicketHandler(db)

	// Set up the router
	router := mux.NewRouter()

	// Add CORS middleware for frontend support
	router.Use(corsMiddleware)

	// Public routes (no authentication required)
	router.HandleFunc("/health", handlers.HealthCheck).Methods("GET")
	router.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/auth/login", authHandler.Login).Methods("POST")

	// Protected routes (JWT authentication required)
	protected := router.PathPrefix("/tickets").Subrouter()
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	protected.HandleFunc("", ticketHandler.CreateTicket).Methods("POST")
	protected.HandleFunc("", ticketHandler.ListTickets).Methods("GET")
	protected.HandleFunc("/{id}", ticketHandler.GetTicket).Methods("GET")
	protected.HandleFunc("/{id}/status", ticketHandler.UpdateStatus).Methods("PATCH")

	// Serve frontend static files
	frontendDir := http.Dir("./frontend")
	router.PathPrefix("/").Handler(http.FileServer(frontendDir))

	// Start the server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

// corsMiddleware adds CORS headers to allow frontend requests.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight OPTIONS requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
