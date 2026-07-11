package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ashutosh-ak01/flexiconnect/pkg/config"
	"github.com/ashutosh-ak01/flexiconnect/pkg/integration"
	"github.com/ashutosh-ak01/flexiconnect/pkg/secret"
	"github.com/ashutosh-ak01/flexiconnect/pkg/track"
)

func main() {
	// Setup modern structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()

	// Set dynamic secret token environment variable to emulate secrets manager resolution
	os.Setenv("HTTPBIN_BEARER_TOKEN", "super-secret-oauth-token-12345")

	fmt.Println("🚀 Initializing FlexiConnect Example...")

	// 1. Initialize In-Memory Configuration Registry
	registry := config.NewInMemoryRegistry()

	// 2. Read config.json
	configPath := filepath.Join("examples", "config.json")
	configFileBytes, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read example config.json: %v", err)
	}

	var apiCfg config.APIConfig
	if err := json.Unmarshal(configFileBytes, &apiCfg); err != nil {
		log.Fatalf("Failed to parse config.json: %v", err)
	}

	// Register configuration
	if err := registry.Register(ctx, &apiCfg); err != nil {
		log.Fatalf("Failed to register config: %v", err)
	}
	fmt.Println("✅ API Configuration 'httpbin' version 'v1' registered.")

	// 3. Setup Env Secrets Provider & Log Audit Tracker
	secretsProvider := secret.NewEnvSecretProvider()
	auditTracker := track.NewLogTracker()

	// 4. Initialize FlexiConnect Engine
	engine := integration.NewEngine(registry, secretsProvider, auditTracker)

	// 5. Execute Request
	inputPayload := map[string]interface{}{
		"user": "developer_john",
		"pass": "super_secret_user_password_99",
	}

	fmt.Println("\n📡 Executing HTTP request 'post_test' (posting to httpbin.org)...")
	result, err := engine.ExecuteRequest(ctx, "httpbin", "v1", "post_test", inputPayload)
	if err != nil {
		log.Fatalf("❌ Execution failed: %v", err)
	}

	fmt.Println("\n🎉 Execution completed successfully!")
	fmt.Println("-------------------- TRANSFORMED CLIENT RESPONSE --------------------")
	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(prettyJSON))
	fmt.Println("---------------------------------------------------------------------")

	// Allow a moment for the async audit logger goroutine to write to stdout
	time.Sleep(500 * time.Millisecond)
	fmt.Println("\n👋 Finished example execution.")
}
