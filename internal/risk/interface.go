package risk

// RiskManager defines the interface for risk management
type RiskManager interface {
	// ValidateOrder validates if an order meets risk management criteria
	ValidateOrder(order *Order, portfolio *Portfolio) error

	// CalculatePositionSize calculates the appropriate position size based on signal strength
	CalculatePositionSize(signal Signal, balance float64) float64

	// ShouldStopTrading determines if trading should be stopped based on portfolio state
	ShouldStopTrading(portfolio *Portfolio) bool
}
