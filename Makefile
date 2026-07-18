.PHONY: all build test test-cover fmt lint tidy run clean help

# Binary name
BINARY_NAME=flexiconnect

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

all: help

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build the daemon/cli binary"
	@echo "  test        Run all unit tests"
	@echo "  test-race   Run all unit tests with race detector enabled"
	@echo "  test-cover  Run all unit tests with coverage reporting"
	@echo "  fmt         Run go fmt against codebase"
	@echo "  lint        Run golangci-lint (falls back to go vet if not installed)"
	@echo "  tidy        Run go mod tidy to lock dependencies"
	@echo "  run         Run the example main program"
	@echo "  clean       Clean built binaries"

# Build Flags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

build:
	mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/flexiconnect/main.go

test:
	$(GOTEST) -v ./...

test-race:
	$(GOTEST) -race -v ./...

test-cover:
	$(GOTEST) -coverprofile=coverage.out -v ./...
	$(GOCMD) tool cover -html=coverage.out

fmt:
	$(GOFMT) ./...

lint:
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, falling back to go vet"; \
		$(GOVET) ./...; \
	fi

tidy:
	$(GOMOD) tidy

run:
	$(GOCMD) run examples/main.go

clean:
	$(GOCLEAN)
	rm -rf bin
	rm -f coverage.out
