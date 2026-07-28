## Purpose

Reference systemd unit and install runbook so operators can deploy linux-mcp as a hardened read-oriented service without inventing the unit from scratch.

## Requirements

### Requirement: Systemd unit file shipped in repository
The repository MUST include a reference systemd unit at `deploy/systemd/linux-mcp.service` that runs the service as user `mcp-agent`, sets `ExecStart` to the documented binary path, grants only `CAP_DAC_READ_SEARCH` via ambient/bounding capabilities, and applies write-hardening directives suitable for a read-oriented ops agent.

#### Scenario: Unit present with elevated read capability
- **WHEN** an operator inspects `deploy/systemd/linux-mcp.service`
- **THEN** the unit MUST specify `User=mcp-agent`, `AmbientCapabilities=CAP_DAC_READ_SEARCH`, and `CapabilityBoundingSet=CAP_DAC_READ_SEARCH`

### Requirement: Install runbook documents OS setup
The repository MUST include `docs/runbooks/install-systemd.md` that documents creating the OS user/group, installing the binary, installing and enabling the unit, verifying the service, updating, and uninstalling. The runbook MUST state that in-process read policy remains required even when systemd is not used.

#### Scenario: Runbook covers user and enable steps
- **WHEN** an operator follows `docs/runbooks/install-systemd.md`
- **THEN** the document MUST include commands to create `mcp-agent`, install the unit under `/etc/systemd/system/`, and `enable --now` the service

### Requirement: Docs index links the runbook
`docs/README.md` and the project `README.md` MUST link to the systemd install runbook so operators can discover it.

#### Scenario: README points to runbook
- **WHEN** a reader opens the project README or docs index
- **THEN** there MUST be a link to `docs/runbooks/install-systemd.md` (or equivalent relative path)
