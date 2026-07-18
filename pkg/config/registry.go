package config

import "context"

// ConfigRegistry defines actions for loading and managing dynamic API configurations
type ConfigRegistry interface {
	// Get retrieves the config for the specified apiName and version.
	// If version is empty (""), it will retrieve the active version (where is_active = true).
	Get(ctx context.Context, apiName string, version string) (*APIConfig, error)

	// Register inserts or registers a new API configuration version.
	Register(ctx context.Context, config *APIConfig) error

	// Reload triggers a refresh of all configuration caches if caching is utilized
	Reload(ctx context.Context) error
}
