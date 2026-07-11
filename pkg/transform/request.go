package transform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"

	"github.com/ashutosh-ak01/flexiconnect/pkg/secret"
)

// RequestTransformer handles rendering text templates with input parameters and dynamic secrets resolution
type RequestTransformer struct {
	secretResolver secret.SecretProvider
}

// NewRequestTransformer creates a new instance of RequestTransformer
func NewRequestTransformer(sp secret.SecretProvider) *RequestTransformer {
	return &RequestTransformer{
		secretResolver: sp,
	}
}

// Transform compiles and executes a Go template string with dynamic inputs and secret resolving functions.
func (rt *RequestTransformer) Transform(ctx context.Context, templateStr string, input map[string]interface{}) (string, error) {
	if templateStr == "" {
		return "", nil
	}

	// Setup custom functions for Go template interpolation (e.g. {{ secret "KEY" }} or {{ env "KEY" }})
	funcMap := template.FuncMap{
		"secret": func(key string) (string, error) {
			if rt.secretResolver == nil {
				return "", fmt.Errorf("secret provider is not configured")
			}
			return rt.secretResolver.GetSecret(ctx, key)
		},
		"env": func(key string) string {
			return os.Getenv(key)
		},
	}

	t, err := template.New("request_template").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse request template: %w", err)
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"input": input,
	}

	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute request template: %w", err)
	}

	return buf.String(), nil
}
