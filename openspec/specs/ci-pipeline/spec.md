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
Publishing a version (via the kli-driven flow on `main`, or an equivalent tag-based release path) MUST produce a GitHub Release with both architecture binaries attached, so anyone can download them without a GitHub account and the download URL remains stable. The release MUST include a `SHA256SUMS` file covering every attached binary. Because events authenticated only with `GITHUB_TOKEN` do not trigger other workflows, binary build and `gh release create` (or upload) MUST run in the same workflow run that creates the tag, not in a separate workflow that only listens for `push` of tags.

#### Scenario: Tag produces a downloadable release
- **WHEN** the pipeline creates a new version tag on `main`
- **THEN** a release MUST be published in that same run with both binaries and a `SHA256SUMS` file attached

#### Scenario: Checksum verifies the download
- **WHEN** a user downloads a binary and the `SHA256SUMS` file from a release
- **THEN** the recorded checksum MUST match the downloaded binary

### Requirement: Semantic versioning with kli on main
On every push to `main`, after verification succeeds, the pipeline MUST compute the next semantic version with KaribuLab `kli` (`kli semver`) from Conventional Commit messages (`feat:` → minor, `fix:` → patch, breaking → major). Authentication MUST use the job `GITHUB_TOKEN` with `contents: write` (and MAY declare `id-token: write`). The pipeline MUST NOT require a personal access token or SSH deploy key for tagging or publishing.

#### Scenario: New commits imply a newer version than the latest tag
- **WHEN** a push to `main` passes verification and `kli semver` reports a version different from the latest git tag (or there is no tag yet)
- **THEN** the pipeline MUST create the version tag with `kli semver -t` and publish a GitHub Release for that tag in the same workflow run

#### Scenario: No version bump
- **WHEN** a push to `main` passes verification and `kli semver` equals the latest git tag
- **THEN** the pipeline MUST NOT create a new tag and MUST NOT publish a new Release

#### Scenario: Pull requests do not release
- **WHEN** the workflow runs for a pull request
- **THEN** the pipeline MUST NOT create tags and MUST NOT publish a GitHub Release

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
