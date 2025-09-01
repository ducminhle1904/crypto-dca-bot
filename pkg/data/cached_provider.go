package data

import (
	"log"
	"path/filepath"
	"sync"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// MemoryCache implements DataCache using in-memory storage
type MemoryCache struct {
	cache map[string][]types.OHLCV
	mutex sync.RWMutex
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		cache: make(map[string][]types.OHLCV),
	}
}

// Get retrieves data from cache if available
func (c *MemoryCache) Get(key string) ([]types.OHLCV, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	data, exists := c.cache[key]
	if exists {
		// Return a copy to prevent external modifications
		result := make([]types.OHLCV, len(data))
		copy(result, data)
		return result, true
	}
	
	return nil, false
}

// Set stores data in cache
func (c *MemoryCache) Set(key string, data []types.OHLCV) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Store a copy to prevent external modifications
	cached := make([]types.OHLCV, len(data))
	copy(cached, data)
	c.cache[key] = cached
}

// Clear removes all cached data
func (c *MemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache = make(map[string][]types.OHLCV)
}

// Size returns the number of cached entries
func (c *MemoryCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return len(c.cache)
}

// CachedProvider wraps another DataProvider with caching functionality
type CachedProvider struct {
	provider DataProvider
	cache    DataCache
}

// NewCachedProvider creates a new cached data provider
func NewCachedProvider(provider DataProvider) *CachedProvider {
	return &CachedProvider{
		provider: provider,
		cache:    NewMemoryCache(),
	}
}

// NewCachedProviderWithCache creates a new cached data provider with custom cache
func NewCachedProviderWithCache(provider DataProvider, cache DataCache) *CachedProvider {
	return &CachedProvider{
		provider: provider,
		cache:    cache,
	}
}

// GetName returns the name of the underlying provider with cache indication
func (p *CachedProvider) GetName() string {
	return "Cached " + p.provider.GetName()
}

// LoadData loads data with caching to improve performance
func (p *CachedProvider) LoadData(source string) ([]types.OHLCV, error) {
	// Check cache first
	if cachedData, exists := p.cache.Get(source); exists {
		return cachedData, nil
	}

	// Load data if not in cache
	log.Printf("🔄 Loading historical data from %s", filepath.Base(source))
	data, err := p.provider.LoadData(source)
	if err != nil {
		log.Printf("❌ Failed to load data from %s: %v", filepath.Base(source), err)
		return nil, err
	}

	// Store in cache
	p.cache.Set(source, data)
	
	log.Printf("✅ Loaded and cached data from %s (%d records)", filepath.Base(source), len(data))
	return data, nil
}

// ValidateData validates data using the underlying provider
func (p *CachedProvider) ValidateData(data []types.OHLCV) error {
	return p.provider.ValidateData(data)
}

// GetCache returns the underlying cache for external management
func (p *CachedProvider) GetCache() DataCache {
	return p.cache
}

// ClearCache clears all cached data
func (p *CachedProvider) ClearCache() {
	p.cache.Clear()
}

// GetCacheSize returns the number of cached entries
func (p *CachedProvider) GetCacheSize() int {
	return p.cache.Size()
}
