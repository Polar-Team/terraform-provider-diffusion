# Terraform Provider: Diffusion

The `diffusion` Terraform/OpenTofu provider deploys Ansible roles to remote hosts using the [Diffusion](https://github.com/Polar-Team/diffusion) molecule container.

## Features

- Deploy Ansible roles from Git repositories or Ansible Galaxy
- Automatic dependency resolution via `diffusion.lock`
- Inventory management (hosts, groups, variables)
- Idempotent deployments with configurable skip periods
- Host reachability probing before deployment
- HashiCorp Vault integration for credentials
- Private artifact source support

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0 (or [OpenTofu](https://opentofu.org/) >= 1.0)
- [Go](https://golang.org/doc/install) >= 1.25 (for building from source)
- [Diffusion CLI](https://github.com/Polar-Team/diffusion) installed and on `$PATH`
- Docker running (the provider uses the diffusion molecule container)

## Installation

### From Registry

```hcl
terraform {
  required_providers {
    diffusion = {
      source  = "Polar-Team/diffusion"
      version = "~> 0.1"
    }
  }
}
```

### Local Development

```bash
make install
```

Or use `dev_overrides` in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "Polar-Team/diffusion" = "/path/to/terraform-provider-diffusion"
  }
  direct {}
}
```

## Usage

```hcl
provider "diffusion" {
  registry_server = "ghcr.io"
  container_tag   = "latest-amd64"
}

resource "diffusion_deploy" "app" {
  role_sources = [
    {
      scm     = "git"
      url     = "https://github.com/org/ansible-role-myapp.git"
      version = "v1.0.0"
    },
  ]

  hosts = {
    "web-01" = {
      vars = {
        ansible_host = "192.168.1.10"
        ansible_user = "deploy"
      }
    }
  }

  groups = {
    webservers = ["web-01"]
  }

  skip_if_succeeded_within = "24h"
}
```

## Resources

- `diffusion_deploy` — Deploys Ansible roles to remote hosts

## Data Sources

- `diffusion_inventory` — Renders an Ansible YAML inventory (no deployment)

## Tests

Integration tests live in `tests/`:

```bash
# Run unit tests
make test

# Run Terraform integration tests (requires AWS credentials + infrastructure)
cd tests/aws-ec2-checkpoint-waf
tofu test
```

## Development

```bash
# Build
make build

# Run acceptance tests (requires infrastructure)
make testacc

# Generate docs
make docs

# Local snapshot release
make snapshot
```

## Publishing

This provider is published to:

- **Terraform Registry**: [registry.terraform.io/providers/Polar-Team/diffusion](https://registry.terraform.io/providers/Polar-Team/diffusion)
- **OpenTofu Registry**: Auto-detected from GitHub releases

Releases are automated via GoReleaser on tag push (`v*.*.*`). See [PUBLISHING.md](docs/PUBLISHING.md) for setup details.

## License

MIT — see [LICENSE](LICENSE).
