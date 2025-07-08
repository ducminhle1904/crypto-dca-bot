package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TotalTrades = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dca_bot_trades_total",
			Help: "Total number of trades executed",
		},
		[]string{"symbol", "side", "strategy"},
	)

	TradePnL = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dca_bot_trade_pnl",
			Help:    "Profit and loss per trade",
			Buckets: prometheus.LinearBuckets(-1000, 100, 20),
		},
		[]string{"symbol"},
	)

	PortfolioValue = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dca_bot_portfolio_value_usd",
			Help: "Current portfolio value in USD",
		},
		[]string{"symbol"},
	)

	IndicatorValues = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dca_bot_indicator_value",
			Help: "Current technical indicator values",
		},
		[]string{"indicator", "symbol"},
	)

	ExchangeLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dca_bot_exchange_latency_seconds",
			Help:    "Exchange API response latency",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"exchange", "endpoint"},
	)
)

func RecordTrade(symbol, side, strategy string, pnl float64) {
	TotalTrades.WithLabelValues(symbol, side, strategy).Inc()
	TradePnL.WithLabelValues(symbol).Observe(pnl)
}
