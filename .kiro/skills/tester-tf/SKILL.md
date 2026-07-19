---
name: tester-tf
description: Testing provider using terraforma and opentofu binaries under tests for integrations.
---

# Testing

When running or writing tests for this repository, you will need to have the `terraform` and `opentofu` binaries available in your PATH. The tests will invoke these binaries to perform integration tests against the Diffusion provider.

Before using command terraform test ./... or tofu test ./... you need to initialize the provider with the following commands:

```bash
terraform init
```

```bash
tofu init
```

