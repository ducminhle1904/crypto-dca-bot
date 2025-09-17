package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// PortfolioMonitor provides centralized monitoring for the portfolio
type PortfolioMonitor struct {
	botManager      *BotManager
	portfolioConfig *PortfolioConfig
	startTime       time.Time
	healthChecks    map[string]*HealthCheck
	alerts          []Alert
	mu              sync.RWMutex
	stopChan        chan struct{}
	stopped         bool // Track if monitor is already stopped
}

// HealthCheck represents a health check for a bot
type HealthCheck struct {
	BotID        string
	LastCheck    time.Time
	Status       string
	ResponseTime time.Duration
	Healthy      bool
	Errors       []string
}

// Alert represents a portfolio alert
type Alert struct {
	Type      AlertType
	Message   string
	BotID     string
	Timestamp time.Time
	Severity  AlertSeverity
	Resolved  bool
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypePortfolioDrawdown AlertType = "portfolio_drawdown"
	AlertTypeBotLoss          AlertType = "bot_loss"
	AlertTypeBotError         AlertType = "bot_error"
	AlertTypeBotOffline       AlertType = "bot_offline"
	AlertTypeCorrelation      AlertType = "correlation_spike"
	AlertTypeHealthCheck      AlertType = "health_check"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "INFO"
	SeverityWarning  AlertSeverity = "WARNING"
	SeverityError    AlertSeverity = "ERROR"
	SeverityCritical AlertSeverity = "CRITICAL"
)

// NewPortfolioMonitor creates a new portfolio monitor
func NewPortfolioMonitor(botManager *BotManager, portfolioConfig *PortfolioConfig) *PortfolioMonitor {
	return &PortfolioMonitor{
		botManager:      botManager,
		portfolioConfig: portfolioConfig,
		startTime:       time.Now(),
		healthChecks:    make(map[string]*HealthCheck),
		alerts:          make([]Alert, 0),
		stopChan:        make(chan struct{}),
	}
}

// Start begins the monitoring process (safe to call multiple times)
func (pm *PortfolioMonitor) Start() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.stopped {
		fmt.Printf("⚠️ Cannot start already stopped monitor\n")
		return
	}
	
	fmt.Printf("📊 Starting portfolio monitoring...\n")
	
	// Parse monitoring intervals
	healthCheckInterval, err := time.ParseDuration(pm.portfolioConfig.Monitoring.HealthCheckInterval)
	if err != nil {
		healthCheckInterval = 60 * time.Second // Default 1 minute
	}
	
	heartbeatInterval, err := time.ParseDuration(pm.portfolioConfig.Monitoring.HeartbeatInterval)
	if err != nil {
		heartbeatInterval = 30 * time.Second // Default 30 seconds
	}
	
	// Start monitoring routines
	go pm.healthCheckLoop(healthCheckInterval)
	go pm.heartbeatLoop(heartbeatInterval)
	go pm.alertProcessor()
	
	fmt.Printf("✅ Portfolio monitoring started\n")
}

// Stop stops the monitoring process (safe to call multiple times)
func (pm *PortfolioMonitor) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.stopped {
		return // Already stopped
	}
	
	fmt.Printf("🛑 Stopping portfolio monitoring...\n")
	close(pm.stopChan)
	pm.stopped = true
	fmt.Printf("✅ Portfolio monitoring stopped\n")
}

// healthCheckLoop performs periodic health checks on all bots
func (pm *PortfolioMonitor) healthCheckLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pm.performHealthChecks()
		case <-pm.stopChan:
			return
		}
	}
}

// heartbeatLoop provides regular status updates
func (pm *PortfolioMonitor) heartbeatLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pm.logPortfolioStatus()
		case <-pm.stopChan:
			return
		}
	}
}

// alertProcessor handles alert processing and notifications
func (pm *PortfolioMonitor) alertProcessor() {
	// This could be enhanced to send alerts via Telegram, email, etc.
	for {
		select {
		case <-time.After(5 * time.Second):
			pm.checkForAlerts()
		case <-pm.stopChan:
			return
		}
	}
}

// performHealthChecks checks the health of all bots
func (pm *PortfolioMonitor) performHealthChecks() {
	// First, safely get a snapshot of all bot instances
	pm.botManager.mu.RLock()
	instances := make(map[string]*BotInstance, len(pm.botManager.instances))
	for id, instance := range pm.botManager.instances {
		instances[id] = instance
	}
	pm.botManager.mu.RUnlock()
	
	// Now perform health checks with monitor lock
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for botID, instance := range instances {
		healthCheck := &HealthCheck{
			BotID:     botID,
			LastCheck: time.Now(),
			Errors:    make([]string, 0),
		}
		
		// Check bot status
		status := instance.GetStatus()
		healthCheck.Status = string(status)
		
		switch status {
		case StatusRunning:
			healthCheck.Healthy = true
		case StatusError:
			healthCheck.Healthy = false
			if err := instance.GetError(); err != nil {
				healthCheck.Errors = append(healthCheck.Errors, err.Error())
			}
		case StatusStopped, StatusShutdown:
			healthCheck.Healthy = false
			healthCheck.Errors = append(healthCheck.Errors, "Bot is not running")
		default:
			healthCheck.Healthy = false
			healthCheck.Errors = append(healthCheck.Errors, "Unknown status")
		}
		
		// Check error count
		errorCount := instance.GetErrorCount()
		if errorCount > 5 {
			healthCheck.Healthy = false
			healthCheck.Errors = append(healthCheck.Errors, 
				fmt.Sprintf("High error count: %d", errorCount))
		}
		
		pm.healthChecks[botID] = healthCheck
		
		// Generate alerts for unhealthy bots
		if !healthCheck.Healthy {
			pm.generateAlert(AlertTypeBotError, botID, 
				fmt.Sprintf("Bot %s is unhealthy: %s", botID, strings.Join(healthCheck.Errors, ", ")),
				SeverityError)
		}
	}
}

// logPortfolioStatus logs the current portfolio status
func (pm *PortfolioMonitor) logPortfolioStatus() {
	statuses := pm.botManager.GetAllBotStatuses()
	runningCount := pm.botManager.GetRunningBotCount()
	totalBots := len(statuses)
	
	uptime := time.Since(pm.startTime)
	
	fmt.Printf("💼 Portfolio Status [%s] - Running: %d/%d bots | Uptime: %v\n", 
		time.Now().Format("15:04:05"), runningCount, totalBots, 
		uptime.Truncate(time.Second))
	
	// Log individual bot statuses
	for botID, status := range statuses {
		statusIcon := pm.getStatusIcon(status)
		fmt.Printf("   %s %s: %s", statusIcon, botID, status)
		
		if instance, err := pm.botManager.GetBotInstance(botID); err == nil {
			errorCount := instance.GetErrorCount()
			if errorCount > 0 {
				fmt.Printf(" (%d errors)", errorCount)
			}
		}
		fmt.Println()
	}
}

// getStatusIcon returns an icon for the bot status
func (pm *PortfolioMonitor) getStatusIcon(status BotStatus) string {
	switch status {
	case StatusRunning:
		return "🟢"
	case StatusStarting:
		return "🟡"
	case StatusError:
		return "🔴"
	case StatusStopped:
		return "⚫"
	case StatusShutdown:
		return "🔵"
	default:
		return "❓"
	}
}

// checkForAlerts checks for portfolio-level alerts
func (pm *PortfolioMonitor) checkForAlerts() {
	// Check if any bots are offline for too long
	pm.checkBotOfflineAlerts()
	
	// Check for portfolio-level issues
	pm.checkPortfolioHealthAlerts()
}

// checkBotOfflineAlerts checks for bots that have been offline too long
func (pm *PortfolioMonitor) checkBotOfflineAlerts() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	for botID, healthCheck := range pm.healthChecks {
		if !healthCheck.Healthy && time.Since(healthCheck.LastCheck) > 5*time.Minute {
			pm.generateAlert(AlertTypeBotOffline, botID,
				fmt.Sprintf("Bot %s has been offline for %v", botID, time.Since(healthCheck.LastCheck)),
				SeverityCritical)
		}
	}
}

// checkPortfolioHealthAlerts checks overall portfolio health
func (pm *PortfolioMonitor) checkPortfolioHealthAlerts() {
	runningCount := pm.botManager.GetRunningBotCount()
	totalEnabledBots := len(pm.portfolioConfig.GetEnabledBots())
	
	if runningCount == 0 {
		pm.generateAlert(AlertTypePortfolioDrawdown, "",
			"No bots are currently running",
			SeverityCritical)
	} else if float64(runningCount)/float64(totalEnabledBots) < 0.5 {
		pm.generateAlert(AlertTypePortfolioDrawdown, "",
			fmt.Sprintf("Only %d/%d bots are running (< 50%%)", runningCount, totalEnabledBots),
			SeverityWarning)
	}
}

// generateAlert generates a new alert
func (pm *PortfolioMonitor) generateAlert(alertType AlertType, botID, message string, severity AlertSeverity) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Check if similar alert already exists and is not resolved
	for i := range pm.alerts {
		if pm.alerts[i].Type == alertType && 
		   pm.alerts[i].BotID == botID && 
		   !pm.alerts[i].Resolved &&
		   time.Since(pm.alerts[i].Timestamp) < 5*time.Minute {
			// Don't create duplicate alerts within 5 minutes
			return
		}
	}
	
	alert := Alert{
		Type:      alertType,
		Message:   message,
		BotID:     botID,
		Timestamp: time.Now(),
		Severity:  severity,
		Resolved:  false,
	}
	
	pm.alerts = append(pm.alerts, alert)
	
	// Log the alert
	severityIcon := pm.getSeverityIcon(severity)
	fmt.Printf("%s ALERT [%s] %s: %s\n", 
		severityIcon, 
		severity, 
		alertType, 
		message)
}

// getSeverityIcon returns an icon for the alert severity
func (pm *PortfolioMonitor) getSeverityIcon(severity AlertSeverity) string {
	switch severity {
	case SeverityInfo:
		return "ℹ️"
	case SeverityWarning:
		return "⚠️"
	case SeverityError:
		return "❌"
	case SeverityCritical:
		return "🚨"
	default:
		return "❓"
	}
}

// GetPortfolioSummary returns a summary of the portfolio status
func (pm *PortfolioMonitor) GetPortfolioSummary() PortfolioSummary {
	// Get data from bot manager safely
	statuses := pm.botManager.GetAllBotStatuses()
	runningBots := pm.botManager.GetRunningBotCount()
	
	// Now lock monitor for our data
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	summary := PortfolioSummary{
		TotalBots:    len(statuses),
		RunningBots:  runningBots,
		Uptime:       time.Since(pm.startTime),
		Alerts:       pm.getActiveAlerts(),
		BotStatuses:  make(map[string]BotSummary),
	}
	
	for botID, status := range statuses {
		botSummary := BotSummary{
			Status: string(status),
			Healthy: status == StatusRunning,
		}
		
		if instance, err := pm.botManager.GetBotInstance(botID); err == nil {
			botSummary.Uptime = instance.GetUptime()
			botSummary.ErrorCount = instance.GetErrorCount()
			if lastError := instance.GetError(); lastError != nil {
				botSummary.LastError = lastError.Error()
			}
		}
		
		if healthCheck, exists := pm.healthChecks[botID]; exists {
			botSummary.LastHealthCheck = healthCheck.LastCheck
		}
		
		summary.BotStatuses[botID] = botSummary
	}
	
	return summary
}

// getActiveAlerts returns currently active alerts
func (pm *PortfolioMonitor) getActiveAlerts() []Alert {
	var active []Alert
	for _, alert := range pm.alerts {
		if !alert.Resolved && time.Since(alert.Timestamp) < 1*time.Hour {
			active = append(active, alert)
		}
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(active, func(i, j int) bool {
		return active[i].Timestamp.After(active[j].Timestamp)
	})
	
	return active
}

// PortfolioSummary represents a summary of the portfolio status
type PortfolioSummary struct {
	TotalBots   int                    `json:"total_bots"`
	RunningBots int                    `json:"running_bots"`
	Uptime      time.Duration          `json:"uptime"`
	Alerts      []Alert                `json:"alerts"`
	BotStatuses map[string]BotSummary  `json:"bot_statuses"`
}

// BotSummary represents a summary of a bot's status
type BotSummary struct {
	Status          string        `json:"status"`
	Healthy         bool          `json:"healthy"`
	Uptime          time.Duration `json:"uptime"`
	ErrorCount      int           `json:"error_count"`
	LastError       string        `json:"last_error,omitempty"`
	LastHealthCheck time.Time     `json:"last_health_check"`
}

// PrintDetailedStatus prints a detailed status report
func (pm *PortfolioMonitor) PrintDetailedStatus() {
	summary := pm.GetPortfolioSummary()
	
	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                    PORTFOLIO STATUS REPORT                  ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Portfolio: %s\n", pm.portfolioConfig.Description)
	fmt.Printf("║ Uptime: %v\n", summary.Uptime.Truncate(time.Second))
	fmt.Printf("║ Bots: %d/%d running\n", summary.RunningBots, summary.TotalBots)
	fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
	
	// Bot details
	for botID, botSummary := range summary.BotStatuses {
		icon := pm.getStatusIcon(BotStatus(botSummary.Status))
		fmt.Printf("║ %s %-15s │ Status: %-10s │ Uptime: %-10v ║\n", 
			icon, botID, botSummary.Status, botSummary.Uptime.Truncate(time.Second))
		
		if botSummary.ErrorCount > 0 {
			fmt.Printf("║   └─ Errors: %d │ Last: %s\n", 
				botSummary.ErrorCount, 
				truncateString(botSummary.LastError, 35))
		}
	}
	
	// Active alerts
	if len(summary.Alerts) > 0 {
		fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
		fmt.Printf("║                        ACTIVE ALERTS                        ║\n")
		fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
		
		for _, alert := range summary.Alerts {
			icon := pm.getSeverityIcon(alert.Severity)
			age := time.Since(alert.Timestamp).Truncate(time.Minute)
			fmt.Printf("║ %s [%s] %s (%v ago)\n", 
				icon, alert.Severity, alert.Message, age)
		}
	}
	
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")
}

// truncateString truncates a string to the specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}
