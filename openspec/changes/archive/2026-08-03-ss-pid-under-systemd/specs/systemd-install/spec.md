## MODIFIED Requirements

### Requirement: Systemd unit file shipped in repository
The repository MUST include a reference systemd unit at `deploy/systemd/linux-mcp.service` that runs the service as user `mcp-agent`, sets `ExecStart` to the documented binary path **invoking the `serve` subcommand**, grants `CAP_DAC_READ_SEARCH` and `CAP_SYS_PTRACE` via ambient and bounding capabilities (and MUST NOT grant unrelated capabilities), and applies write-hardening directives suitable for a read-oriented ops agent.

The unit MUST NOT set `ProtectProc=invisible` (or any ProtectProc mode that hides other users' `/proc` entries from the service), so that socket owner resolution and process inventory can see foreign PIDs. Omitting `ProtectProc=` (systemd default) is acceptable.

The unit MUST additionally provide a runtime directory for the issuance socket, grant the service the administrative group as a supplementary group so it can set the socket's group ownership, and prevent core dumps because the token signing key lives only in process memory.

#### Scenario: Unit present with elevated read and ptrace-class capabilities
- **WHEN** an operator inspects `deploy/systemd/linux-mcp.service`
- **THEN** the unit MUST specify `User=mcp-agent`, and both `AmbientCapabilities` and `CapabilityBoundingSet` MUST include `CAP_DAC_READ_SEARCH` and `CAP_SYS_PTRACE`

#### Scenario: Unit does not hide foreign proc entries
- **WHEN** an operator inspects `deploy/systemd/linux-mcp.service`
- **THEN** the unit MUST NOT set `ProtectProc=invisible`

#### Scenario: Unit invokes the serve subcommand
- **WHEN** an operator inspects the unit's `ExecStart`
- **THEN** it MUST invoke the binary with the `serve` subcommand, because the bare binary prints help and exits

#### Scenario: Unit provisions the issuance socket directory
- **WHEN** an operator inspects the unit
- **THEN** it MUST declare a runtime directory for the socket and MUST grant `mcp-admin` as a supplementary group

#### Scenario: Core dumps disabled
- **WHEN** an operator inspects the unit
- **THEN** it MUST disable core dumps so the in-memory signing key cannot be written to disk by a crash

## ADDED Requirements

### Requirement: Runbook documents socket Pid resolution privileges and risk
The install runbook (`docs/runbooks/install-systemd.md`) MUST document that the reference unit grants `CAP_SYS_PTRACE` and does not use `ProtectProc=invisible` so `ss`/`ss_grep` can resolve socket inodes to Pid/Process for processes of other users via `/proc/*/fd`. It MUST state the accepted risk: a valid Bearer token can obtain that process-owner inventory through the MCP API (read-only tools; no memory-dump tool). It MUST include a verification step (or equivalent troubleshooting) that after installing the unit, `ss` with Pid visible shows a non-empty Pid for at least one listening socket owned by a non-`mcp-agent` process when such a socket exists on the host.

#### Scenario: Runbook explains why ptrace-class cap is present
- **WHEN** an operator reads the install runbook sections covering the unit hardening or capabilities
- **THEN** the document MUST mention `CAP_SYS_PTRACE` together with socket Pid/Process resolution and MUST mention the Bearer risk in neutral Spanish or clear operational language

#### Scenario: Runbook verification covers foreign Pid
- **WHEN** an operator follows the post-install or update verification guidance
- **THEN** the runbook MUST instruct checking that Pid/Process on `ss` (or `ss_grep`) is populated for a foreign listening socket when one exists
