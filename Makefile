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
	@echo "  test-cover  Run all unit tests with coverage reporting"
	@echo "  fmt         Run go fmt against codebase"
	@echo "  lint        Run go vet against codebase"
	@echo "  tidy        Run go mod tidy to lock dependencies"
	@echo "  run         Run the example main program"
	@echo "  clean       Clean built binaries"

build:
	mkdir -p bin
	$(GOBUILD) -o bin/$(BINARY_NAME) cmd/flexiconnect/main.go

test:
	$(GOTEST) -v ./...

test-cover:
	$(GOTEST) -coverprofile=coverage.out -v ./...
	$(GOCMD) tool cover -html=coverage.out

fmt:
	$(GOFMT) ./...

lint:
	$(GOVET) ./...

tidy:
	$(GOMOD) tidy

run:
	$(GOCMD) run examples/main.go

clean:
	$(GOCLEAN)
	rm -rf bin
	rm -f coverage.out
