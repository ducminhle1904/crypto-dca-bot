package config

import (
	"os"
	"strconv"
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

		Notifications: struct {
			TelegramToken  string
			TelegramChatID string
			SlackWebhook   string
		}{
			TelegramToken:  getEnv("TELEGRAM_TOKEN", ""),
			TelegramChatID: getEnv("TELEGRAM_CHAT_ID", ""),
			SlackWebhook:   getEnv("SLACK_WEBHOOK", ""),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	return defaultVal
}
