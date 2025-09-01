package validation

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DefaultDataSplitter implements the DataSplitter interface
type DefaultDataSplitter struct{}

// NewDefaultDataSplitter creates a new default data splitter
func NewDefaultDataSplitter() *DefaultDataSplitter {
	return &DefaultDataSplitter{}
}

// SplitByRatio splits data into train/test by ratio - extracted from main.go
func (s *DefaultDataSplitter) SplitByRatio(data []types.OHLCV, ratio float64) ([]types.OHLCV, []types.OHLCV) {
	if ratio <= 0 || ratio >= 1 {
		return data, nil
	}
	
	n := int(float64(len(data)) * ratio)
	if n < 1 || n >= len(data) {
		return data, nil
	}
	
	return data[:n], data[n:]
}

// CreateRollingFolds creates rolling walk-forward folds - extracted from main.go
func (s *DefaultDataSplitter) CreateRollingFolds(data []types.OHLCV, trainDays, testDays, rollDays int) []WalkForwardFold {
	var folds []WalkForwardFold
	
	trainDur := time.Duration(trainDays) * 24 * time.Hour
	testDur := time.Duration(testDays) * 24 * time.Hour
	rollDur := time.Duration(rollDays) * 24 * time.Hour
	
	if len(data) < 100 {
		return folds // Need minimum data
	}
	
	start := 0
	for {
		// Find train window
		trainEndTs := data[start].Timestamp.Add(trainDur)
		trainEnd := start
		for trainEnd < len(data) && data[trainEnd].Timestamp.Before(trainEndTs) {
			trainEnd++
		}
		
		// Find test window
		testEndTs := trainEndTs.Add(testDur)
		testEnd := trainEnd
		for testEnd < len(data) && data[testEnd].Timestamp.Before(testEndTs) {
			testEnd++
		}
		
		// Check if we have enough data
		trainSize := trainEnd - start
		testSize := testEnd - trainEnd
		
		if trainSize < 50 || testSize < 10 {
			break // Not enough data for this fold
		}
		
		fold := WalkForwardFold{
			Train:      data[start:trainEnd],
			Test:       data[trainEnd:testEnd],
			TrainStart: data[start].Timestamp,
			TrainEnd:   data[trainEnd-1].Timestamp,
			TestStart:  data[trainEnd].Timestamp,
			TestEnd:    data[testEnd-1].Timestamp,
		}
		
		folds = append(folds, fold)
		
		// Roll forward
		nextStartTs := data[start].Timestamp.Add(rollDur)
		nextStart := start
		for nextStart < len(data) && data[nextStart].Timestamp.Before(nextStartTs) {
			nextStart++
		}
		
		if nextStart <= start {
			nextStart = start + 1
		}
		if nextStart >= len(data) {
			break
		}
		
		start = nextStart
	}
	
	return folds
}

// Package-level convenience functions

// SplitByRatio is a convenience function that uses the default splitter
func SplitByRatio(data []types.OHLCV, ratio float64) ([]types.OHLCV, []types.OHLCV) {
	splitter := NewDefaultDataSplitter()
	return splitter.SplitByRatio(data, ratio)
}

// CreateRollingFolds is a convenience function that uses the default splitter
func CreateRollingFolds(data []types.OHLCV, trainDays, testDays, rollDays int) []WalkForwardFold {
	splitter := NewDefaultDataSplitter()
	return splitter.CreateRollingFolds(data, trainDays, testDays, rollDays)
}
