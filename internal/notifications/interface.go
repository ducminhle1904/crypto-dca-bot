package notifications

// Notifier defines the interface for notification services
type Notifier interface {
	// SendAlert sends an alert with the specified level and message
	SendAlert(level, message string) error
}
