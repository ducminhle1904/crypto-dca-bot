package monitoring

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type HealthChecker struct {
	mu          sync.RWMutex
	lastTrade   time.Time
	lastPrice   float64
	isConnected bool
	errors      []string
}

type HealthStatus struct {
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
	LastTrade   time.Time `json:"last_trade"`
	LastPrice   float64   `json:"last_price"`
	IsConnected bool      `json:"is_connected"`
	Uptime      string    `json:"uptime"`
	Errors      []string  `json:"errors,omitempty"`
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		errors: make([]string, 0),
	}
}

func (h *HealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status := "healthy"
	if !h.isConnected || time.Since(h.lastTrade) > time.Hour*24 {
		status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if len(h.errors) > 0 {
		status = "unhealthy"
		w.WriteHeader(http.StatusInternalServerError)
	}

	health := HealthStatus{
		Status:      status,
		Timestamp:   time.Now(),
		LastTrade:   h.lastTrade,
		LastPrice:   h.lastPrice,
		IsConnected: h.isConnected,
		Uptime:      time.Since(startTime).String(),
		Errors:      h.errors,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}
