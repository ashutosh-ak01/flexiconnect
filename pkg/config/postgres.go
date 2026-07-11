package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// PostgresRegistry implements ConfigRegistry storing API configs in PostgreSQL
type PostgresRegistry struct {
	db *sql.DB
}

// NewPostgresRegistry creates a new PostgresRegistry
func NewPostgresRegistry(db *sql.DB) *PostgresRegistry {
	return &PostgresRegistry{db: db}
}

// Get queries PostgreSQL for the specified API configuration. If version is empty, retrieves the active one.
func (r *PostgresRegistry) Get(ctx context.Context, apiName string, version string) (*APIConfig, error) {
	var query string
	var row *sql.Row

	if version != "" {
		query = `SELECT id, api_name, version, is_active, config FROM api_configs WHERE api_name = $1 AND version = $2`
		row = r.db.QueryRowContext(ctx, query, apiName, version)
	} else {
		query = `SELECT id, api_name, version, is_active, config FROM api_configs WHERE api_name = $1 AND is_active = true`
		row = r.db.QueryRowContext(ctx, query, apiName)
	}

	var cfg APIConfig
	var configJSON []byte
	err := row.Scan(&cfg.ID, &cfg.APIName, &cfg.Version, &cfg.IsActive, &configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuration not found for api: %s (version: %s)", apiName, version)
		}
		return nil, fmt.Errorf("failed to query database config: %w", err)
	}

	if err := json.Unmarshal(configJSON, &cfg.Config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration JSON: %w", err)
	}

	return &cfg, nil
}

// Register stores the config in database. If active, it deactivates all other versions of the same API first.
func (r *PostgresRegistry) Register(ctx context.Context, config *APIConfig) error {
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin registry transaction: %w", err)
	}
	defer tx.Rollback()

	if config.IsActive {
		// Reset active versions
		_, err = tx.ExecContext(ctx, `UPDATE api_configs SET is_active = false WHERE api_name = $1`, config.APIName)
		if err != nil {
			return fmt.Errorf("failed to reset active configurations: %w", err)
		}
	}

	query := `
		INSERT INTO api_configs (api_name, version, is_active, config)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (api_name, version)
		DO UPDATE SET is_active = EXCLUDED.is_active, config = EXCLUDED.config, updated_at = CURRENT_TIMESTAMP
	`
	_, err = tx.ExecContext(ctx, query, config.APIName, config.Version, config.IsActive, configJSON)
	if err != nil {
		return fmt.Errorf("failed to insert configuration: %w", err)
	}

	return tx.Commit()
}

// Reload is a no-op for postgres registry direct queries
func (r *PostgresRegistry) Reload(ctx context.Context) error {
	return nil
}
