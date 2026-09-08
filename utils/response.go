// utils/response - Helper functions for sending consistent JSON responses.
package utils

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON response with the given HTTP status code and data.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Error writes a JSON error response in the format {"error": "message"}.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}
