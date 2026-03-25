## Why

Vynilino ships as a self-hosted container image that users run on their own infrastructure. Without supply-chain integrity guarantees, users have no way to verify that the artifact they downloaded was built from the exact source they trust and was not tampered with in transit. Achieving SLSA Level 2 (and documenting the path to Level 3) closes this gap by producing signed build provenance, signed container images, and a software bill of materials with every release.

## What Changes

- Add a GitHub Actions release workflow that generates SLSA provenance using `slsa-github-generator`
- Sign the released container image with `cosign` (keyless, Sigstore transparency log)
- Generate and attach an SBOM (SPDX format) to every release via `syft`
- Pin all workflow action dependencies to full commit SHAs to prevent mutable-tag attacks
- Add a `Makefile` target and documentation for local provenance verification
- Produce a `SECURITY.md` that describes the attestation trust model and verification instructions for end users

## Capabilities

### New Capabilities
- `slsa-build-provenance`: CI/CD pipeline produces and publishes SLSA provenance attestations for every release binary and container image
- `artifact-signing`: Container images and release binaries are signed via cosign (keyless Sigstore), enabling cryptographic verification without managing signing keys
- `sbom-generation`: An SPDX SBOM is generated for each release and attached as a GitHub release asset and OCI attestation

### Modified Capabilities
<!-- No existing spec requirements change — this is new build/release infrastructure. -->

## Impact

- **CI/CD**: New GitHub Actions workflow file(s); existing workflows updated to pin action SHAs
- **Release process**: Container image publishing via GHCR gains cosign attestations and SBOM OCI layer
- **Go binary releases**: `goreleaser` (or direct `go build`) gains provenance attestation step
- **Dependencies**: `slsa-github-generator`, `cosign`, `syft` added as release-time tools (not runtime)
- **Documentation**: `SECURITY.md` added at repo root; `README.md` updated with verification badge and instructions
- **No runtime code changes**: zero impact on the application itself
