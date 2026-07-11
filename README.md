# FlexiConnect

[![Go Reference](https://pkg.go.dev/badge/github.com/ashutosh-ak01/flexiconnect.svg)](https://pkg.go.dev/github.com/ashutosh-ak01/flexiconnect)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**FlexiConnect** is a flexible, configuration-driven API integration engine in Go. Integrate, secure, and monitor any external HTTP API simply by defining a JSON/YAML configuration.

---

## ✨ Features

- ⚙️ **Config-Driven:** Add new API integrations dynamically without deploying new code.
- 🔄 **Payload Transformations:** Dynamically map request payloads using Go templates, and extract/reshape response bodies using JSONPath.
- 🔑 **Pluggable Secrets:** Resolve auth credentials at runtime from environment variables, files, or AWS Secrets Manager.
- ⚡ **Local Circuit Breakers:** Protect your application from cascade failures using per-API/endpoint local circuit breakers.
- 🔒 **PII Masking & Tracking:** Asynchronously audit request/response metadata and payloads with automatic masking of sensitive headers and JSON keys.
- ⏱️ **Latency Metrics:** Automatically track request processing durations.

---

## 🏗️ Architecture

```text
  Client App 
      │
      ▼
┌────────────────────────────────────────────────────────┐
│ FlexiConnect Engine                                    │
│                                                        │
│  1. Resolve Config  ──► In-Memory / Postgres           │
│  2. Fetch Secrets   ──► Environment / AWS Secrets      │
│  3. Transform Req   ──► Go templates (JSON/Headers)    │
│  4. Rate Limit      ──► Local Token Bucket             │
│  5. Circuit Breaker ──► Local gobreaker                │
│  6. Execute HTTP    ──► External HTTP Endpoint         │
│  7. Transform Resp  ──► JSONPath Extraction            │
│  8. Audit Track     ──► Async DB Logger (PII Masked)   │
└────────────────────────────────────────────────────────┘
```

---

## 🚀 Quick Start

### Installation

```bash
go get github.com/ashutosh-ak01/flexiconnect
```

### Simple Usage Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ashutosh-ak01/flexiconnect/pkg/config"
	"github.com/ashutosh-ak01/flexiconnect/pkg/integration"
	"github.com/ashutosh-ak01/flexiconnect/pkg/secret"
)

func main() {
	ctx := context.Background()

	// 1. Initialize configuration registry (in-memory)
	registry := config.NewInMemoryRegistry()

	// 2. Load API configuration
	apiCfg := &config.APIConfig{
		APIName:  "stripe",
		Version:  "v1",
		IsActive: true,
		Config: config.ConfigDetail{
			BaseURL:   "https://api.stripe.com",
			TimeoutMs: 5000,
			Endpoints: []config.EndpointConfig{
				{
					Name:        "create_payment",
					Method:      "POST",
					Path:        "/v1/charges",
					BodyTemplate: `{"amount": {{.input.amount}}, "currency": "{{.input.currency}}"}`,
					ResponseTransformation: map[string]string{
						"transaction_id": "$.id",
						"status":         "$.status",
					},
				},
			},
		},
	}
	registry.Register(ctx, apiCfg)

	// 3. Setup Secret Resolver & Engine
	secretProvider := secret.NewEnvSecretProvider()
	engine := integration.NewEngine(registry, secretProvider, nil)

	// 4. Execute API request
	input := map[string]interface{}{
		"amount":   2000,
		"currency": "usd",
	}
	
	resp, err := engine.ExecuteRequest(ctx, "stripe", "", "create_payment", input)
	if err != nil {
		log.Fatalf("Error executing request: %v", err)
	}

	fmt.Printf("Response: %v\n", resp)
}
```

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.
