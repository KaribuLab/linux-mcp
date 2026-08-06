## ADDED Requirements

### Requirement: Repository ships a one-line shell installer

The repository MUST include an executable shell script at `install.sh` (root of the repo) that, when invoked as `curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/install.sh | sudo sh`, performs a complete systemd install of linux-mcp: it downloads the binary from GitHub Releases, validates its SHA256, installs it under `/usr/local/bin/linux-mcp`, creates the `mcp-agent` user and `mcp-admin` group if missing, downloads the reference unit from `main`, enables and starts the service, and verifies the final state.

The script MUST be written in **POSIX `sh`** (shebang `#!/bin/sh`) and MUST NOT require bash or any non-POSIX shell extension. Concretely the script MUST NOT use `[[ ]]`, `local`, arrays, `pipefail`, `${var,,}`, `${var^^}`, process substitution, or any other bashism. The script MUST be marked executable in git (`chmod +x`) and MUST NOT depend on any tool beyond `curl`, `getent`, `sha256sum`, `install`, `systemctl`, `mktemp`, `stat`, `trap`, `command`, `case`, `printf`, `set`, and other POSIX-mandated utilities that ship with any modern Linux distribution.

#### Scenario: Script present and executable in repo root
- **WHEN** an operator clones or browses the repository
- **THEN** `install.sh` MUST exist at the repo root and MUST be executable (`-rwxr-xr-x` or equivalent)

#### Scenario: One-liner downloads and executes
- **WHEN** an operator runs `curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/install.sh | sudo sh` on a supported host
- **THEN** the script MUST be downloaded and executed with root privileges and MUST proceed without prompting the operator for input

### Requirement: Installer is idempotent

Re-running the installer on a host where linux-mcp is already installed MUST be safe: it MUST NOT fail if `mcp-agent` or `mcp-admin` already exist, it MUST be able to replace an existing `/usr/local/bin/linux-mcp`, it MUST be able to replace an existing `/etc/systemd/system/linux-mcp.service`, and it MUST leave any existing members of the `mcp-admin` group untouched. Re-running it with the default version MUST pick up the latest published release and replace both the binary and the unit, then `daemon-reload` and restart the service so the new binary takes effect.

#### Scenario: Re-run on already-installed host
- **WHEN** the script is invoked a second time on a host where the service is already running
- **THEN** it MUST exit with status `0` (assuming no other failure), the service MUST be running the newly installed binary, and any pre-existing members of `mcp-admin` MUST still be members

#### Scenario: Existing unit is replaced on re-run
- **WHEN** the script is invoked a second time and `deploy/systemd/linux-mcp.service` on `main` differs from the version currently installed
- **THEN** the unit file under `/etc/systemd/system/` MUST be replaced, `systemctl daemon-reload` MUST run, and the service MUST be restarted

### Requirement: Installer detects target version and architecture

The script MUST resolve the version to install as follows: if the environment variable `LINUX_MCP_VERSION` is set to a non-empty string matching the `vX.Y.Z` pattern, that value is used; otherwise the script MUST follow the redirect from `https://github.com/KaribuLab/linux-mcp/releases/latest` and parse the resulting tag (so the latest published release is used without consuming GitHub API rate limit).

The script MUST determine the target architecture from `uname -m`: `x86_64` maps to `amd64`, `aarch64` maps to `arm64`. Any other value MUST cause the script to abort with a non-zero exit and a clear message that points the operator to building from source. The `aarch64` mapping MUST cover Raspberry Pi 3, 4, 5 and Zero 2 W running Raspberry Pi OS 64-bit (Bookworm or later, or any other 64-bit ARMv8 distribution such as Ubuntu Server for ARM64), because those report `aarch64` from `uname -m` and the published `linux-mcp-linux-arm64` asset runs on them.

#### Scenario: Default version is latest release
- **WHEN** `LINUX_MCP_VERSION` is unset or empty
- **THEN** the script MUST install the latest published release tag as of script execution

#### Scenario: Explicit version override
- **WHEN** `LINUX_MCP_VERSION=v0.10.0` is set in the environment
- **THEN** the script MUST install release `v0.10.0` (or abort if that release does not exist)

#### Scenario: Raspberry Pi 3/4/5 with 64-bit OS installs via arm64
- **WHEN** the script runs on a Raspberry Pi 3, 4, 5 or Zero 2 W with Raspberry Pi OS 64-bit (or any distribution reporting `aarch64`)
- **THEN** the script MUST download and install the `linux-mcp-linux-arm64` asset and the service MUST start successfully

#### Scenario: Unsupported architecture
- **WHEN** `uname -m` returns something other than `x86_64` or `aarch64` (for example `armv7l`, `armv6l`, `i386`, `riscv64`)
- **THEN** the script MUST exit non-zero with a message explaining that the supported architectures are `amd64` (for `x86_64`) and `arm64` (for `aarch64`, which covers Raspberry Pi 3/4/5 with 64-bit OS), and that other architectures require building from source

### Requirement: Installer validates binary SHA256 before installing

The script MUST download the binary asset and the `SHA256SUMS` file from the resolved release into a temporary directory, run `sha256sum --ignore-missing -c SHA256SUMS` against the downloaded binary, and only on success copy the binary to its final destination with mode `0755`. If the validation fails, the script MUST NOT modify `/usr/local/bin/linux-mcp`, MUST remove the temporary directory, and MUST exit non-zero with a message identifying the failed asset.

#### Scenario: Valid hash, binary installed
- **WHEN** the downloaded `SHA256SUMS` validates the downloaded binary
- **THEN** the script MUST install the binary at `/usr/local/bin/linux-mcp` with mode `0755` and the temporary directory MUST be removed before exit

#### Scenario: Invalid hash, no install
- **WHEN** the downloaded `SHA256SUMS` does NOT validate the downloaded binary
- **THEN** the script MUST exit non-zero, MUST NOT modify `/usr/local/bin/linux-mcp`, and MUST print a message identifying the failed asset

### Requirement: Installer provisions OS user and group idempotently

The script MUST ensure the system group `mcp-admin` and the system user `mcp-agent` (with home `/nonexistent` and shell `/usr/sbin/nologin`) exist before it enables the service, because the unit declares them as primary and supplementary groups respectively and `serve` will fail to start otherwise. The script MUST use `getent` to test for existence and MUST NOT add any operator to `mcp-admin` (that membership is the operator's responsibility).

#### Scenario: Group and user created on a fresh host
- **WHEN** the script runs on a host without `mcp-admin` or `mcp-agent`
- **THEN** both MUST be created as system accounts with the documented options, and the group MUST exist before `systemctl enable --now linux-mcp.service` is invoked

#### Scenario: Existing group and user left untouched
- **WHEN** the script runs on a host that already has `mcp-admin` and `mcp-agent`
- **THEN** neither MUST be recreated and any existing members of `mcp-admin` MUST be preserved

### Requirement: Installer installs unit, reloads systemd, and starts the service

The script MUST download `deploy/systemd/linux-mcp.service` from `main` (not from the release tag), install it under `/etc/systemd/system/linux-mcp.service` with mode `0644`, run `systemctl daemon-reload`, and run `systemctl enable --now linux-mcp.service`. The unit MUST always come from `main` regardless of the release being installed, because the unit is deployment configuration rather than a release artifact.

#### Scenario: Unit installed from main and service started
- **WHEN** the script runs on a supported host with systemd
- **THEN** `/etc/systemd/system/linux-mcp.service` MUST contain the contents of `deploy/systemd/linux-mcp.service` on `main`, and `systemctl is-active linux-mcp.service` MUST report `active`

### Requirement: Installer verifies final state

After starting the service, the script MUST verify both that the HTTP endpoint on `127.0.0.1:5000` returns HTTP `401` (the canonical "alive and protected" signal) and that `/run/linux-mcp/issue.sock` exists with mode `0660`, owner `mcp-agent`, and group `mcp-admin`. If either check fails after a bounded poll (the endpoint check MUST tolerate a few seconds for startup), the script MUST exit non-zero with a diagnostic pointing at the Troubleshooting section of the install runbook.

#### Scenario: Endpoint returns 401
- **WHEN** the service has finished starting
- **THEN** `curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:5000` MUST eventually print `401` within the poll budget

#### Scenario: Issuance socket has correct ownership
- **WHEN** the service has finished starting
- **THEN** `/run/linux-mcp/issue.sock` MUST exist and `stat -c '%a %U %G' /run/linux-mcp/issue.sock` MUST print `660 mcp-agent mcp-admin`

### Requirement: Installer aborts on missing prerequisites

The script MUST abort with a non-zero exit and a clear message if any of the following are missing: `systemctl` (the host has no systemd), `curl`, `sha256sum`, `getent`, `mktemp`, or write access to `/usr/local/bin` and `/etc/systemd/system`. The script MUST also refuse to run if it is not invoked as root (effective UID MUST be `0`).

#### Scenario: Non-root invocation
- **WHEN** the script is invoked without root privileges
- **THEN** it MUST exit non-zero with a message instructing the operator to use `sudo sh` and MUST NOT modify the system

#### Scenario: Host without systemd
- **WHEN** `systemctl` is not on `PATH`
- **THEN** the script MUST exit non-zero with a message that the installer requires systemd, and MUST NOT attempt to install the binary or unit

### Requirement: README and runbook point to the one-liner

`README.md` and `docs/runbooks/install-systemd.md` MUST expose the one-liner as the recommended install path. The runbook MUST keep the manual steps as a reference (so operators who cannot pipe to bash, or who need to understand each step, can still follow them) and MUST link to the installer script. The runbook MUST also document that the unit installed by the script comes from `main`, so a breaking change to the unit can affect an existing deploy.

#### Scenario: README shows the one-liner
- **WHEN** a reader opens `README.md`
- **THEN** there MUST be a visible install section containing the exact `curl -fsSL … | sudo sh` command

#### Scenario: Runbook promotes the one-liner and keeps manual steps
- **WHEN** an operator reads `docs/runbooks/install-systemd.md`
- **THEN** the runbook MUST lead with the one-liner, MUST still include the manual steps for reference, and MUST warn that the unit installed by the script comes from `main`

### Requirement: CI lints the installer with shellcheck

`.github/workflows/ci.yml` MUST include a step that runs `shellcheck -s sh install.sh` on every push and pull request, and the build MUST fail if shellcheck reports any issue at severity `error` or `warning`. Using `-s sh` is mandatory so the linter enforces POSIX compliance (no `[[ ]]`, `local`, arrays, `pipefail`, etc.).

#### Scenario: shellcheck passes
- **WHEN** a push or pull request is made to the repository and `install.sh` is free of POSIX shellcheck errors and warnings
- **THEN** the CI build MUST pass the shellcheck step

#### Scenario: shellcheck fails
- **WHEN** `install.sh` contains a construct flagged by `shellcheck -s sh` at severity `error` or `warning` (for example a `[[ … ]]` test or a `local` declaration)
- **THEN** the CI build MUST fail and report the shellcheck diagnostic