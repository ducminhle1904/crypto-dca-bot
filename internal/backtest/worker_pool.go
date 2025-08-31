package backtest

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// WorkerPool manages parallel backtest execution
type WorkerPool struct {
	workerCount int
	jobQueue    chan BacktestJob
	resultQueue chan BacktestResult
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// BacktestJob represents a single backtest task
type BacktestJob struct {
	ID       string
	Config   BacktestConfig
	Data     []types.OHLCV
	Strategy strategy.Strategy
}

// BacktestResult represents the result of a backtest job
type BacktestResult struct {
	ID       string
	Results  *BacktestResults
	Config   BacktestConfig
	Duration time.Duration
	Error    error
}

// BacktestConfig represents backtest configuration
type BacktestConfig struct {
	InitialBalance float64
	Commission     float64
	WindowSize     int
	TPPercent      float64
	MinOrderQty    float64
	UseTPLevels    bool
	Symbol         string
	Interval       string
}

// NewWorkerPool creates a new worker pool for parallel backtesting
func NewWorkerPool(workerCount int, jobBufferSize int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workerCount: workerCount,
		jobQueue:    make(chan BacktestJob, jobBufferSize),
		resultQueue: make(chan BacktestResult, jobBufferSize),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop stops the worker pool gracefully
func (wp *WorkerPool) Stop() {
	close(wp.jobQueue)
	wp.wg.Wait()
	close(wp.resultQueue)
	wp.cancel()
}

// SubmitJob submits a backtest job to the pool
func (wp *WorkerPool) SubmitJob(job BacktestJob) error {
	select {
	case wp.jobQueue <- job:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

// GetResults returns the result channel for collecting completed jobs
func (wp *WorkerPool) GetResults() <-chan BacktestResult {
	return wp.resultQueue
}

// worker processes backtest jobs
func (wp *WorkerPool) worker(workerID int) {
	defer wp.wg.Done()

	for {
		select {
		case job, ok := <-wp.jobQueue:
			if !ok {
				return // Channel closed, worker should exit
			}

			result := wp.processJob(job)
			
			select {
			case wp.resultQueue <- result:
			case <-wp.ctx.Done():
				return
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// processJob processes a single backtest job
func (wp *WorkerPool) processJob(job BacktestJob) BacktestResult {
	startTime := time.Now()

	result := BacktestResult{
		ID:     job.ID,
		Config: job.Config,
	}

	// Create backtest engine
	engine := NewBacktestEngine(
		job.Config.InitialBalance,
		job.Config.Commission,
		job.Strategy,
		job.Config.TPPercent,
		job.Config.MinOrderQty,
		job.Config.UseTPLevels,
	)

	// Run backtest
	backtestResults := engine.Run(job.Data, job.Config.WindowSize)
	
	// Update metrics
	backtestResults.UpdateMetrics()

	result.Results = backtestResults
	result.Duration = time.Since(startTime)

	return result
}

// BatchProcessor handles batch processing of multiple backtests
type BatchProcessor struct {
	workerPool   *WorkerPool
	dataProvider DataProvider
	maxJobs      int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(workerCount, jobBufferSize, maxJobs int, dataProvider DataProvider) *BatchProcessor {
	return &BatchProcessor{
		workerPool:   NewWorkerPool(workerCount, jobBufferSize),
		dataProvider: dataProvider,
		maxJobs:      maxJobs,
	}
}

// ProcessBatch processes multiple backtest configurations in parallel
func (bp *BatchProcessor) ProcessBatch(configs []BacktestConfig, strategyFactory func(BacktestConfig) strategy.Strategy) ([]BacktestResult, error) {
	bp.workerPool.Start()
	defer bp.workerPool.Stop()

	// Submit jobs
	jobCount := 0
	for i, config := range configs {
		if jobCount >= bp.maxJobs {
			break
		}

		// Load data for this configuration
		params := map[string]interface{}{
			"file_path": getDataFilePath(config.Symbol, config.Interval),
		}
		
		data, err := bp.dataProvider.LoadData(config.Symbol, config.Interval, params)
		if err != nil {
			continue // Skip configurations with data loading errors
		}

		// Create strategy for this configuration
		strat := strategyFactory(config)

		job := BacktestJob{
			ID:       generateJobID(config, i),
			Config:   config,
			Data:     data,
			Strategy: strat,
		}

		if err := bp.workerPool.SubmitJob(job); err != nil {
			break // Worker pool is shutting down
		}
		jobCount++
	}

	// Collect results
	results := make([]BacktestResult, 0, jobCount)
	for i := 0; i < jobCount; i++ {
		result := <-bp.workerPool.GetResults()
		results = append(results, result)
	}

	return results, nil
}

// generateJobID generates a unique job ID
func generateJobID(config BacktestConfig, index int) string {
	return fmt.Sprintf("%s_%s_%d_%d", config.Symbol, config.Interval, index, time.Now().Unix())
}

// getDataFilePath constructs the data file path for a symbol and interval
func getDataFilePath(symbol, interval string) string {
	// This should match your actual data file structure
	return fmt.Sprintf("data/bybit/linear/%s/%s/candles.csv", symbol, interval)
}

// ProgressTracker tracks the progress of batch processing
type ProgressTracker struct {
	total     int
	completed int
	startTime time.Time
	mutex     sync.RWMutex
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total:     total,
		completed: 0,
		startTime: time.Now(),
	}
}

// Increment increments the completion count
func (pt *ProgressTracker) Increment() {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	pt.completed++
}

// GetProgress returns the current progress
func (pt *ProgressTracker) GetProgress() (int, int, float64, time.Duration) {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	elapsed := time.Since(pt.startTime)
	progress := float64(pt.completed) / float64(pt.total) * 100

	return pt.completed, pt.total, progress, elapsed
}

// EstimateTimeRemaining estimates the remaining time based on current progress
func (pt *ProgressTracker) EstimateTimeRemaining() time.Duration {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	if pt.completed == 0 {
		return 0
	}

	elapsed := time.Since(pt.startTime)
	avgTimePerItem := elapsed / time.Duration(pt.completed)
	remaining := pt.total - pt.completed

	return avgTimePerItem * time.Duration(remaining)
}
