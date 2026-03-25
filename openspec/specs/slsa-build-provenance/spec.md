## ADDED Requirements

### Requirement: SLSA provenance generation on release
The CI/CD pipeline SHALL generate SLSA provenance attestations for all release artifacts (binaries and container image) whenever a `v*` tag is pushed, using the `slsa-framework/slsa-github-generator` reusable workflow.

#### Scenario: Provenance generated for release tag
- **WHEN** a Git tag matching `v*` is pushed to the repository
- **THEN** the release workflow SHALL invoke `slsa-github-generator` to produce a signed SLSA provenance file (`.intoto.jsonl`) for each release binary

#### Scenario: Provenance attached to GitHub release
- **WHEN** the provenance file is produced successfully
- **THEN** it SHALL be uploaded as an asset to the corresponding GitHub release, co-located with the binary it attests

#### Scenario: Provenance not generated for non-release commits
- **WHEN** a commit is pushed to a branch (not a `v*` tag)
- **THEN** the provenance generation step SHALL NOT run and no attestation SHALL be produced

### Requirement: Workflow action SHA pinning
All GitHub Actions workflow files in the repository SHALL reference action dependencies using their full commit SHA, not mutable version tags.

#### Scenario: Action reference uses full SHA
- **WHEN** a workflow file references an external action (e.g., `actions/checkout`)
- **THEN** the `uses:` field SHALL contain the full 40-character commit SHA (e.g., `actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af68`) with a comment indicating the human-readable version

#### Scenario: PR CI check for mutable action references
- **WHEN** a pull request modifies a workflow file
- **THEN** CI SHALL fail if any `uses:` reference does not resolve to a full SHA

### Requirement: Release workflow isolation
The release workflow producing attestations SHALL use dedicated job-level permissions limited to what is strictly required for signing and publishing.

#### Scenario: Provenance job has minimal permissions
- **WHEN** the provenance generation job runs
- **THEN** it SHALL have `id-token: write` (for OIDC signing) and `contents: write` (for release asset upload) and no broader permissions

#### Scenario: Build job cannot access signing credentials
- **WHEN** the build job compiles the binary
- **THEN** it SHALL have `contents: read` only; the signing token SHALL be issued only to the provenance job

### Requirement: SLSA provenance verifiability
End users SHALL be able to verify the provenance of any release binary using standard open-source tooling without proprietary dependencies.

#### Scenario: Verification with slsa-verifier
- **WHEN** a user runs `slsa-verifier verify-artifact <binary> --provenance-path <provenance.intoto.jsonl> --source-uri github.com/<owner>/<repo>`
- **THEN** the tool SHALL exit 0 and print `PASSED: Verified SLSA provenance` for an unmodified artifact from a legitimate release

#### Scenario: Tampered artifact fails verification
- **WHEN** a user runs the same verification command against a binary that has been modified after signing
- **THEN** `slsa-verifier` SHALL exit non-zero with a descriptive error indicating the digest mismatch
