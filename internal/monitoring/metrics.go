package monitoring

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Trading metrics
	tradesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dca_bot_trades_total",
			Help: "Total number of trades executed",
		},
		[]string{"symbol", "side"},
	)

	tradeAmount = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dca_bot_trade_amount",
			Help:    "Distribution of trade amounts",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"symbol"},
	)

	// Market data metrics
	currentPrice = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dca_bot_current_price",
			Help: "Current price of trading symbol",
		},
		[]string{"symbol"},
	)

	// Strategy metrics
	strategyConfidence = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dca_bot_strategy_confidence",
			Help: "Strategy confidence level",
		},
		[]string{"strategy"},
	)

	// Error metrics
	errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dca_bot_errors_total",
			Help: "Total number of errors",
		},
		[]string{"type"},
	)
)

func init() {
	// Register metrics
	prometheus.MustRegister(tradesTotal)
	prometheus.MustRegister(tradeAmount)
	prometheus.MustRegister(currentPrice)
	prometheus.MustRegister(strategyConfidence)
	prometheus.MustRegister(errorsTotal)
}

// MetricsHandler handles Prometheus metrics endpoint
type MetricsHandler struct{}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// ServeHTTP serves the Prometheus metrics endpoint
func (m *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

// RecordTrade records a trade metric
func RecordTrade(symbol, side string, amount float64) {
	tradesTotal.WithLabelValues(symbol, side).Inc()
	tradeAmount.WithLabelValues(symbol).Observe(amount)
}

// UpdatePrice updates the current price metric
func UpdatePrice(symbol string, price float64) {
	currentPrice.WithLabelValues(symbol).Set(price)
}

// UpdateStrategyConfidence updates the strategy confidence metric
func UpdateStrategyConfidence(strategy string, confidence float64) {
	strategyConfidence.WithLabelValues(strategy).Set(confidence)
}

// RecordError records an error metric
func RecordError(errorType string) {
	errorsTotal.WithLabelValues(errorType).Inc()
}
