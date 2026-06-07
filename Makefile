# terraform-provider-diffusion Makefile

BINARY_NAME=terraform-provider-diffusion
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null | sed -E 's/^v?([0-9]+\.[0-9]+\.[0-9]+).*/\1/' || echo "0.0.0")
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

.PHONY: build test testacc install clean fmt lint

# Build the provider binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Run unit tests
test:
	go test -v -count=1 ./...

# Run acceptance tests (requires TF_ACC=1, terraform on PATH, diffusion on PATH)
testacc:
	TF_ACC=1 go test -v -timeout 30m ./internal/provider/ -run TestAcc

# Install to local Terraform plugin directory for dev testing
install: build
	@mkdir -p ~/.terraform.d/plugins/registry.terraform.io/Polar-Team/diffusion/$(VERSION)/$(shell go env GOOS)_$(shell go env GOARCH)
	@cp $(BINARY_NAME) ~/.terraform.d/plugins/registry.terraform.io/Polar-Team/diffusion/$(VERSION)/$(shell go env GOOS)_$(shell go env GOARCH)/$(BINARY_NAME)_v$(VERSION)
	@echo "Installed to ~/.terraform.d/plugins/registry.terraform.io/Polar-Team/diffusion/$(VERSION)/$(shell go env GOOS)_$(shell go env GOARCH)/"

# Format code
fmt:
	go fmt ./...

# Lint
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Generate documentation (requires tfplugindocs)
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate

# Run GoReleaser locally (snapshot, no publish)
snapshot:
	goreleaser release --snapshot --clean
