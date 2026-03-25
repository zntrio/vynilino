# Security & Supply Chain Verification

Vynilino releases are SLSA Level 2 compliant. Every release includes:

- **Signed container image** — keyless cosign signature on GHCR
- **SLSA provenance** — signed `.intoto.jsonl` file attesting the build origin
- **SPDX SBOM** — bill of materials for the container image, as both a release asset and OCI attestation

The signing identity is the GitHub Actions release workflow, verified via the Sigstore Rekor transparency log. No long-lived signing keys are used.

---

## Verify a Container Image (cosign)

### Install cosign

```bash
# macOS
brew install cosign

# Linux
curl -fsSL https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64 -o cosign
chmod +x cosign && sudo mv cosign /usr/local/bin/
```

### Verify the signature

```bash
cosign verify \
  --certificate-identity-regexp="https://github.com/YOUR_ORG/vynilino/.github/workflows/release.yml@refs/tags/v.*" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  ghcr.io/YOUR_ORG/vynilino:VERSION
```

A successful verification prints `Verified OK` and the signature payload containing the repository, commit SHA, and workflow reference.

---

## Verify a Release Binary (slsa-verifier)

### Install slsa-verifier

```bash
# macOS
brew install slsa-verifier

# Or download from: https://github.com/slsa-framework/slsa-verifier/releases
```

### Verify binary provenance

Download the binary and its `.intoto.jsonl` provenance file from the GitHub release page, then:

```bash
slsa-verifier verify-artifact \
  --provenance-path vynilino_VERSION_linux_amd64.intoto.jsonl \
  --source-uri github.com/YOUR_ORG/vynilino \
  vynilino_VERSION_linux_amd64.tar.gz
```

Expected output: `PASSED: Verified SLSA provenance`

---

## Verify the SBOM Attestation (cosign)

```bash
cosign verify-attestation \
  --type spdxjson \
  --certificate-identity-regexp="https://github.com/YOUR_ORG/vynilino/.github/workflows/release.yml@refs/tags/v.*" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  ghcr.io/YOUR_ORG/vynilino:VERSION \
  | jq '.payload | @base64d | fromjson'
```

This returns the full SPDX SBOM JSON with all Go module and OS package dependencies listed.

The SBOM is also available as a downloadable file on the GitHub release page: `vynilino-VERSION-sbom.spdx.json`.

---

## Trust Model

| Artifact | Signing identity | Transparency log |
|---|---|---|
| Container image signature | `github.com/YOUR_ORG/vynilino` workflow | Sigstore Rekor |
| SLSA binary provenance | `slsa-framework/slsa-github-generator` workflow | Sigstore Rekor |
| SBOM OCI attestation | `github.com/YOUR_ORG/vynilino` workflow | Sigstore Rekor |

All signatures are ephemeral (keyless). Verification does not require trusting any private key — only the Sigstore Rekor log and the GitHub OIDC issuer.

---

## Reporting Security Issues

Please do not open public issues for security vulnerabilities. Contact the maintainers directly via GitHub private vulnerability reporting:
`https://github.com/YOUR_ORG/vynilino/security/advisories/new`
