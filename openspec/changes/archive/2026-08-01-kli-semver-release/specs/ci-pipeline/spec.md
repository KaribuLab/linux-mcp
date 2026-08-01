## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Binaries attached to public releases
Publishing a version (via the kli-driven flow on `main`, or an equivalent tag-based release path) MUST produce a GitHub Release with both architecture binaries attached, so anyone can download them without a GitHub account and the download URL remains stable. The release MUST include a `SHA256SUMS` file covering every attached binary. Because events authenticated only with `GITHUB_TOKEN` do not trigger other workflows, binary build and `gh release create` (or upload) MUST run in the same workflow run that creates the tag, not in a separate workflow that only listens for `push` of tags.

#### Scenario: Tag produces a downloadable release
- **WHEN** the pipeline creates a new version tag on `main`
- **THEN** a release MUST be published in that same run with both binaries and a `SHA256SUMS` file attached

#### Scenario: Checksum verifies the download
- **WHEN** a user downloads a binary and the `SHA256SUMS` file from a release
- **THEN** the recorded checksum MUST match the downloaded binary
