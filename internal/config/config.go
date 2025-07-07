package config

import (
	"os"
	"time"
)

type Config struct {
	Environment string
	LogLevel    string

	Exchange struct {
		Name    string
		APIKey  string
		Secret  string
		Testnet bool
	}

	Strategy struct {
		Symbol        string
		BaseAmount    float64
		MaxMultiplier float64
		Interval      time.Duration
	}

	Monitoring struct {
		PrometheusPort int
		HealthPort     int
	}

	Notifications struct {
		TelegramToken  string
		TelegramChatID string
		SlackWebhook   string
	}
}

func Load() *Config {
	return &Config{
		Environment: getEnv("ENV", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),

		Exchange: struct {
			Name    string
			APIKey  string
			Secret  string
			Testnet bool
		}{
			Name:    getEnv("EXCHANGE_NAME", "binance"),
			APIKey:  getEnv("EXCHANGE_API_KEY", ""),
			Secret:  getEnv("EXCHANGE_SECRET", ""),
			Testnet: getEnvBool("EXCHANGE_TESTNET", true),
		},

		Strategy: struct {
			Symbol        string
			BaseAmount    float64
			MaxMultiplier float64
			Interval      time.Duration
		}{
			Symbol:        getEnv("TRADING_SYMBOL", "BTCUSDT"),
			BaseAmount:    getEnvFloat("BASE_AMOUNT", 100.0),
			MaxMultiplier: getEnvFloat("MAX_MULTIPLIER", 3.0),
			Interval:      getEnvDuration("TRADING_INTERVAL", time.Hour),
		},

		Monitoring: struct {
			PrometheusPort int
			HealthPort     int
		}{
			PrometheusPort: getEnvInt("PROMETHEUS_PORT", 8080),
			HealthPort:     getEnvInt("HEALTH_PORT", 8081),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
