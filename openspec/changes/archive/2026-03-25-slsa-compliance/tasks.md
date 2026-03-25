## 1. CI Workflow

- [x] 1.1 Create `.github/workflows/ci.yml` triggering on `push` (main) and `pull_request`
- [x] 1.2 Add `build` job: `actions/checkout` + `actions/setup-go` (with `go-version-file: go.mod`) + `go build ./...` + `go test ./...` (all SHA-pinned)
- [x] 1.3 Add `lint` job: `golangci/golangci-lint-action` (SHA-pinned) with default config — no `.golangci.yml` needed
- [x] 1.4 Add `nilaway` job: install `go.uber.org/nilaway/cmd/nilaway` via `go install`, run `nilaway ./...` with `continue-on-error: true` so failures are visible but non-blocking
- [x] 1.5 Add `zizmor` job: path-filtered to `.github/workflows/**`; run `zizmor .` via `uvx` or binary install (blocking)
- [x] 1.6 Update `make lint` to run `golangci-lint run ./...` alongside existing `go vet ./...`
- [x] 1.7 Add `make nilaway` target: `go install go.uber.org/nilaway/cmd/nilaway@latest && nilaway ./...`

## 2. Workflow Hardening (SHA Pinning)

- [x] 2.1 Audit all `.github/workflows/*.yml` files and replace mutable action tags with full commit SHAs (add version comment alongside each SHA)
- [x] 2.2 Ensure all actions introduced in the CI workflow (section 1) are SHA-pinned at creation time

## 3. Release Workflow — Goreleaser & Provenance

- [x] 3.1 Create `.goreleaser.yaml` at repo root: define builds for `linux/amd64`, `linux/arm64`, `darwin/arm64` with `CGO_ENABLED=0`; configure archives, checksum file, and GitHub release settings
- [x] 3.2 Create `.github/workflows/release.yml` triggered on `push: tags: ['v*']`
- [x] 3.3 Add a `goreleaser` job using `goreleaser/goreleaser-action` (SHA-pinned, OSS tier — no `GORELEASER_KEY` required) that runs `goreleaser release --clean`
- [x] 3.4 Add a `provenance` job using `slsa-framework/slsa-github-generator` Go reusable workflow (`generator_go_slsa3.yml`, SHA-pinned), wired to the goreleaser artifacts for provenance coverage
- [x] 3.5 Confirm the `.intoto.jsonl` provenance file is uploaded as a GitHub release asset by the generator automatically

## 4. Container Image Build & Signing

- [x] 4.1 Add a `docker-build` job in `release.yml` that builds and pushes the container image to GHCR using `docker/build-push-action` (SHA-pinned), outputting the image digest
- [x] 4.2 Install `cosign` via `sigstore/cosign-installer` (SHA-pinned) in the signing job
- [x] 4.3 Run `cosign sign --yes <image>@<digest>` to produce a keyless signature bound to the GitHub Actions OIDC identity
- [x] 4.4 Verify signing permissions: ensure the signing job has `id-token: write` and `packages: write` only

## 5. SBOM Generation & Attestation

- [x] 5.1 Add `anchore/syft-action` (SHA-pinned) to the release workflow to scan the built container image and produce `sbom.spdx.json`
- [x] 5.2 Set the SBOM `documentName` to `vynilino-<version>` using the `--output` and `--name` flags
- [x] 5.3 Upload `sbom.spdx.json` as a GitHub release asset named `vynilino-<version>-sbom.spdx.json`
- [x] 5.4 Run `cosign attest --yes --predicate sbom.spdx.json --type spdxjson <image>@<digest>` to attach the SBOM as an OCI attestation

## 6. Documentation

- [x] 6.1 Create `SECURITY.md` at the repository root with: trust model overview, copy-paste `slsa-verifier` command for binary verification, copy-paste `cosign verify` command for image verification, copy-paste `cosign verify-attestation` command for SBOM, and tool installation links
- [x] 6.2 Update `README.md` with a SLSA Level 2 badge and a one-liner linking to `SECURITY.md` for verification instructions

## 7. Validation

- [ ] 7.1 Open a test PR and verify the `build`, `lint`, `nilaway`, and `zizmor` CI jobs all run as expected
- [ ] 7.2 Trigger a test release (e.g., `v0.0.1-rc1`) and verify all release workflow jobs succeed
- [ ] 7.3 Run `slsa-verifier verify-artifact` against the produced binary and provenance file locally — confirm exit 0
- [ ] 7.4 Run `cosign verify` against the signed image — confirm exit 0 with expected certificate identity
- [ ] 7.5 Run `cosign verify-attestation --type spdxjson` against the image — confirm SBOM payload is returned
- [ ] 7.6 Delete the test release and tag after validation
