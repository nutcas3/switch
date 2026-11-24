package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// Health check endpoint
func (app *Application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	// Check database health
	if err := app.db.Health(); err != nil {
		health["database"] = "unhealthy"
		health["status"] = "unhealthy"
	} else {
		health["database"] = "healthy"
	}

	// Add stats
	stats := app.tcpServer.GetStats()
	health["tcp_server"] = stats

	routingStats := app.routingSvc.GetStats()
	health["routing"] = routingStats

	hsmStats := app.hsmAdapter.GetKeyInfo()
	health["hsm"] = hsmStats

	w.Header().Set("Content-Type", "application/json")
	if health["status"] == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(health)
}
