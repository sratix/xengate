# Makefile for Go Web Application with Templ and Tailwind

# Shell configuration
SHELL := /usr/bin/env bash

#------------------------------------------------------------------------------
# OS-specific configurations
#------------------------------------------------------------------------------
ifeq ($(OS),Windows_NT)
    EXECUTABLE_EXTENSION := .exe
    PATH_SEPARATOR      := ;
    RM_CMD             := rd /s /q
    MKDIR_CMD          := mkdir
    SHELL_SYMBOL       := =>
else
    EXECUTABLE_EXTENSION :=
    PATH_SEPARATOR      := :
    RM_CMD             := rm -rf
    MKDIR_CMD          := mkdir -p
    SHELL_SYMBOL       := \033[34;1m▶\033[0m
endif

#------------------------------------------------------------------------------
# Project variables
#------------------------------------------------------------------------------
NAME          := xengate
MODULE        := $(shell go list -m)
VERSION       := $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || cat .version 2> /dev/null || echo v0)
DATE          := $(shell date +%FT%T%z)
PKGS          := $(or $(PKG),$(shell go list ./...))

#------------------------------------------------------------------------------
# Build configurations
#------------------------------------------------------------------------------
GO            := go
BIN           := bin
TIMEOUT       := 15
V             := 0
Q             := $(if $(filter 1,$V),,@)
M             := $(shell printf "$(SHELL_SYMBOL)")

# Build flags
LDFLAGS       := -X $(MODULE)/cmd.Version=$(VERSION) -X $(MODULE)/cmd.BuildDate=$(DATE)
GOFLAGS       := -tags "release jsoniter"

# Conditional build settings
ifeq ($(OS),Windows_NT)
    GOBUILD   := $(GO) build
else
    GOBUILD   := CGO_ENABLED=0 GOOS=$$(go env GOOS) GOARCH=$$(go env GOARCH) $(GO) build
endif

#------------------------------------------------------------------------------
# Tool binaries
#------------------------------------------------------------------------------
TOOLS_DIR     := $(BIN)
GOIMPORTS     := $(TOOLS_DIR)/goimports$(EXECUTABLE_EXTENSION)
REVIVE        := $(TOOLS_DIR)/revive$(EXECUTABLE_EXTENSION)
GOCOV         := $(TOOLS_DIR)/gocov$(EXECUTABLE_EXTENSION)
GOCOV_XML     := $(TOOLS_DIR)/gocov-xml$(EXECUTABLE_EXTENSION)
GOTESTSUM     := $(TOOLS_DIR)/gotestsum$(EXECUTABLE_EXTENSION)

#------------------------------------------------------------------------------
# Main targets
#------------------------------------------------------------------------------
.PHONY: all build clean tools test lint fmt

all: tools fmt lint test build ## Build everything

#------------------------------------------------------------------------------
# Build targets
#------------------------------------------------------------------------------
build:  ## Build the application
	$(info $(M) Building executable...)
	$(Q) $(GOBUILD) $(GOFLAGS) \
		-trimpath -ldflags '$(LDFLAGS)' \
		-o $(BIN)/$(NAME)$(EXECUTABLE_EXTENSION) main.go

#------------------------------------------------------------------------------
# Server targets
#------------------------------------------------------------------------------
serve: build ## Start Proxy server
	$(info $(M) Starting Proxy server...)
	$(Q) $(BIN)/$(NAME)$(EXECUTABLE_EXTENSION) serve --config=$$(pwd)/cfg/config.yml

#------------------------------------------------------------------------------
# Development tools
#------------------------------------------------------------------------------
tools: $(GOIMPORTS) $(REVIVE) $(GOCOV) $(GOCOV_XML) $(GOTESTSUM) ## Install all development tools

fmt: $(GOIMPORTS) ## Format code
	$(info $(M) Formatting code...)
	$(Q) $(GOIMPORTS) -local $(MODULE) -w \
		$(shell $(GO) list -f \
			'{{$$d := .Dir}}{{range $$f := .GoFiles}}{{printf "%s/%s\n" $$d $$f}}{{end}}' \
			$(PKGS))

lint: $(REVIVE) ## Run linter
	$(info $(M) Running linter...)
	$(Q) $(REVIVE) -formatter friendly ./...

test: $(GOTESTSUM) ## Run tests
	$(info $(M) Running tests...)
	$(Q) $(GOTESTSUM) -- -race -cover ./...

#------------------------------------------------------------------------------
# Cleanup and utility targets
#------------------------------------------------------------------------------
clean: ## Clean up build artifacts
	$(info $(M) Cleaning...)
	$(Q) $(RM_CMD) $(BIN)

version: ## Show version information
	@echo $(VERSION)

# Directory creation
$(BIN):
	$(Q) $(MKDIR_CMD) $(BIN)

#------------------------------------------------------------------------------
# Tool installation targets
#------------------------------------------------------------------------------
$(GOIMPORTS): | $(BIN)
	$(info $(M) Installing goimports...)
	$(Q) GOBIN=$(abspath $(BIN)) $(GO) install golang.org/x/tools/cmd/goimports@latest

$(REVIVE): | $(BIN)
	$(info $(M) Installing revive...)
	$(Q) GOBIN=$(abspath $(BIN)) $(GO) install github.com/mgechev/revive@latest

$(GOCOV): | $(BIN)
	$(info $(M) Installing gocov...)
	$(Q) GOBIN=$(abspath $(BIN)) $(GO) install github.com/axw/gocov/gocov@latest

$(GOCOV_XML): | $(BIN)
	$(info $(M) Installing gocov-xml...)
	$(Q) GOBIN=$(abspath $(BIN)) $(GO) install github.com/AlekSi/gocov-xml@latest

$(GOTESTSUM): | $(BIN)
	$(info $(M) Installing gotestsum...)
	$(Q) GOBIN=$(abspath $(BIN)) $(GO) install gotest.tools/gotestsum@latest

#------------------------------------------------------------------------------
# Help
#------------------------------------------------------------------------------
help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help