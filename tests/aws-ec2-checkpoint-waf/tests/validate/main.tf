# =============================================================================
# Validation module — checks remote host state after diffusion deploy
# =============================================================================

terraform {
  required_version = ">= 1.0"

  required_providers {
    null = {
      source  = "hashicorp/null"
      version = ">= 3.0"
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
  description = "Path to SSH private key"
  type        = string
}

# -----------------------------------------------------------------------------
# Check: diffusion state directory exists
# -----------------------------------------------------------------------------

resource "null_resource" "check_diffusion_state" {
  connection {
    type        = "ssh"
    host        = var.host
    user        = var.ssh_user
    private_key = file(var.ssh_private_key)
    timeout     = "2m"
  }

  provisioner "remote-exec" {
    inline = [
      "test -d ~/.diffusion/state && echo 'DIFFUSION_STATE_EXISTS=true' > /tmp/diffusion_check.txt || echo 'DIFFUSION_STATE_EXISTS=false' > /tmp/diffusion_check.txt",
    ]
  }
}

data "external" "diffusion_state" {
  depends_on = [null_resource.check_diffusion_state]

  program = [
    "ssh",
    "-o", "StrictHostKeyChecking=no",
    "-i", var.ssh_private_key,
    "${var.ssh_user}@${var.host}",
    "echo '{\"exists\": \"'$(test -d ~/.diffusion/state && echo true || echo false)'\"}'",
  ]
}

# -----------------------------------------------------------------------------
# Check: role installed (ansible roles path or diffusion cache)
# -----------------------------------------------------------------------------

data "external" "role_installed" {
  depends_on = [null_resource.check_diffusion_state]

  program = [
    "ssh",
    "-o", "StrictHostKeyChecking=no",
    "-i", var.ssh_private_key,
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
