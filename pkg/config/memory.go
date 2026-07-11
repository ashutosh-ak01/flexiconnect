package config

import (
	"context"
	"fmt"
	"sync"
)

type InMemoryRegistry struct {
	mu      sync.RWMutex
	configs map[string]*APIConfig // key format: api_name:version
}

// NewInMemoryRegistry creates a new instance of InMemoryRegistry
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		configs: make(map[string]*APIConfig),
	}
}

// Get fetches the API config from in-memory map. If version is empty, returns the active version.
func (r *InMemoryRegistry) Get(ctx context.Context, apiName string, version string) (*APIConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If explicit version requested
	if version != "" {
		key := fmt.Sprintf("%s:%s", apiName, version)
		cfg, exists := r.configs[key]
		if !exists {
			return nil, fmt.Errorf("configuration not found for API %s version %s", apiName, version)
		}
		return cfg, nil
	}

	// Look for active version
	for _, cfg := range r.configs {
		if cfg.APIName == apiName && cfg.IsActive {
			return cfg, nil
		}
	}

	return nil, fmt.Errorf("active configuration not found for API %s", apiName)
}

// Register registers a configuration in-memory. If config is marked active, it deactivates other versions.
func (r *InMemoryRegistry) Register(ctx context.Context, config *APIConfig) error {
	if config.APIName == "" || config.Version == "" {
		return fmt.Errorf("api_name and version must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// If the new config is active, deactivate any other versions of this API
	if config.IsActive {
		for _, cfg := range r.configs {
			if cfg.APIName == config.APIName {
				cfg.IsActive = false
			}
		}
	}

	key := fmt.Sprintf("%s:%s", config.APIName, config.Version)
	r.configs[key] = config
	return nil
}

// Reload is a no-op for in-memory registry
func (r *InMemoryRegistry) Reload(ctx context.Context) error {
	return nil
}
