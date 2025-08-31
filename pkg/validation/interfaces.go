package validation

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// Package validation provides walk-forward validation for trading strategies

// WalkForwardValidator defines the interface for walk-forward validation
type WalkForwardValidator interface {
	Validate(config interface{}, data []types.OHLCV, wfConfig WalkForwardConfig) (*WalkForwardSummary, error)
	SetOptimizer(optimizer func(config interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error))
}

// DataSplitter defines the interface for splitting data into train/test sets
type DataSplitter interface {
	SplitByRatio(data []types.OHLCV, ratio float64) ([]types.OHLCV, []types.OHLCV)
	CreateRollingFolds(data []types.OHLCV, trainDays, testDays, rollDays int) []WalkForwardFold
}

// WalkForwardConfig holds the configuration for walk-forward validation
type WalkForwardConfig struct {
	Enable     bool
	Rolling    bool
	SplitRatio float64
	TrainDays  int
	TestDays   int
	RollDays   int
}

// WalkForwardFold represents a single fold in walk-forward validation
type WalkForwardFold struct {
	Train      []types.OHLCV
	Test       []types.OHLCV
	TrainStart time.Time
	TrainEnd   time.Time
	TestStart  time.Time
	TestEnd    time.Time
}

// WalkForwardResults holds the results for a single fold
type WalkForwardResults struct {
	TrainResults *backtest.BacktestResults
	TestResults  *backtest.BacktestResults
	BestConfig   interface{}
	Fold         int
}

// WalkForwardSummary holds the summary of all walk-forward validation results
type WalkForwardSummary struct {
	Results              []WalkForwardResults
	AverageTrainReturn   float64
	AverageTestReturn    float64
	AverageTrainDrawdown float64
	AverageTestDrawdown  float64
	ReturnDegradation    float64
	IsRobust             bool
	OverfittingRisk      string
}
