package secret

import "context"

// SecretProvider defines operations to safely extract credential values (keys, tokens) from external engines
type SecretProvider interface {
	GetSecret(ctx context.Context, key string) (string, error)
}
