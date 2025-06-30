package risk

type RiskManager interface {
	ValidateOrder(order *Order, portfolio *Portfolio) error
	CalculatePositionSize(signal Signal, balance float64) float64
	ShouldStopTrading(portfolio *Portfolio) bool
}
