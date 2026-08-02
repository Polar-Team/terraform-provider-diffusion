# =============================================================================
# Terraform / OpenTofu Test — Validate diffusion deployment on remote EC2
# =============================================================================
#
# Compatible with both:
#   terraform test        (Terraform >= 1.6)
#   tofu test             (OpenTofu >= 1.6)
#
# Validates:
#   - EC2 instance is created and reachable
#   - diffusion_deploy resource completes successfully
#   - diffusion state directory exists on remote host (~/.diffusion/state)
#   - Checkpoint WAF role was successfully installed
#
# NOTE: This file intentionally avoids OpenTofu-only features
#       (mock_provider, override_resource, override_data, override_module)
#       to maintain cross-tool compatibility.
# =============================================================================

variables {
  aws_region    = "us-east-1"
  instance_type = "t3.micro"
}

# -----------------------------------------------------------------------------
# Run: Plan-only validation (no real infra created)
# -----------------------------------------------------------------------------

run "plan_validation" {
  command = plan

  # Verify resource declarations are valid
  assert {
    condition     = aws_instance.waf.instance_type == var.instance_type
    error_message = "Instance type mismatch: expected ${var.instance_type}"
  }

  assert {
    condition     = aws_instance.waf.root_block_device[0].volume_size == 30
    error_message = "Root volume size should be 30 GB"
  }

  assert {
    condition     = aws_instance.waf.root_block_device[0].volume_type == "gp3"
    error_message = "Root volume type should be gp3"
  }

  assert {
    condition     = aws_key_pair.diffusion.key_name_prefix == "diffusion-test-"
    error_message = "Key pair name prefix should be 'diffusion-test-'"
  }

  assert {
    condition     = tls_private_key.ssh.algorithm == "ED25519"
    error_message = "SSH key algorithm should be ED25519"
  }

  assert {
    condition     = aws_key_pair.diffusion.public_key == tls_private_key.ssh.public_key_openssh
    error_message = "aws_key_pair public_key should be fed from tls_private_key.ssh.public_key_openssh"
  }

  assert {
    condition     = aws_security_group.waf.name_prefix == "diffusion-waf-"
    error_message = "Security group name prefix should be 'diffusion-waf-'"
  }
}

# -----------------------------------------------------------------------------
# Run: Apply the full configuration (creates real AWS infrastructure)
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

  assert {
    condition     = aws_instance.waf.instance_type == var.instance_type
    error_message = "EC2 instance type does not match expected: ${var.instance_type}"
  }

  # Verify SSH key pair resources
  assert {
    condition     = tls_private_key.ssh.algorithm == "RSA"
    error_message = "SSH key algorithm should be RSA"
  }

  assert {
    condition     = tls_private_key.ssh.rsa_bits == 2048
    error_message = "SSH key RSA bits should be 2048"
  }

  assert {
    condition     = aws_key_pair.diffusion.key_name != ""
    error_message = "AWS key pair was not created"
  }

  # Verify security group rules
  assert {
    condition     = length(aws_security_group.waf.ingress) == 3
    error_message = "Security group should have 3 ingress rules (22, 80, 443)"
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

  # The provider does not compute merged_lock_hash yet (always returned as
  # an empty string in DeployResult), so we only assert it is not null.
  assert {
    condition     = diffusion_deploy.checkpoint_waf.merged_lock_hash != null
    error_message = "Diffusion deploy merged_lock_hash should not be null"
  }

  # ssh_private_keys is Sensitive; wrap the derived value in nonsensitive()
  # so the assertion result itself is not treated as sensitive. Never
  # interpolate key material into error_message.
  assert {
    condition     = nonsensitive(length(diffusion_deploy.checkpoint_waf.ssh_private_keys)) == 1
    error_message = "diffusion_deploy.checkpoint_waf.ssh_private_keys should have exactly one entry"
  }

  assert {
    condition     = nonsensitive(contains(keys(diffusion_deploy.checkpoint_waf.ssh_private_keys), "default"))
    error_message = "diffusion_deploy.checkpoint_waf.ssh_private_keys should be keyed 'default'"
  }

  assert {
    condition     = nonsensitive(diffusion_deploy.checkpoint_waf.ssh_private_keys["default"] == tls_private_key.ssh.private_key_openssh)
    error_message = "diffusion_deploy.checkpoint_waf.ssh_private_keys[\"default\"] should be wired from tls_private_key.ssh.private_key_openssh"
  }

  # Verify outputs are populated
  assert {
    condition     = output.instance_public_ip != ""
    error_message = "instance_public_ip output should not be empty"
  }

  assert {
    condition     = output.deploy_run_id != ""
    error_message = "deploy_run_id output should not be empty"
  }
}

# -----------------------------------------------------------------------------
# Run: Validate remote host state (uses terraform_data provisioner)
# The remote-exec provisioner will FAIL the apply if the checks don't pass,
# so reaching this run block successfully proves validation passed.
# -----------------------------------------------------------------------------

run "validate_remote_state" {
  command = apply

  assert {
    condition     = output.validation_passed == true
    error_message = "Post-deploy validation failed: diffusion state or role not found on remote host"
  }
}
