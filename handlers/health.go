// handlers/health - Health check endpoint.
// Returns {"status": "ok"} to confirm the server is running.
package handlers

import (
	"net/http"

	"github.com/gauravjain0377/ticket-system/utils"
)

// HealthCheck handles GET /health
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	utils.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
