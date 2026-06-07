# =============================================================================
# Diffusion Terraform Provider — AWS EC2 + Checkpoint WAF Role Example
# =============================================================================
#
# This example:
#   1. Provisions an AWS EC2 instance (Ubuntu 22.04)
#   2. Deploys the Checkpoint WAF Yandex Cloud Ansible role via diffusion
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
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
    diffusion = {
      source  = "Polar-Team/diffusion"
      version = ">= 0.1.0"
    }
  }
}

variable "aws_region" {
  default = "eu-central-1"
}

variable "instance_type" {
  default = "t3.medium"
}

variable "key_name" {
  description = "AWS key pair name"
  type        = string
}

variable "ssh_private_key_path" {
  default = "~/.ssh/id_ed25519"
}

provider "aws" {
  region = var.aws_region
}

provider "diffusion" {
  container_tag = "latest-amd64"
}

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
  key_name               = var.key_name
  vpc_security_group_ids = [aws_security_group.waf.id]

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
  }

  tags = {
    Name = "diffusion-checkpoint-waf"
  }
}

resource "diffusion_deploy" "checkpoint_waf" {
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
        ansible_ssh_private_key_file = var.ssh_private_key_path
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

output "instance_public_ip" {
  value = aws_instance.waf.public_ip
}

output "deploy_run_id" {
  value = diffusion_deploy.checkpoint_waf.run_id
}
