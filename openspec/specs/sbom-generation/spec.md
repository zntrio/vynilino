## ADDED Requirements

### Requirement: SPDX SBOM generation for each release
The release pipeline SHALL generate a Software Bill of Materials in SPDX JSON format for each release, covering both the Go module dependency graph and the container image layers.

#### Scenario: SBOM generated for release container image
- **WHEN** a `v*` release tag is pushed and the container image is built
- **THEN** `syft` SHALL scan the image and produce an SPDX JSON SBOM file (`sbom.spdx.json`) listing all OS packages and Go module dependencies present in the image

#### Scenario: SBOM covers Go module dependencies
- **WHEN** the SBOM is generated
- **THEN** it SHALL include all direct and transitive Go module dependencies (from `go.sum`) with their name, version, and license metadata where available

#### Scenario: SBOM generation does not fail the release on partial data
- **WHEN** license metadata is unavailable for one or more packages
- **THEN** the SBOM SHALL still be generated with available metadata and the missing fields SHALL be left empty per the SPDX spec (NOASSERTION)

### Requirement: SBOM attached to GitHub release
The SBOM SHALL be published as a downloadable asset on every GitHub release so that operators can inspect dependency information without pulling the container image.

#### Scenario: SBOM uploaded as release asset
- **WHEN** the SBOM file is produced
- **THEN** it SHALL be uploaded to the corresponding GitHub release with the filename `<repo>-<version>-sbom.spdx.json`

#### Scenario: SBOM release asset present before release is published
- **WHEN** the GitHub release is finalized (status changes from draft to published)
- **THEN** the SBOM asset SHALL already be attached, not added in a subsequent step

### Requirement: SBOM attached as OCI attestation
In addition to the release asset, the SBOM SHALL be attached to the container image in the registry as a cosign attestation using the `cyclonedx` predicate type for maximum tooling compatibility.

#### Scenario: SBOM attestation pushed to registry
- **WHEN** the container image is pushed and the SBOM is generated
- **THEN** `cosign attest --yes --predicate sbom.spdx.json --type spdxjson <image-digest>` SHALL be executed to attach the SBOM as an OCI attestation in the same repository

#### Scenario: SBOM attestation verifiable with cosign
- **WHEN** a user runs `cosign verify-attestation --type spdxjson --certificate-identity-regexp="..." --certificate-oidc-issuer="..." <image-ref>`
- **THEN** cosign SHALL return exit 0 and print the SBOM JSON payload, confirming the attestation was produced by the release workflow

### Requirement: SBOM version matches release version
The SBOM document SHALL carry the release version as its document name and namespace to enable unambiguous correlation between a release artifact and its bill of materials.

#### Scenario: SBOM document name matches release tag
- **WHEN** a release is tagged `v1.2.3`
- **THEN** the SBOM `name` field SHALL be `vynilino-v1.2.3` and the `documentNamespace` SHALL be unique (e.g., containing the Git SHA or a UUID)
