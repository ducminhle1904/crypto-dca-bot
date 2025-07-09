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
	startTime   time.Time
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
		errors:    make([]string, 0),
		startTime: time.Now(),
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
		Uptime:      time.Since(h.startTime).String(),
		Errors:      h.errors,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// SetConnected updates the connection status
func (h *HealthChecker) SetConnected(connected bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.isConnected = connected
}

// UpdatePrice updates the last known price
func (h *HealthChecker) UpdatePrice(price float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastPrice = price
}

// UpdateLastTrade updates the last trade timestamp
func (h *HealthChecker) UpdateLastTrade(tradeTime time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastTrade = tradeTime
}

// AddError adds an error to the error list
func (h *HealthChecker) AddError(err string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = append(h.errors, err)

	// Keep only last 10 errors
	if len(h.errors) > 10 {
		h.errors = h.errors[len(h.errors)-10:]
	}
}
