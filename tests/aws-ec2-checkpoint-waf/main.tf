# =============================================================================
# Diffusion Terraform Provider — AWS EC2 + Checkpoint WAF Role Example
# =============================================================================
#
# This example:
#   1. Generates an SSH key pair (tls_private_key + aws_key_pair)
#   2. Provisions an AWS EC2 instance (Ubuntu 22.04)
#   3. Deploys the Checkpoint WAF Yandex Cloud Ansible role via diffusion
#
# Usage:
#   tofu init && tofu apply
#
# Prerequisites:
#   - AWS credentials configured
#   - diffusion CLI on $PATH
#   - Docker running
# =============================================================================

terraform {
  required_version = ">= 1.4"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = ">= 4.0"
    }
    diffusion = {
      source  = "Polar-Team/diffusion"
      version = ">= 0.1.0"
    }
  }
}

variable "aws_region" {
  description = "AWS region for EC2 instance"
  type        = string
  default     = "us-east-1"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

provider "aws" {
  region = var.aws_region
}

provider "diffusion" {
  container_tag = "latest-amd64"
}

# -----------------------------------------------------------------------------
# SSH Key Generation
# -----------------------------------------------------------------------------

resource "tls_private_key" "ssh" {
  algorithm = "ED25519"
}

resource "aws_key_pair" "diffusion" {
  key_name_prefix = "diffusion-test-"
  public_key      = tls_private_key.ssh.public_key_openssh
}

# Write private key to ~/.ssh so diffusion's container can mount it via /root/.ssh.
# The container only mounts the user's ~/.ssh directory — project-local paths won't
# be visible inside the probe/deploy containers.
# Uses terraform_data (built-in) to avoid requiring the 'local' provider.
resource "terraform_data" "ssh_key" {
  input = tls_private_key.ssh.private_key_openssh

  provisioner "local-exec" {
    command = <<-EOT
      mkdir -p ~/.ssh
      printf '%s' '${tls_private_key.ssh.private_key_openssh}' > ~/.ssh/diffusion-test-ed25519
      chmod 600 ~/.ssh/diffusion-test-ed25519
    EOT
  }

  provisioner "local-exec" {
    when    = destroy
    command = "rm -f ~/.ssh/diffusion-test-ed25519"
  }
}

# -----------------------------------------------------------------------------
# EC2 Instance
# -----------------------------------------------------------------------------

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
}

resource "aws_security_group" "waf" {
  name_prefix = "diffusion-waf-"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "waf" {
  ami                    = data.aws_ami.ubuntu.id
  instance_type          = var.instance_type
  key_name               = aws_key_pair.diffusion.key_name
  vpc_security_group_ids = [aws_security_group.waf.id]

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
  }

  tags = {
    Name = "diffusion-checkpoint-waf-test"
  }
}

# -----------------------------------------------------------------------------
# Deploy: Checkpoint WAF role via diffusion
# -----------------------------------------------------------------------------

resource "diffusion_deploy" "checkpoint_waf" {
  depends_on = [terraform_data.ssh_key]

  role_sources = [
    {
      scm     = "git"
      url     = "https://github.com/Morshimus/ansible-role-checkpoint-waf-yandex-cloud.git"
      version = "main"
      name    = "checkpoint_waf_yandex_cloud"
    },
  ]

  hosts = {
    "waf-01" = {
      vars = {
        ansible_host                 = aws_instance.waf.public_ip
        ansible_user                 = "ubuntu"
        ansible_ssh_private_key_file = "~/.ssh/diffusion-test-ed25519"
        ansible_ssh_common_args      = "-o StrictHostKeyChecking=no"
      }
    }
  }

  groups = {
    checkpoint_waf = ["waf-01"]
  }

  variables = {
    ansible_python_interpreter = "/usr/bin/python3"
  }

  skip_if_succeeded_within = "1h"
  host_wait_initial_delay  = "30s"
  host_wait_timeout        = "5m"
}

# -----------------------------------------------------------------------------
# Post-deploy validation: verify remote host state via SSH
# Uses terraform_data (built-in) — no additional providers needed.
# -----------------------------------------------------------------------------

resource "terraform_data" "validate_remote_state" {
  depends_on = [diffusion_deploy.checkpoint_waf]

  triggers_replace = [diffusion_deploy.checkpoint_waf.run_id]

  connection {
    type        = "ssh"
    host        = aws_instance.waf.public_ip
    user        = "ubuntu"
    private_key = tls_private_key.ssh.private_key_openssh
    timeout     = "2m"
  }

  provisioner "remote-exec" {
    inline = [
      "echo 'Validating diffusion state on remote host...'",
      "test -d ~/.diffusion/state",
      "find ~/.diffusion/ -name '*checkpoint*' -o -name '*waf*' | grep -q .",
    ]
  }
}

# -----------------------------------------------------------------------------
# Outputs
# -----------------------------------------------------------------------------

output "instance_public_ip" {
  value = aws_instance.waf.public_ip
}

output "deploy_run_id" {
  value = diffusion_deploy.checkpoint_waf.run_id
}

output "ssh_private_key" {
  description = "Generated SSH private key (sensitive)"
  value       = tls_private_key.ssh.private_key_openssh
  sensitive   = true
}

output "validation_passed" {
  description = "Whether post-deploy validation completed successfully"
  value       = terraform_data.validate_remote_state.id != ""
}
