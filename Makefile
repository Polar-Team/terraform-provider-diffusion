# terraform-provider-diffusion Makefile
#
# Portable across Windows (cmd / PowerShell / Git Bash), Linux and macOS.
#
# Two independent detections are used:
#   HOST_OS - the machine running make. Selects the filesystem commands the
#             recipes may use and where Terraform looks for local plugins.
#   GOOS    - what we are compiling FOR. Selects the binary suffix (.exe) and
#             the os_arch component of the plugin directory.
#
# Cross compiling works from any host:
#   make build GOOS=linux GOARCH=amd64

BINARY_NAME := terraform-provider-diffusion

# ---------------------------------------------------------------------------
# Host / target detection
# ---------------------------------------------------------------------------
# `go env` is used rather than the OS variable or uname: go is already a hard
# requirement for every target here, and it answers correctly no matter which
# shell make ends up using. GOHOSTOS is the machine, GOOS is the build target.
GO_ENV  := $(shell go env GOHOSTOS GOOS GOARCH)
HOST_OS := $(word 1,$(GO_ENV))
GOOS    ?= $(word 2,$(GO_ENV))
GOARCH  ?= $(word 3,$(GO_ENV))

# Exported so the recipes stay shell agnostic: `VAR=x cmd` is sh only and
# `$$env:VAR` is PowerShell only, but make's own export works everywhere.
# This also makes `make build GOOS=linux` reach the go toolchain.
export GOOS
export GOARCH

ifeq ($(GOOS),windows)
BINARY_EXT := .exe
else
BINARY_EXT :=
endif

BINARY := $(BINARY_NAME)$(BINARY_EXT)

# ---------------------------------------------------------------------------
# Version
# ---------------------------------------------------------------------------
# Derived with make functions only - no sed/awk, which are absent on plain
# Windows. `v0.1.6-3-gabc123-dirty` -> `0.1.6`. Anything that is not a three
# part version (a bare commit sha, or no repo) falls back to 0.0.0, because
# Terraform requires a semver directory name when installing locally.
#
# The .git guard replaces a `2>/dev/null` redirect: make may run recipes through
# either sh or cmd.exe on Windows and neither spelling of the null device is
# portable across both (`2>NUL` under sh creates a real file named NUL).
ifneq ($(wildcard .git),)
GIT_DESCRIBE := $(shell git describe --tags --always --dirty)
else
GIT_DESCRIBE :=
endif

GIT_VERSION := $(patsubst v%,%,$(firstword $(subst -, ,$(GIT_DESCRIBE))))

ifeq ($(words $(subst ., ,$(GIT_VERSION))),3)
VERSION ?= $(GIT_VERSION)
else
VERSION ?= 0.0.0
endif

LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

# ---------------------------------------------------------------------------
# Local plugin directory (Terraform filesystem mirror layout)
# ---------------------------------------------------------------------------
# Windows Terraform reads %APPDATA%\terraform.d\plugins, not ~/.terraform.d.
ifeq ($(HOST_OS),windows)
ifeq ($(strip $(APPDATA)),)
PLUGIN_ROOT := $(subst \,/,$(USERPROFILE))/AppData/Roaming/terraform.d/plugins
else
PLUGIN_ROOT := $(subst \,/,$(APPDATA))/terraform.d/plugins
endif
else
PLUGIN_ROOT := $(HOME)/.terraform.d/plugins
endif

PLUGIN_DIR     := $(PLUGIN_ROOT)/registry.terraform.io/Polar-Team/diffusion/$(VERSION)/$(GOOS)_$(GOARCH)
INSTALL_BINARY := $(BINARY_NAME)_v$(VERSION)$(BINARY_EXT)

.PHONY: build test testacc install clean fmt lint docs snapshot info

# ===========================================================================
# Shell independent targets
# ===========================================================================

# Show what the detection resolved to - run this first when a build lands in
# an unexpected place or without the expected extension.
info:
	@echo host_os=$(HOST_OS) goos=$(GOOS) goarch=$(GOARCH)
	@echo version=$(VERSION) binary=$(BINARY)
	@echo plugin_dir=$(PLUGIN_DIR)
	@echo install_binary=$(INSTALL_BINARY)

# Build the provider binary for GOOS/GOARCH
build:
	go build $(LDFLAGS) -o $(BINARY) .
	@echo Built $(BINARY) for $(GOOS)/$(GOARCH) version $(VERSION)

# Run unit tests
test:
	go test -v -count=1 ./...

# Run acceptance tests (requires terraform on PATH, diffusion on PATH).
# Target specific export keeps TF_ACC out of the plain `make test` run.
testacc: export TF_ACC := 1
testacc:
	go test -v -timeout 30m ./internal/provider/ -run TestAcc

# Format code
fmt:
	go fmt ./...

# Lint
lint:
	golangci-lint run ./...

# Generate documentation (requires tfplugindocs)
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate

# Run GoReleaser locally (snapshot, no publish)
snapshot:
	goreleaser release --snapshot --clean

# ===========================================================================
# Targets that touch the filesystem, so they need per host commands
# ===========================================================================

ifeq ($(HOST_OS),windows)

# PowerShell is used instead of mkdir -p / cp / rm. The commands contain no
# shell metacharacters, so they behave identically whether make's shell is
# cmd.exe or an MSYS sh.

install: build
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(PLUGIN_DIR)' | Out-Null"
	@powershell -NoProfile -Command "Copy-Item -Force '$(BINARY)' '$(PLUGIN_DIR)/$(INSTALL_BINARY)'"
	@echo Installed $(INSTALL_BINARY) to $(PLUGIN_DIR)

# Both binary names are removed so the extensionless artifact produced by the
# previous Makefile is cleaned up too. `exit 0` keeps make happy when nothing
# is there to delete.
clean:
	@powershell -NoProfile -Command "Remove-Item -Force -Recurse -ErrorAction SilentlyContinue '$(BINARY_NAME)','$(BINARY_NAME).exe','dist'; exit 0"

else

install: build
	@mkdir -p "$(PLUGIN_DIR)"
	@cp "$(BINARY)" "$(PLUGIN_DIR)/$(INSTALL_BINARY)"
	@echo "Installed $(INSTALL_BINARY) to $(PLUGIN_DIR)"

clean:
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	@rm -rf dist/

endif
