## Purpose

Continuous integration, release publishing, and build visibility for linux-mcp.

## Requirements

### Requirement: Continuous integration workflow in the repository
The repository MUST include a GitHub Actions workflow that builds the module, runs `go vet`, and runs the test suite. The workflow MUST use the Go version declared in `go.mod` rather than a hardcoded one, so the pipeline follows the module as it is upgraded.

#### Scenario: Workflow runs on changes to main
- **WHEN** a commit is pushed to `main` or a pull request targets it
- **THEN** the workflow MUST run build, vet, and tests

#### Scenario: Failing tests fail the pipeline
- **WHEN** any test in the module fails
- **THEN** the workflow MUST report failure

### Requirement: Systemd unit validated statically in CI
The workflow MUST validate the reference systemd unit with `systemd-analyze verify`, so that a malformed or unknown directive is caught without provisioning a host.

#### Scenario: Malformed unit caught by the pipeline
- **WHEN** `deploy/systemd/linux-mcp.service` contains an invalid directive
- **THEN** the workflow MUST report failure

### Requirement: Build status visible in the README
The project `README.md` MUST display the workflow status badge so the health of `main` is visible without opening the Actions tab.

#### Scenario: README shows the badge
- **WHEN** a reader opens the project README
- **THEN** it MUST contain the workflow status badge

### Requirement: Binaries built for both Linux architectures
The pipeline MUST build the binary for `linux/amd64` and `linux/arm64`. It MUST NOT build for macOS or Windows, because token issuance depends on `SO_PEERCRED`, which is Linux specific, and the tools depend on Linux filesystem semantics.

#### Scenario: Both architectures produced
- **WHEN** the pipeline builds release binaries
- **THEN** it MUST produce one binary for `linux/amd64` and one for `linux/arm64`

#### Scenario: Non-Linux targets removed
- **WHEN** an operator inspects the build configuration
- **THEN** there MUST be no build target for macOS or Windows

### Requirement: Binaries attached to public releases
Pushing a version tag MUST publish a GitHub Release with both architecture binaries attached, so anyone can download them without a GitHub account and the download URL remains stable. The release MUST include a `SHA256SUMS` file covering every attached binary.

#### Scenario: Tag produces a downloadable release
- **WHEN** a version tag is pushed
- **THEN** a release MUST be published with both binaries and a `SHA256SUMS` file attached

#### Scenario: Checksum verifies the download
- **WHEN** a user downloads a binary and the `SHA256SUMS` file from a release
- **THEN** the recorded checksum MUST match the downloaded binary

### Requirement: Pull request builds available as artifacts
The pipeline MUST upload the built binaries as workflow artifacts on every run, so a reviewer can test the binary produced by a pull request before it is merged.

#### Scenario: Artifacts available for a pull request run
- **WHEN** the workflow runs for a pull request
- **THEN** the built binaries MUST be uploaded as artifacts of that run

### Requirement: Released binaries report their version
The release build MUST inject the tag into the binary so that a downloaded binary can be identified. Running the binary with `--version` MUST print the version it was built from.

#### Scenario: Downloaded binary identifies itself
- **WHEN** a user runs `--version` on a binary downloaded from a release
- **THEN** it MUST print the version of the tag that produced it
