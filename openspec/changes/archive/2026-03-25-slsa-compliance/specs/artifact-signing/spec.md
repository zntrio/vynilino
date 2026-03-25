## ADDED Requirements

### Requirement: Container image signing with cosign
The release pipeline SHALL sign the published container image using cosign keyless signing (Sigstore) with the GitHub Actions OIDC workload identity as the signing identity.

#### Scenario: Image signed after push to registry
- **WHEN** a container image is pushed to the registry as part of a `v*` release
- **THEN** the workflow SHALL run `cosign sign --yes <image-digest>` to create a signature entry in the Sigstore Rekor transparency log and attach the signature to the image in the registry

#### Scenario: Signature references image digest, not tag
- **WHEN** the image is signed
- **THEN** the cosign signature SHALL be bound to the immutable image digest (e.g., `ghcr.io/owner/repo@sha256:<digest>`) not the mutable tag, ensuring the signature cannot be reused for a different image

#### Scenario: Image signing fails if OIDC token unavailable
- **WHEN** the GitHub Actions OIDC token cannot be obtained (e.g., workflow not triggered by a tag push or OIDC disabled)
- **THEN** the signing step SHALL fail with a non-zero exit code and the release workflow SHALL halt without publishing an unsigned image as a "signed" release

### Requirement: Keyless signature verification
Any user with `cosign` installed SHALL be able to verify the authenticity of a released container image without requiring access to a private key or key management system.

#### Scenario: Successful image verification
- **WHEN** a user runs `cosign verify --certificate-identity-regexp="https://github.com/<owner>/<repo>/.github/workflows/release.yml@refs/tags/v.*" --certificate-oidc-issuer="https://token.actions.githubusercontent.com" <image-ref>`
- **THEN** cosign SHALL return exit 0 and print the verified signature payload confirming the image was built by the expected workflow

#### Scenario: Unsigned or tampered image fails verification
- **WHEN** a user runs the same `cosign verify` command against an image that was not signed by the release workflow
- **THEN** cosign SHALL return a non-zero exit code with an error indicating no matching signatures were found

### Requirement: Signing identity documentation
The repository SHALL document the exact cosign verification command and the expected certificate identity so that operators can verify images without guessing parameters.

#### Scenario: SECURITY.md contains verification command
- **WHEN** a user reads `SECURITY.md` at the repository root
- **THEN** they SHALL find a copy-paste–ready `cosign verify` command with the correct `--certificate-identity-regexp` and `--certificate-oidc-issuer` values for this repository

#### Scenario: README references SECURITY.md
- **WHEN** a user reads `README.md`
- **THEN** they SHALL find a link or brief mention directing them to `SECURITY.md` for supply-chain verification instructions
