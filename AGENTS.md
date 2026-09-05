# terraform-provider-diffusion — Technical Steering Document

## Project Overview

`terraform-provider-diffusion` is a Terraform / OpenTofu provider that exposes the
[Diffusion CLI](https://github.com/Polar-Team/diffusion) `deploy` module directly from HCL.
It lets users run Ansible-based deployments as part of a normal `terraform plan` / `terraform apply`
lifecycle, with inventories built from Terraform state.

- **Language**: Go
- **Framework**: [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework)
- **Registry namespace**: `registry.terraform.io/Polar-Team/diffusion`
- **Provider type name**: `diffusion`
- **License**: MIT

## Provider Surface

| Kind | Address | Source |
|---|---|---|
| Resource | `diffusion_deploy` | `internal/provider/resource_deploy.go` |
| Data source | `diffusion_inventory` | `internal/provider/datasource_inventory.go` |

## Repository Layout

```
.
├── main.go                     # Provider entry point; Version injected via LDFLAGS
├── internal/
│   ├── provider/               # Provider schema, resource, data source, runner, validators
│   │   ├── provider.go
│   │   ├── resource_deploy.go
│   │   ├── datasource_inventory.go
│   │   ├── runner.go                        # Executes the diffusion CLI
│   │   └── validator_map_key_no_equals.go
│   └── deploy/
│       └── inventory.go        # Inventory construction helpers
├── tests/                      # Terraform-level integration tests (*.tftest.hcl)
├── docs/                       # Registry documentation (generated via tfplugindocs)
├── .github/workflows/          # test.yml, tftest.yml, release.yml
├── .goreleaser.yml             # Cross-platform release build + signing
├── terraform-registry-manifest.json
└── Makefile
```

## Build & Test

```bash
make info        # Show resolved host OS, target GOOS/GOARCH, version, plugin dir
make build       # Build the provider binary for GOOS/GOARCH
make install     # Build and install into the local Terraform filesystem mirror
make test        # go test -v -count=1 ./...
make testacc     # Acceptance tests (sets TF_ACC=1; requires terraform and diffusion on PATH)
make fmt         # go fmt ./...
make lint        # golangci-lint run ./...
make docs        # Regenerate docs/ with tfplugindocs
make snapshot    # Local GoReleaser snapshot build (no publish)
make clean       # Remove build artifacts
```

The Makefile is portable across Windows (cmd / PowerShell / Git Bash), Linux, and macOS.
Host and target are detected with `go env`; `VERSION` is derived from `git describe --tags`
and falls back to `0.0.0` when no three-part semver tag is reachable.

Cross-compiling works from any host:

```bash
make build GOOS=linux GOARCH=amd64
```

## Integration Testing

Terraform-level tests require both `terraform` and `tofu` on `PATH`, plus a locally installed
provider build. See the `tester-tf` skill for the full workflow.

```bash
make install
terraform init && terraform test
tofu init && tofu test
```

## Conventions

- The provider schema is a **public API**. Adding attributes is cheap; renaming or removing them
  is a breaking change and requires a major version bump.
- Regenerate `docs/` with `make docs` whenever the schema changes.
- Never place credentials in state, plan output, or logs — mark sensitive attributes as `Sensitive: true`.
- The provider shells out to the `diffusion` binary. Always build argument slices; never construct
  shell command strings. Sanitize any value that reaches the command line or a file name.
- Honour `context.Context` cancellation and Terraform timeouts in all CLI invocations.
- Write table-driven unit tests. Add `.tftest.hcl` coverage under `tests/` for HCL-visible behavior.
- Releases are cut by pushing a `vX.Y.Z` tag, which triggers `.github/workflows/release.yml`.

## Agents

Agent definitions live in `.opencode/agents/`:

| Agent | Mode | Role |
|---|---|---|
| `team-lead` | primary | Architecture, planning, coordination. Read-only; delegates implementation. |
| `go-senior-developer` | subagent | Go / terraform-plugin-framework implementation and tests. |
| `devops-senior-engineer` | subagent | CI/CD, GoReleaser, Makefile, Registry publishing. |

Skills live in `.opencode/skills/`:

| Skill | Purpose |
|---|---|
| `tester-tf` | Running provider integration tests with the `terraform` and `tofu` binaries. |
