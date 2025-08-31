package reporting

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultPathManager implements path management functionality
type DefaultPathManager struct{}

// NewDefaultPathManager creates a new path manager
func NewDefaultPathManager() *DefaultPathManager {
	return &DefaultPathManager{}
}

// GetDefaultOutputDir returns default output directory - extracted from main.go defaultOutputDir
func (p *DefaultPathManager) GetDefaultOutputDir(symbol, interval string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	i := strings.ToLower(strings.TrimSpace(interval))
	if s == "" {
		s = "UNKNOWN"
	}
	if i == "" {
		i = "unknown"
	}
	
	return filepath.Join("results", fmt.Sprintf("%s_%s", s, i))
}

// EnsureDirectoryExists creates directory if it doesn't exist
func (p *DefaultPathManager) EnsureDirectoryExists(path string) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// Package-level convenience function
func DefaultOutputDir(symbol, interval string) string {
	manager := NewDefaultPathManager()
	return manager.GetDefaultOutputDir(symbol, interval)
}
