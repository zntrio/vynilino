## Context

Vynilino is a self-hosted Go + embedded-frontend service distributed as a Docker image and potentially as standalone binaries. The current release process has no cryptographic guarantees that a downloaded artifact was built from the published source. Users running vynilino in home-lab or production environments cannot verify supply-chain integrity.

SLSA (Supply chain Levels for Software Artifacts, https://slsa.dev) defines a graduated framework. This design targets **SLSA Level 2** (hosted source + build, signed provenance) as an achievable first milestone with a documented path to Level 3 (hardened, non-falsifiable provenance via the `slsa-github-generator` reusable workflow).

Current state:
- Releases are manual or ad-hoc; no formal release pipeline
- Container images pushed to a registry with no signatures
- No SBOM produced
- Workflow action versions are not pinned by SHA

## Goals / Non-Goals

**Goals:**
- Achieve SLSA Level 2 for release binaries and the container image
- Sign container images keylessly via cosign + Sigstore (no key management overhead)
- Generate and publish an SPDX SBOM for each release
- Pin all GitHub Actions action references to full commit SHAs
- Document end-user verification steps in `SECURITY.md`
- Zero changes to application runtime code

**Non-Goals:**
- SLSA Level 3 or 4 (hermetic builds, two-person review) — acknowledged as future work
- Signing development/PR builds (only release tags)
- Reproducible builds (requires additional tooling investment outside this scope)
- Dependency auditing / vulnerability scanning (separate concern)

## Decisions

### Decision 1: goreleaser for release builds; `go build` for development

**Choice**: `goreleaser/goreleaser-action` (SHA-pinned) with a `.goreleaser.yaml` config for the release pipeline. Local development continues to use `go build ./...` unchanged.

**Rationale**: Goreleaser handles multi-arch cross-compilation (`linux/amd64`, `linux/arm64`, `darwin/arm64`), checksums, GitHub release creation, and archive naming in a single declarative config. Its native `slsa-goreleaser` reusable workflow (`generator_go_slsa3.yml`) produces SLSA provenance without requiring the generic generator. `go build` is retained for development — goreleaser is a release-time concern only and should not slow down the inner loop.

**Alternatives considered**:
- Generic `slsa-github-generator` + manual `go build` matrix: More boilerplate for the same outcome. Goreleaser is strictly more capable when multi-arch and release packaging are needed.
- goreleaser for development too: Unnecessary overhead; `go build` is instant and dependency-free for local work.

### Decision 2: Keyless signing via cosign + Sigstore

**Choice**: `sigstore/cosign-installer` + `cosign sign --yes` with workload identity (OIDC from GitHub Actions).

**Rationale**: Keyless signing uses the GitHub Actions OIDC token to bind the signature to the workflow identity (repository, SHA, ref). No long-lived signing key to rotate or protect. Verification requires only `cosign verify` with the repo URL as the identity.

**Alternatives considered**:
- Cosign with KMS key (AWS/GCP): Requires managing a key and cloud credentials; adds operational overhead disproportionate to project size.
- GPG signing: Standard but requires distributing and managing public keys; no transparency log.

### Decision 3: SBOM via `syft` in SPDX-JSON format

**Choice**: `anchore/syft` producing `spdx-json` attached as a release asset and pushed as an OCI attestation.

**Rationale**: SPDX is the ISO/IEC 5962:2021 standard and supported by most downstream tooling. `syft` supports Go modules and container image scanning in a single invocation. Attaching as both release asset (human-accessible) and OCI attestation (machine-verifiable) gives maximum interoperability.

**Alternatives considered**:
- CycloneDX: Also widely supported but SPDX is the SLSA-recommended format for interoperability.
- `bom` (Kubernetes project): Less active; fewer integrations.

### Decision 4: Pin all action SHAs in workflows; enforce with zizmor

**Choice**: Replace all `uses: action/foo@v1` with `uses: action/foo@<full-sha>` plus a comment with the human-readable version. Add `zizmor` as a blocking CI lint step on every PR that touches `.github/workflows/`.

**Rationale**: Mutable tags can be hijacked — SHA pinning is a hard SLSA requirement for Level 2+. Zizmor goes further: it catches overly broad `permissions:` declarations, missing `permissions:` blocks, dangerous expression injection patterns, and other GitHub Actions security issues that SHA pinning alone does not cover.

**Alternatives considered**:
- Dependabot/Renovate SHA pin auto-update: Complements zizmor (keeps pins current) — recommended to add after this change for ongoing maintenance, but not a substitute for zizmor's broader security checks.
- Manual review only: Does not scale; people forget.

### Decision 5: Release workflow triggers on `v*` tags only

**Choice**: The new `release.yml` workflow triggers on `push: tags: ['v*']`. PR and main-branch builds remain unaffected.

**Rationale**: Provenance is only meaningful for released artifacts. Generating attestations on every commit would create noise without user benefit.

## Risks / Trade-offs

- **Sigstore dependency**: Keyless signing relies on the public Sigstore Rekor transparency log. If Rekor is unavailable during a release, signing fails. → Mitigation: Add `--no-certificate-verify` fallback or handle gracefully in the workflow with a retry.
- **slsa-github-generator reusable workflow permissions**: The generator requires `id-token: write` and `contents: write` permissions. Granting broad permissions to a reusable workflow is a trust decision. → Mitigation: Pin the generator to a specific SHA and review its changelog before upgrades.
- **Large release artifacts**: Adding SBOM and provenance files increases release asset size. → Negligible for a typical Go binary; accepted trade-off.
- **User friction for verification**: End users must install `cosign` and `slsa-verifier` to verify. Most users will not bother. → Mitigation: Document clearly in `SECURITY.md`; the primary audience is security-conscious operators.
- **No reproducible builds**: Two independent builds of the same source may produce different binaries (embedded timestamps, CGO). SLSA Level 2 does not require reproducibility, but users expecting bit-for-bit reproducibility will be disappointed. → Document explicitly as a non-goal.

## Migration Plan

1. Add `release.yml` workflow (no impact on existing dev workflows)
2. Update existing workflow action references to SHA pins
3. Add `SECURITY.md` at repo root
4. Update `README.md` with verification badge
5. On next `v*` tag push, the full pipeline runs and produces signed artifacts
6. No database migrations, no runtime changes, no user-facing API changes

**Rollback**: Delete the `release.yml` workflow. Existing releases are unaffected.

### Decision 6: GHCR as the sole container registry

**Choice**: Container images are pushed to and signed on `ghcr.io/<owner>/vynilino` only.

**Rationale**: GHCR integrates natively with GitHub Actions OIDC for cosign attestations, has no pull rate limits for GitHub users, and keeps the entire toolchain within a single platform. Docker Hub adds distribution surface without meaningful benefit for a self-hosted tool where operators pull images by name.

**Alternatives considered**:
- Docker Hub in addition to GHCR: More discoverable, but doubles the signing/attestation surface and requires managing additional registry credentials.

### Decision 7: CI workflow — four parallel jobs; nilaway non-blocking

**Choice**: A single `ci.yml` workflow with four parallel jobs: `build` (compile + test), `lint` (golangci-lint default config), `nilaway` (`continue-on-error: true`), `zizmor` (path-filtered to `.github/workflows/**`, blocking).

**Rationale**: Parallel jobs maximize feedback speed. golangci-lint uses default config to avoid a custom ruleset maintenance burden. Nilaway runs with `continue-on-error: true` because it has known false positives on generated code (e.g., `internal/adapter/graphql/graph/generated.go`); informational mode lets the team build a baseline before promoting it to blocking. Zizmor is blocking but scoped to workflow file changes — it has no opinion on Go code.

**Alternatives considered**:
- Single lint job with all tools: nilaway's runtime serializes with golangci-lint and delays PR feedback significantly.
- nilaway blocking from day one: High risk of noisy false positives blocking work before a baseline is established.
- Custom golangci-lint config: Unnecessary complexity at this stage; defaults cover the most impactful linters.

### Decision 8: Align `make lint` with CI; nilaway as opt-in local target

**Choice**: Update the `lint` Makefile target to run `golangci-lint run ./...` in addition to `go vet ./...`. Add a separate `make nilaway` target for local opt-in use.

**Rationale**: Developer experience must mirror CI — if `make lint` is green locally, the CI lint job should be green too. Nilaway is excluded from the default `lint` target due to install overhead and runtime cost, but surfaced as an explicit target so developers can run it before pushing if desired.

**Alternatives considered**:
- nilaway in `make lint`: Too slow for routine use; breaks the fast feedback loop.
