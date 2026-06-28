# =============================================================================
# Validation module — checks remote host state after diffusion deploy
# =============================================================================
#
# Used as a helper module by the tofu test framework.
# Connects to the remote host via SSH and verifies deployment state.
# =============================================================================

terraform {
  required_version = ">= 1.0"

  required_providers {
    null = {
      source  = "hashicorp/null"
      version = ">= 3.0"
    }
    external = {
      source  = "hashicorp/external"
      version = ">= 2.0"
    }
    local = {
      source  = "hashicorp/local"
      version = ">= 2.0"
    }
  }
}

variable "host" {
  description = "IP or hostname of the remote host to validate"
  type        = string
}

variable "ssh_user" {
  description = "SSH user"
  type        = string
}

variable "ssh_private_key" {
  description = "SSH private key content (from tls_private_key output)"
  type        = string
  sensitive   = true
}

# Write key content to a temp file for SSH usage
resource "local_sensitive_file" "validate_key" {
  content         = var.ssh_private_key
  filename        = "${path.module}/.ssh/validate_key"
  file_permission = "0600"
}

# -----------------------------------------------------------------------------
# Check: diffusion state directory exists
# -----------------------------------------------------------------------------

resource "null_resource" "check_diffusion_state" {
  depends_on = [local_sensitive_file.validate_key]

  triggers = {
    always_run = timestamp()
  }

  connection {
    type        = "ssh"
    host        = var.host
    user        = var.ssh_user
    private_key = var.ssh_private_key
    timeout     = "2m"
  }

  provisioner "remote-exec" {
    inline = [
      "test -d ~/.diffusion/state && echo 'DIFFUSION_STATE_EXISTS=true' || echo 'DIFFUSION_STATE_EXISTS=false'",
    ]
  }
}

data "external" "diffusion_state" {
  depends_on = [null_resource.check_diffusion_state]

  program = [
    "ssh",
    "-o", "StrictHostKeyChecking=no",
    "-o", "UserKnownHostsFile=/dev/null",
    "-i", local_sensitive_file.validate_key.filename,
    "${var.ssh_user}@${var.host}",
    "echo '{\"exists\": \"'$(test -d ~/.diffusion/state && echo true || echo false)'\"}'",
  ]
}

# -----------------------------------------------------------------------------
# Check: role installed
# -----------------------------------------------------------------------------

data "external" "role_installed" {
  depends_on = [null_resource.check_diffusion_state]

  program = [
    "ssh",
    "-o", "StrictHostKeyChecking=no",
    "-o", "UserKnownHostsFile=/dev/null",
    "-i", local_sensitive_file.validate_key.filename,
    "${var.ssh_user}@${var.host}",
    "echo '{\"installed\": \"'$(find ~/.diffusion/ -name '*checkpoint*' -o -name '*waf*' 2>/dev/null | head -1 | grep -q . && echo true || echo false)'\"}'",
  ]
}

# -----------------------------------------------------------------------------
# Outputs for assertions
# -----------------------------------------------------------------------------

output "diffusion_state_exists" {
  description = "Whether ~/.diffusion/state directory exists on remote host"
  value       = data.external.diffusion_state.result.exists == "true"
}

output "role_installed" {
  description = "Whether the checkpoint WAF role artifacts exist on remote host"
  value       = data.external.role_installed.result.installed == "true"
}
