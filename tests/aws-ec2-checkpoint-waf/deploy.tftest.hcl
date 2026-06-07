# =============================================================================
# Terraform Test — Validate diffusion deployment on remote EC2 instance
# =============================================================================
#
# Runs:
#   tofu test        (or: terraform test)
#
# Validates:
#   - EC2 instance is created and reachable
#   - diffusion state directory exists on remote host (~/.diffusion/state)
#   - Checkpoint WAF role was successfully installed
# =============================================================================

variables {
  aws_region           = "eu-central-1"
  instance_type        = "t3.medium"
  key_name             = "diffusion-test-key"
  ssh_private_key_path = "~/.ssh/id_ed25519"
  ssh_user             = "ubuntu"
  allowed_ssh_cidr     = "0.0.0.0/0"
}

# -----------------------------------------------------------------------------
# Run: Apply the full configuration
# -----------------------------------------------------------------------------

run "deploy_checkpoint_waf" {
  command = apply

  # Verify EC2 instance was created
  assert {
    condition     = aws_instance.waf.id != ""
    error_message = "EC2 instance was not created"
  }

  assert {
    condition     = aws_instance.waf.public_ip != ""
    error_message = "EC2 instance has no public IP assigned"
  }

  # Verify diffusion_deploy resource completed
  assert {
    condition     = diffusion_deploy.checkpoint_waf.run_id != ""
    error_message = "Diffusion deploy did not produce a run_id"
  }

  assert {
    condition     = diffusion_deploy.checkpoint_waf.last_deployed != ""
    error_message = "Diffusion deploy did not record last_deployed timestamp"
  }

  assert {
    condition     = diffusion_deploy.checkpoint_waf.merged_lock_hash != ""
    error_message = "Diffusion deploy did not produce a merged_lock_hash"
  }
}

# -----------------------------------------------------------------------------
# Run: Validate remote host state via SSH
# -----------------------------------------------------------------------------

run "validate_remote_state" {
  command = apply

  module {
    source = "./tests/validate"
  }

  variables {
    host             = run.deploy_checkpoint_waf.instance_public_ip
    ssh_user         = var.ssh_user
    ssh_private_key  = var.ssh_private_key_path
  }

  # Diffusion state directory must exist
  assert {
    condition     = output.diffusion_state_exists == true
    error_message = "Diffusion state directory (~/.diffusion/state) does not exist on remote host"
  }

  # Role must be installed
  assert {
    condition     = output.role_installed == true
    error_message = "Checkpoint WAF role was not installed on remote host"
  }
}
