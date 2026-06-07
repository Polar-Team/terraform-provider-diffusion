# Publishing terraform-provider-diffusion

## Overview

| Registry | Detection | First-time setup |
|----------|-----------|------------------|
| Terraform Registry (registry.terraform.io) | Webhook (auto on GitHub release) | Register repo + add GPG key |
| OpenTofu Registry | Auto after first PR | Submit manifest PR |

## Prerequisites

### GPG Signing Key

Both registries require GPG-signed SHA256SUMS (not Cosign).

```bash
# Generate key
gpg --full-generate-key
# RSA 4096, no expiration, use your GitHub email

# Export public key (for registry registration)
gpg --armor --export <KEY_ID> > gpg-public-key.asc

# Export private key (for GitHub Secrets)
gpg --armor --export-secret-keys <KEY_ID>
```

### GitHub Repository Secrets

| Secret | Value |
|--------|-------|
| `GPG_PRIVATE_KEY` | Armored private key |
| `GPG_PASSPHRASE` | Key passphrase |

## Release Process

```bash
git tag v0.1.0
git push origin v0.1.0
```

This triggers `.github/workflows/release.yml`:
1. GoReleaser builds for all platforms
2. Creates ZIP archives with correct naming
3. Generates GPG-signed SHA256SUMS
4. Attaches `terraform-registry-manifest.json`
5. Creates GitHub Release

## Terraform Registry (One-time)

1. Go to [registry.terraform.io/publish/provider](https://registry.terraform.io/publish/provider)
2. Sign in with the GitHub owner account
3. Select `terraform-provider-diffusion` repository
4. Add GPG public key under Settings → GPG Keys

After setup, every GitHub release is auto-detected.

## OpenTofu Registry (One-time)

1. Fork [github.com/opentofu/registry](https://github.com/opentofu/registry)
2. Create `providers/p/Polar-Team/diffusion.json`:

```json
{
  "repository": "https://github.com/Polar-Team/terraform-provider-diffusion",
  "key": "-----BEGIN PGP PUBLIC KEY BLOCK-----\n...\n-----END PGP PUBLIC KEY BLOCK-----"
}
```

3. Submit PR — after merge, future releases are auto-detected.

## Verification

```bash
# After first release
terraform init  # Should resolve from registry
tofu init       # Same for OpenTofu
```

## Produced Artifacts

Each release creates:
```
terraform-provider-diffusion_0.1.0_linux_amd64.zip
terraform-provider-diffusion_0.1.0_linux_arm64.zip
terraform-provider-diffusion_0.1.0_linux_arm.zip
terraform-provider-diffusion_0.1.0_darwin_amd64.zip
terraform-provider-diffusion_0.1.0_darwin_arm64.zip
terraform-provider-diffusion_0.1.0_windows_amd64.zip
terraform-provider-diffusion_0.1.0_windows_arm64.zip
terraform-provider-diffusion_0.1.0_SHA256SUMS
terraform-provider-diffusion_0.1.0_SHA256SUMS.sig
terraform-registry-manifest.json
```
