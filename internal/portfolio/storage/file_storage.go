package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/portfolio"
)

// FileStorage implements portfolio.StateManager for file-based persistence
type FileStorage struct {
	mu       sync.RWMutex
	filePath string
	lockFile string
	isLocked bool
}

// NewFileStorage creates a new file-based state manager
func NewFileStorage(filePath string) portfolio.StateManager {
	if filePath == "" {
		filePath = "portfolio_state.json"
	}
	
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}
	
	return &FileStorage{
		filePath: filePath,
		lockFile: filePath + ".lock",
		isLocked: false,
	}
}

// Save saves the portfolio state to file
func (f *FileStorage) Save(state *portfolio.PortfolioState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if state == nil {
		return fmt.Errorf("cannot save nil state")
	}
	
	// Update timestamp
	state.LastUpdated = time.Now()
	
	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal portfolio state: %w", err)
	}
	
	// Write to temporary file first
	tempFile := f.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}
	
	// Atomic rename to ensure consistency
	if err := os.Rename(tempFile, f.filePath); err != nil {
		// Clean up temp file on failure
		os.Remove(tempFile)
		return fmt.Errorf("failed to commit state file: %w", err)
	}
	
	return nil
}

// Load loads the portfolio state from file
func (f *FileStorage) Load() (*portfolio.PortfolioState, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	// Check if file exists
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("portfolio state file does not exist: %s", f.filePath)
	}
	
	// Read file
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read portfolio state file: %w", err)
	}
	
	// Unmarshal JSON
	var state portfolio.PortfolioState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal portfolio state: %w", err)
	}
	
	// Validate loaded state
	if err := f.validateState(&state); err != nil {
		return nil, fmt.Errorf("invalid portfolio state: %w", err)
	}
	
	return &state, nil
}

// Lock creates a lock file to prevent concurrent access
func (f *FileStorage) Lock() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.isLocked {
		return fmt.Errorf("storage is already locked")
	}
	
	// Check if lock file exists
	if _, err := os.Stat(f.lockFile); err == nil {
		// Lock file exists, check if it's stale
		if err := f.checkStaleLock(); err != nil {
			return err
		}
	}
	
	// Create lock file with timestamp and process info
	lockInfo := map[string]interface{}{
		"timestamp": time.Now(),
		"pid":       os.Getpid(),
		"hostname":  getHostname(),
	}
	
	lockData, err := json.Marshal(lockInfo)
	if err != nil {
		return fmt.Errorf("failed to create lock data: %w", err)
	}
	
	if err := os.WriteFile(f.lockFile, lockData, 0644); err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	
	f.isLocked = true
	return nil
}

// Unlock removes the lock file
func (f *FileStorage) Unlock() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if !f.isLocked {
		return nil // Already unlocked
	}
	
	// Remove lock file
	if err := os.Remove(f.lockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}
	
	f.isLocked = false
	return nil
}

// IsLocked returns true if the storage is currently locked
func (f *FileStorage) IsLocked() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return f.isLocked
}

// BackupState creates a backup of the current state file
func (f *FileStorage) BackupState() error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	// Check if state file exists
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		return fmt.Errorf("no state file to backup")
	}
	
	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.backup_%s", f.filePath, timestamp)
	
	// Read original file
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return fmt.Errorf("failed to read state file for backup: %w", err)
	}
	
	// Write backup
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	
	fmt.Printf("✅ Portfolio state backed up to: %s\n", backupPath)
	return nil
}

// RestoreFromBackup restores state from a backup file
func (f *FileStorage) RestoreFromBackup(backupPath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}
	
	// Read backup file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}
	
	// Validate backup data
	var state portfolio.PortfolioState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("invalid backup file format: %w", err)
	}
	
	if err := f.validateState(&state); err != nil {
		return fmt.Errorf("invalid backup state: %w", err)
	}
	
	// Create backup of current state before restore
	if _, err := os.Stat(f.filePath); err == nil {
		if err := f.BackupState(); err != nil {
			fmt.Printf("⚠️ Warning: Could not backup current state before restore: %v\n", err)
		}
	}
	
	// Restore data
	if err := os.WriteFile(f.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}
	
	fmt.Printf("✅ Portfolio state restored from: %s\n", backupPath)
	return nil
}

// GetStateFileInfo returns information about the state file
func (f *FileStorage) GetStateFileInfo() (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	info := make(map[string]interface{})
	
	// Check if file exists
	fileInfo, err := os.Stat(f.filePath)
	if os.IsNotExist(err) {
		info["exists"] = false
		return info, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	info["exists"] = true
	info["size"] = fileInfo.Size()
	info["modified"] = fileInfo.ModTime()
	info["path"] = f.filePath
	
	// Check if locked
	if _, err := os.Stat(f.lockFile); err == nil {
		info["locked"] = true
		
		// Try to read lock info
		lockData, err := os.ReadFile(f.lockFile)
		if err == nil {
			var lockInfo map[string]interface{}
			if json.Unmarshal(lockData, &lockInfo) == nil {
				info["lock_info"] = lockInfo
			}
		}
	} else {
		info["locked"] = false
	}
	
	return info, nil
}

// Private helper methods

func (f *FileStorage) validateState(state *portfolio.PortfolioState) error {
	if state.TotalBalance <= 0 {
		return fmt.Errorf("invalid total balance: %.2f", state.TotalBalance)
	}
	
	if state.Allocations == nil {
		state.Allocations = make(map[string]*portfolio.BotAllocation)
	}
	
	// Validate each bot allocation
	for botID, allocation := range state.Allocations {
		if allocation == nil {
			return fmt.Errorf("nil allocation for bot: %s", botID)
		}
		
		if allocation.BotID != botID {
			return fmt.Errorf("bot ID mismatch: expected %s, got %s", botID, allocation.BotID)
		}
		
		if allocation.AllocatedBalance < 0 {
			return fmt.Errorf("negative allocated balance for bot %s: %.2f", botID, allocation.AllocatedBalance)
		}
		
		if allocation.Leverage <= 0 {
			return fmt.Errorf("invalid leverage for bot %s: %.2f", botID, allocation.Leverage)
		}
	}
	
	return nil
}

func (f *FileStorage) checkStaleLock() error {
	lockData, err := os.ReadFile(f.lockFile)
	if err != nil {
		return fmt.Errorf("failed to read lock file: %w", err)
	}
	
	var lockInfo map[string]interface{}
	if err := json.Unmarshal(lockData, &lockInfo); err != nil {
		// Invalid lock file, remove it
		os.Remove(f.lockFile)
		return nil
	}
	
	// Check lock age
	if timestampStr, ok := lockInfo["timestamp"].(string); ok {
		if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			age := time.Since(timestamp)
			if age > 5*time.Minute { // Lock older than 5 minutes is considered stale
				fmt.Printf("⚠️ Removing stale lock file (age: %v)\n", age)
				os.Remove(f.lockFile)
				return nil
			}
		}
	}
	
	return fmt.Errorf("portfolio storage is locked by another process")
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
