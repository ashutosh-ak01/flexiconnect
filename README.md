# FlexiConnect

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**FlexiConnect** is a flexible, configuration-driven API Gateway & Integration Engine. Integrate, secure, and monitor any external HTTP API simply by defining a configuration, and expose it internally via a clean, unified REST interface.

---

## ✨ Features

- ⚙️ **Standalone API Gateway:** Deploy FlexiConnect as a daemon or Docker container to centralize all 3rd-party vendor integrations.
- 🔄 **Config-Driven:** Add new API integrations dynamically without deploying new code.
- 🔄 **Payload Transformations:** Dynamically map request payloads using Go templates, and extract/reshape response bodies using JSONPath.
- 🔑 **Centralized Secrets:** Resolve auth credentials securely from environment variables or Secrets Managers. Your internal microservices never need to know the vendor API keys.
- ⚡ **Global Circuit Breakers:** Protect your systems from cascade failures. If a vendor goes down, FlexiConnect trips the breaker for all incoming traffic automatically.
- 🔒 **PII Masking & Tracking:** Asynchronously audit request/response metadata with automatic masking of sensitive headers and JSON keys.

---

## 🏗️ Architecture

```text
  Internal Microservices 
       │ (POST /v1/execute)
       ▼
┌────────────────────────────────────────────────────────────┐
│ FlexiConnect API Gateway (Daemon)                          │
│                                                            │
│  1. HTTP Server     ──► Parses JSON payload                │
│  2. Resolve Config  ──► In-Memory / Postgres               │
│  3. Fetch Secrets   ──► Environment / AWS Secrets          │
│  4. Transform Req   ──► Go templates (JSON/Headers)        │
│  5. Circuit Breaker ──► Global gobreaker state             │
│  6. Execute HTTP    ──► External Vendor Endpoint           │
│  7. Transform Resp  ──► JSONPath Extraction                │
│  8. Audit Track     ──► Async DB Logger (PII Masked)       │
└────────────────────────────────────────────────────────────┘
```

---

## 🚀 Quick Start

### 1. Installation

You can download the pre-compiled binaries from the GitHub Releases page, or run it via Docker:

```bash
docker pull ghcr.io/ashutosh-ak01/flexiconnect:latest
docker run -p 8080:8080 ghcr.io/ashutosh-ak01/flexiconnect:latest
```

Or build from source:

```bash
git clone https://github.com/ashutosh-ak01/flexiconnect.git
cd flexiconnect
make build
./bin/flexiconnect
```

### 2. Executing an Integration

Once the gateway is running and a configuration is registered in the engine, your internal services can trigger an integration by sending a single POST request.

**Request:**
```bash
curl -X POST http://localhost:8080/v1/execute \
     -H "Content-Type: application/json" \
     -d '{
       "service": "stripe",
       "version": "v1",
       "action": "create_payment",
       "payload": {
         "amount": 2000,
         "currency": "usd"
       }
     }'
```

**Response:**
FlexiConnect will resolve the Stripe API keys, transform the payload, execute the request, run the success assertions, and extract just the fields you care about:
```json
{
  "transaction_id": "ch_1J2Y3Z4",
  "status": "succeeded"
}
```

---

## 🛠️ Advanced: Go Library Usage

While FlexiConnect shines as a standalone API gateway, the core `Engine` is completely decoupled from the HTTP server and can be imported natively into your Go applications.

```bash
go get github.com/ashutosh-ak01/flexiconnect/pkg/integration
```

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.
