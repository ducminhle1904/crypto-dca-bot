package portfolio

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SyncManager handles real-time coordination between multiple bot instances
type SyncManager struct {
	mu              sync.RWMutex
	stateManager    StateManager
	allocManager    *AllocationManager
	botID           string
	heartbeatFreq   time.Duration
	heartbeatCtx    context.Context
	heartbeatCancel context.CancelFunc
	eventHandlers   []EventHandler
	lastSyncTime    time.Time
	syncInterval    time.Duration
	isRunning       bool
}

// EventHandler defines the interface for handling sync events
type EventHandler interface {
	HandleEvent(event SyncEvent) error
}

// SyncEvent represents a synchronization event
type SyncEvent struct {
	Type        string                 `json:"type"`         // REGISTER, UNREGISTER, POSITION_UPDATE, PROFIT_RECORD, REBALANCE
	BotID       string                 `json:"bot_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	RequiresAck bool                   `json:"requires_ack"`
}

// HeartbeatInfo tracks bot heartbeat information
type HeartbeatInfo struct {
	BotID       string    `json:"bot_id"`
	LastSeen    time.Time `json:"last_seen"`
	Status      string    `json:"status"`      // ACTIVE, INACTIVE, DEAD
	Version     string    `json:"version"`     // Bot version for compatibility
	PID         int       `json:"pid"`         // Process ID
	Hostname    string    `json:"hostname"`    // Host machine
}

// SyncStats provides synchronization statistics
type SyncStats struct {
	LastSync          time.Time `json:"last_sync"`
	SyncCount         int64     `json:"sync_count"`
	ConflictCount     int64     `json:"conflict_count"`
	SuccessfulSyncs   int64     `json:"successful_syncs"`
	FailedSyncs       int64     `json:"failed_syncs"`
	AverageSync       float64   `json:"average_sync_ms"`
	ActiveBots        int       `json:"active_bots"`
	DeadBots          int       `json:"dead_bots"`
	LastConflict      time.Time `json:"last_conflict"`
	LockContentions   int64     `json:"lock_contentions"`
	StateCorruptions  int64     `json:"state_corruptions"`
}

// NewSyncManager creates a new sync manager
func NewSyncManager(botID string, stateManager StateManager, allocManager *AllocationManager) *SyncManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SyncManager{
		stateManager:    stateManager,
		allocManager:    allocManager,
		botID:           botID,
		heartbeatFreq:   30 * time.Second, // Default heartbeat every 30s
		heartbeatCtx:    ctx,
		heartbeatCancel: cancel,
		eventHandlers:   make([]EventHandler, 0),
		syncInterval:    5 * time.Second, // Default sync every 5s
		isRunning:       false,
	}
}

// Start begins sync operations
func (sm *SyncManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.isRunning {
		return fmt.Errorf("sync manager is already running")
	}
	
	// Start heartbeat goroutine
	go sm.heartbeatLoop()
	
	// Start sync monitoring goroutine
	go sm.syncLoop()
	
	sm.isRunning = true
	sm.lastSyncTime = time.Now()
	
	// Send registration event
	event := SyncEvent{
		Type:      "REGISTER",
		BotID:     sm.botID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"startup": true,
		},
	}
	
	sm.broadcastEvent(event)
	
	return nil
}

// Stop gracefully stops sync operations
func (sm *SyncManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if !sm.isRunning {
		return nil
	}
	
	// Send unregistration event
	event := SyncEvent{
		Type:      "UNREGISTER",
		BotID:     sm.botID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"shutdown": true,
		},
	}
	
	sm.broadcastEvent(event)
	
	// Cancel heartbeat
	sm.heartbeatCancel()
	
	sm.isRunning = false
	
	return nil
}

// SyncState synchronizes the current state with other instances
func (sm *SyncManager) SyncState() error {
	syncStart := time.Now()
	
	// Try to acquire lock with timeout
	lockTimeout := 5 * time.Second
	lockAcquired := make(chan bool, 1)
	
	go func() {
		err := sm.stateManager.Lock()
		lockAcquired <- (err == nil)
	}()
	
	select {
	case acquired := <-lockAcquired:
		if !acquired {
			return fmt.Errorf("failed to acquire state lock within timeout")
		}
	case <-time.After(lockTimeout):
		return fmt.Errorf("lock acquisition timeout")
	}
	
	defer func() {
		if err := sm.stateManager.Unlock(); err != nil {
			fmt.Printf("⚠️ Warning: Failed to release lock: %v\n", err)
		}
	}()
	
	// Load current state
	state, err := sm.stateManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	
	// Validate state integrity
	if err := sm.validateStateIntegrity(state); err != nil {
		return fmt.Errorf("state integrity check failed: %w", err)
	}
	
	// Update local allocation manager with remote state
	if err := sm.syncAllocationManager(state); err != nil {
		return fmt.Errorf("failed to sync allocation manager: %w", err)
	}
	
	sm.lastSyncTime = time.Now()
	
	// Update sync statistics
	syncDuration := time.Since(syncStart)
	fmt.Printf("🔄 State sync completed for %s in %v\n", sm.botID, syncDuration)
	
	return nil
}

// UpdatePosition notifies other bots of position change
func (sm *SyncManager) UpdatePosition(positionValue, avgPrice, leverage float64) error {
	event := SyncEvent{
		Type:        "POSITION_UPDATE",
		BotID:       sm.botID,
		Timestamp:   time.Now(),
		RequiresAck: true,
		Data: map[string]interface{}{
			"position_value": positionValue,
			"avg_price":      avgPrice,
			"leverage":       leverage,
		},
	}
	
	return sm.broadcastEventWithLock(event)
}

// RecordProfit notifies other bots of profit recording
func (sm *SyncManager) RecordProfit(profit float64, shareProfit bool) error {
	event := SyncEvent{
		Type:        "PROFIT_RECORD",
		BotID:       sm.botID,
		Timestamp:   time.Now(),
		RequiresAck: shareProfit, // Only require ack if profit sharing is enabled
		Data: map[string]interface{}{
			"profit":       profit,
			"share_profit": shareProfit,
		},
	}
	
	return sm.broadcastEventWithLock(event)
}

// TriggerRebalance initiates portfolio rebalancing across all bots
func (sm *SyncManager) TriggerRebalance() error {
	event := SyncEvent{
		Type:        "REBALANCE",
		BotID:       sm.botID,
		Timestamp:   time.Now(),
		RequiresAck: true,
		Data: map[string]interface{}{
			"trigger": "manual",
		},
	}
	
	return sm.broadcastEventWithLock(event)
}

// AddEventHandler registers an event handler
func (sm *SyncManager) AddEventHandler(handler EventHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sm.eventHandlers = append(sm.eventHandlers, handler)
}

// GetActiveHeartbeats returns current bot heartbeat information
func (sm *SyncManager) GetActiveHeartbeats() (map[string]*HeartbeatInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	// For now, return a simple implementation
	// In a full implementation, this would read from a shared heartbeat store
	heartbeats := make(map[string]*HeartbeatInfo)
	
	// Add self
	heartbeats[sm.botID] = &HeartbeatInfo{
		BotID:    sm.botID,
		LastSeen: time.Now(),
		Status:   "ACTIVE",
		Version:  "2.0.0",
		PID:      123, // Would get actual PID
		Hostname: "localhost",
	}
	
	return heartbeats, nil
}

// GetSyncStats returns synchronization statistics
func (sm *SyncManager) GetSyncStats() *SyncStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	return &SyncStats{
		LastSync:        sm.lastSyncTime,
		SyncCount:       0, // Would track in full implementation
		ConflictCount:   0, // Would track conflicts
		SuccessfulSyncs: 0, // Would track success rate
		FailedSyncs:     0,
		AverageSync:     0.0, // Would calculate average
		ActiveBots:      1,   // Would count active bots
		DeadBots:        0,   // Would count dead bots
	}
}

// IsHealthy returns true if sync manager is operating normally
func (sm *SyncManager) IsHealthy() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if !sm.isRunning {
		return false
	}
	
	// Check if sync is too stale
	maxSyncAge := 60 * time.Second
	if time.Since(sm.lastSyncTime) > maxSyncAge {
		return false
	}
	
	return true
}

// WaitForSync waits for next synchronization cycle
func (sm *SyncManager) WaitForSync(timeout time.Duration) error {
	start := time.Now()
	lastSync := sm.lastSyncTime
	
	for time.Since(start) < timeout {
		time.Sleep(100 * time.Millisecond)
		
		sm.mu.RLock()
		currentSync := sm.lastSyncTime
		sm.mu.RUnlock()
		
		if currentSync.After(lastSync) {
			return nil // Sync occurred
		}
	}
	
	return fmt.Errorf("sync timeout after %v", timeout)
}

// Private methods

func (sm *SyncManager) heartbeatLoop() {
	ticker := time.NewTicker(sm.heartbeatFreq)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := sm.sendHeartbeat(); err != nil {
				fmt.Printf("⚠️ Heartbeat failed for %s: %v\n", sm.botID, err)
			}
		case <-sm.heartbeatCtx.Done():
			return
		}
	}
}

func (sm *SyncManager) syncLoop() {
	ticker := time.NewTicker(sm.syncInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if sm.isRunning {
				if err := sm.SyncState(); err != nil {
					fmt.Printf("⚠️ Sync failed for %s: %v\n", sm.botID, err)
				}
			}
		case <-sm.heartbeatCtx.Done():
			return
		}
	}
}

func (sm *SyncManager) sendHeartbeat() error {
	event := SyncEvent{
		Type:      "HEARTBEAT",
		BotID:     sm.botID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":   "ACTIVE",
			"version":  "2.0.0",
			"hostname": "localhost",
		},
	}
	
	return sm.broadcastEvent(event)
}

func (sm *SyncManager) broadcastEvent(event SyncEvent) error {
	for _, handler := range sm.eventHandlers {
		if err := handler.HandleEvent(event); err != nil {
			fmt.Printf("⚠️ Event handler failed: %v\n", err)
			// Continue with other handlers
		}
	}
	
	// In a full implementation, this would also write events to a shared store
	return nil
}

func (sm *SyncManager) broadcastEventWithLock(event SyncEvent) error {
	// Acquire lock before broadcasting critical events
	if err := sm.stateManager.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock for critical event: %w", err)
	}
	defer sm.stateManager.Unlock()
	
	// Update local allocation manager first
	switch event.Type {
	case "POSITION_UPDATE":
		positionValue := event.Data["position_value"].(float64)
		avgPrice := event.Data["avg_price"].(float64)
		leverage := event.Data["leverage"].(float64)
		
		if err := sm.allocManager.UpdateBotPosition(sm.botID, positionValue, avgPrice, leverage); err != nil {
			return fmt.Errorf("failed to update local allocation: %w", err)
		}
		
	case "PROFIT_RECORD":
		profit := event.Data["profit"].(float64)
		shareProfit := event.Data["share_profit"].(bool)
		
		if err := sm.allocManager.RecordProfit(sm.botID, profit, shareProfit); err != nil {
			return fmt.Errorf("failed to record local profit: %w", err)
		}
	}
	
	// Persist state changes
	if err := sm.persistState(); err != nil {
		return fmt.Errorf("failed to persist state: %w", err)
	}
	
	// Broadcast to handlers
	return sm.broadcastEvent(event)
}

func (sm *SyncManager) validateStateIntegrity(state *PortfolioState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}
	
	if state.TotalBalance <= 0 {
		return fmt.Errorf("invalid total balance: %.2f", state.TotalBalance)
	}
	
	if state.Allocations == nil {
		return fmt.Errorf("allocations map is nil")
	}
	
	// Check allocation integrity
	totalAllocated := 0.0
	for botID, allocation := range state.Allocations {
		if allocation == nil {
			return fmt.Errorf("nil allocation for bot %s", botID)
		}
		
		if allocation.BotID != botID {
			return fmt.Errorf("bot ID mismatch: expected %s, got %s", botID, allocation.BotID)
		}
		
		if allocation.AllocatedBalance < 0 {
			return fmt.Errorf("negative allocated balance for bot %s: %.2f", botID, allocation.AllocatedBalance)
		}
		
		totalAllocated += allocation.AllocatedBalance
	}
	
	// Allow some tolerance for floating point precision
	tolerance := 0.01
	if totalAllocated > state.TotalBalance+tolerance {
		return fmt.Errorf("total allocated %.2f exceeds total balance %.2f", totalAllocated, state.TotalBalance)
	}
	
	return nil
}

func (sm *SyncManager) syncAllocationManager(state *PortfolioState) error {
	// This is a simplified version - in a full implementation,
	// we would need more sophisticated state merging
	
	for botID, _ := range state.Allocations {
		if botID == sm.botID {
			continue // Skip self - we manage our own state
		}
		
		// Update allocation manager with remote bot states
		// This would require extending the allocation manager interface
		// For now, just validate that the state is consistent
	}
	
	return nil
}

func (sm *SyncManager) persistState() error {
	// Get current portfolio state
	summary := sm.allocManager.GetPortfolioSummary()
	allocations := sm.allocManager.GetAllAllocations()
	
	// Create state object
	state := &PortfolioState{
		TotalBalance:    summary.TotalBalance,
		TotalProfit:     summary.TotalRealizedPnL,
		LastUpdated:     time.Now(),
		Allocations:     allocations,
		GlobalSettings:  nil, // Would be set from config
		Version:         "2.0",
	}
	
	// Save to state manager
	return sm.stateManager.Save(state)
}

// DefaultEventHandler provides a basic event handler implementation
type DefaultEventHandler struct {
	syncManager *SyncManager
}

// NewDefaultEventHandler creates a default event handler
func NewDefaultEventHandler(sm *SyncManager) EventHandler {
	return &DefaultEventHandler{syncManager: sm}
}

// HandleEvent processes sync events
func (h *DefaultEventHandler) HandleEvent(event SyncEvent) error {
	switch event.Type {
	case "REGISTER":
		fmt.Printf("📝 Bot %s registered at %v\n", event.BotID, event.Timestamp)
		
	case "UNREGISTER":
		fmt.Printf("📝 Bot %s unregistered at %v\n", event.BotID, event.Timestamp)
		
	case "HEARTBEAT":
		// Silently handle heartbeats to avoid spam
		
	case "POSITION_UPDATE":
		fmt.Printf("📊 Bot %s updated position: $%.2f\n", 
			event.BotID, event.Data["position_value"])
		
	case "PROFIT_RECORD":
		fmt.Printf("💰 Bot %s recorded profit: $%.2f\n", 
			event.BotID, event.Data["profit"])
		
	case "REBALANCE":
		fmt.Printf("⚖️ Rebalance triggered by %s\n", event.BotID)
		
	default:
		fmt.Printf("❓ Unknown event type: %s from %s\n", event.Type, event.BotID)
	}
	
	return nil
}
