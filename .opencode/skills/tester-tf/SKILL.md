---
name: tester-tf
description: Use when running or writing integration tests for terraform-provider-diffusion with the terraform or tofu binaries — .tftest.hcl files, "terraform test", "tofu test", make testacc, or provider init/plugin-mirror setup.
---

# Testing the provider with terraform and opentofu

When running or writing tests for this repository you need the `terraform` and `opentofu` (`tofu`) binaries available on your `PATH`. The tests invoke these binaries to perform integration tests against the Diffusion provider.

## Install the provider locally first

Terraform tests resolve the provider from the local filesystem mirror, so build and install it before running any test:

```bash
make install
```

This places `terraform-provider-diffusion_v<version>` under the plugin directory reported by:

```bash
make info
```

## Initialize before testing

Before using `terraform test ./...` or `tofu test ./...` you must initialize the provider in the test directory:

```bash
terraform init
```

```bash
tofu init
```

## Run the tests

```bash
terraform test
tofu test
```

Go-level acceptance tests are separate and are gated behind `TF_ACC`:

```bash
make testacc
```

## Notes

- Test fixtures live under `tests/`, e.g. `tests/aws-ec2-checkpoint-waf/deploy.tftest.hcl`.
- Both `terraform` and `tofu` should be exercised — behavior can diverge between them.
- The provider shells out to the `diffusion` CLI, so that binary must also be on `PATH` for anything beyond plan-only tests.
