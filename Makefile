# terraform-provider-diffusion Makefile
#
# Portable across Windows (with cygwin/MSYS sh), Linux and macOS.

BINARY_NAME := terraform-provider-diffusion

# ---------------------------------------------------------------------------
# Target detection
# ---------------------------------------------------------------------------
GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
BINARY_EXT := .exe
else
BINARY_EXT :=
endif

BINARY := $(BINARY_NAME)$(BINARY_EXT)

# ---------------------------------------------------------------------------
# Version
# ---------------------------------------------------------------------------
DEVNULL := /dev/null
GIT_DESCRIBE := $(shell git describe --tags --always --dirty 2>$(DEVNULL))
GIT_VERSION  := $(patsubst v%,%,$(firstword $(subst -, ,$(GIT_DESCRIBE))))

ifeq ($(words $(subst ., ,$(GIT_VERSION))),3)
VERSION ?= $(GIT_VERSION)
else
VERSION ?= 0.0.0
endif

LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

# ---------------------------------------------------------------------------
# Local plugin directory (Terraform filesystem mirror layout)
# ---------------------------------------------------------------------------
PLUGIN_ROOT    := $(HOME)/.terraform.d/plugins
PLUGIN_DIR     := $(PLUGIN_ROOT)/registry.terraform.io/Polar-Team/diffusion/$(VERSION)/$(GOOS)_$(GOARCH)
INSTALL_BINARY := $(BINARY_NAME)_v$(VERSION)$(BINARY_EXT)

# ---------------------------------------------------------------------------
# Test directories — install provider binary into each for local testing
# ---------------------------------------------------------------------------
PROVIDER_REL_PATH := .terraform/providers/registry.terraform.io/polar-team/diffusion/$(VERSION)/$(GOOS)_$(GOARCH)

.PHONY: build test testacc install install-global clean fmt lint docs snapshot info

# ===========================================================================
# Targets
# ===========================================================================

info:
	@echo "host_goos=$(GOOS) goarch=$(GOARCH)"
	@echo "version=$(VERSION) binary=$(BINARY)"
	@echo "plugin_dir=$(PLUGIN_DIR)"
	@echo "install_binary=$(INSTALL_BINARY)"
	@echo "provider_rel_path=$(PROVIDER_REL_PATH)"

# Build the provider binary
build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BINARY) .
	@echo "Built $(BINARY) for $(GOOS)/$(GOARCH) version $(VERSION)"

# Run unit tests
test:
	go test -v -count=1 ./...

# Run acceptance tests (requires TF_ACC=1, terraform on PATH, diffusion on PATH)
testacc:
	TF_ACC=1 go test -v -timeout 30m ./internal/provider/ -run TestAcc

# Install binary into all test directories under tests/ and remove lock files
install: build
	@for dir in tests/*/; do \
		mkdir -p "$$dir$(PROVIDER_REL_PATH)"; \
		cp "$(BINARY)" "$$dir$(PROVIDER_REL_PATH)/$(INSTALL_BINARY)"; \
		rm -f "$${dir}.terraform.lock.hcl"; \
		echo "Installed to $$dir$(PROVIDER_REL_PATH)/$(INSTALL_BINARY)"; \
	done

# Install to the global Terraform plugin directory for dev testing
install-global: build
	@mkdir -p "$(PLUGIN_DIR)"
	@cp "$(BINARY)" "$(PLUGIN_DIR)/$(INSTALL_BINARY)"
	@echo "Installed $(INSTALL_BINARY) to $(PLUGIN_DIR)"

# Format code
fmt:
	go fmt ./...

# Lint
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf dist/

# Generate documentation (requires tfplugindocs)
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate

# Run GoReleaser locally (snapshot, no publish)
snapshot:
	goreleaser release --snapshot --clean
