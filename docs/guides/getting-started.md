---
page_title: "Getting Started with the Diffusion Provider"
subcategory: ""
description: |-
  A step-by-step guide to deploying your first Ansible role using the Diffusion Terraform provider.
---

# Getting Started

This guide walks you through deploying an Ansible role to a remote host using the Diffusion Terraform provider.

## Prerequisites

1. **Diffusion CLI** — install from [GitHub Releases](https://github.com/Polar-Team/diffusion/releases) or via Chocolatey:
   ```bash
   choco install diffusion
   ```

2. **Docker** — the provider executes Ansible inside a container, so Docker must be running.

3. **SSH access** — you need SSH connectivity to your target hosts.

4. **Terraform or OpenTofu** — version 1.0 or later.

## Step 1: Configure the Provider

Create a `main.tf`:

```terraform
terraform {
  required_providers {
    diffusion = {
      source  = "Polar-Team/diffusion"
      version = "~> 0.1"
    }
  }
}

provider "diffusion" {
  # Uses default settings:
  #   registry_server = "ghcr.io"
  #   container_tag   = "latest"
  #   All other options auto-detected
}
```

## Step 2: Define Your Deployment

Add a `diffusion_deploy` resource pointing to your Ansible role:

```terraform
resource "diffusion_deploy" "myapp" {
  role_sources = [
    {
      scm     = "git"
      url     = "https://github.com/your-org/ansible-role-myapp.git"
      version = "v1.0.0"
    },
  ]

  hosts = {
    "server-01" = {
      vars = {
        ansible_host            = "192.168.1.100"
        ansible_user            = "deploy"
        ansible_ssh_private_key_file = "~/.ssh/id_ed25519"
      }
    }
  }
}
```

## Step 3: Preview the Inventory (Optional)

Use the `diffusion_inventory` data source to inspect what the inventory looks like before deploying:

```terraform
data "diffusion_inventory" "preview" {
  hosts = {
    "server-01" = {
      vars = {
        ansible_host = "192.168.1.100"
        ansible_user = "deploy"
      }
    }
  }
}

output "inventory_yaml" {
  value = data.diffusion_inventory.preview.rendered
}
```

Run `terraform plan` to see the rendered YAML without deploying anything.

## Step 4: Deploy

```bash
terraform init
terraform plan
terraform apply
```

The provider will:
1. Pull the diffusion molecule container
2. Fetch `diffusion.lock` from your role repository
3. Install dependencies inside the container
4. Run `ansible-playbook` against your hosts
5. Record the deployment state

## Step 5: Idempotent Re-runs

Add `skip_if_succeeded_within` to avoid unnecessary re-deployments:

```terraform
resource "diffusion_deploy" "myapp" {
  # ... same as above ...

  skip_if_succeeded_within = "24h"
}
```

Now `terraform apply` will skip the deployment if the last run succeeded within 24 hours and all inputs are identical.

## Using with Cloud Providers

The provider works well alongside cloud providers like AWS, Azure, or GCP. A common pattern:

1. Provision infrastructure (VMs, networking) with the cloud provider
2. Deploy configuration (Ansible roles) with the diffusion provider
3. Use host wait settings to handle boot delays

```terraform
resource "aws_instance" "app" {
  ami           = "ami-0123456789"
  instance_type = "t3.medium"
  key_name      = "my-key"
}

resource "diffusion_deploy" "app" {
  role_sources = [
    {
      scm     = "git"
      url     = "https://github.com/org/role.git"
      version = "v1.0.0"
    },
  ]

  hosts = {
    "app-01" = {
      vars = {
        ansible_host       = aws_instance.app.public_ip
        ansible_user       = "ubuntu"
        ansible_ssh_common_args = "-o StrictHostKeyChecking=no"
      }
    }
  }

  # Wait for the instance to become SSH-reachable
  host_wait_initial_delay = "30s"
  host_wait_timeout       = "5m"
}
```

## Next Steps

- See the [`diffusion_deploy`](../resources/deploy.md) resource documentation for all options
- See the [`diffusion_inventory`](../data-sources/inventory.md) data source for inventory rendering
- Check the [tests/aws-ec2-checkpoint-waf](https://github.com/Polar-Team/terraform-provider-diffusion/tree/main/tests/aws-ec2-checkpoint-waf) directory for a complete working example with tftest
