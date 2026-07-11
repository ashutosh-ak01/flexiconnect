package secret

import (
	"context"
	"fmt"
	"os"
)

type EnvSecretProvider struct{}

// NewEnvSecretProvider creates a new instance of EnvSecretProvider
func NewEnvSecretProvider() *EnvSecretProvider {
	return &EnvSecretProvider{}
}

// GetSecret retrieves a secret from standard environment variables
func (p *EnvSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	val, exists := os.LookupEnv(key)
	if !exists {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return val, nil
}
