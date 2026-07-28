## MODIFIED Requirements

### Requirement: Systemd unit file shipped in repository
The repository MUST include a reference systemd unit at `deploy/systemd/linux-mcp.service` that runs the service as user `mcp-agent`, sets `ExecStart` to the documented binary path **invoking the `serve` subcommand**, grants only `CAP_DAC_READ_SEARCH` via ambient/bounding capabilities, and applies write-hardening directives suitable for a read-oriented ops agent.

The unit MUST additionally provide a runtime directory for the issuance socket, grant the service the administrative group as a supplementary group so it can set the socket's group ownership, and prevent core dumps because the token signing key lives only in process memory.

#### Scenario: Unit present with elevated read capability
- **WHEN** an operator inspects `deploy/systemd/linux-mcp.service`
- **THEN** the unit MUST specify `User=mcp-agent`, `AmbientCapabilities=CAP_DAC_READ_SEARCH`, and `CapabilityBoundingSet=CAP_DAC_READ_SEARCH`

#### Scenario: Unit invokes the serve subcommand
- **WHEN** an operator inspects the unit's `ExecStart`
- **THEN** it MUST invoke the binary with the `serve` subcommand, because the bare binary prints help and exits

#### Scenario: Unit provisions the issuance socket directory
- **WHEN** an operator inspects the unit
- **THEN** it MUST declare a runtime directory for the socket and MUST grant `mcp-admin` as a supplementary group

#### Scenario: Core dumps disabled
- **WHEN** an operator inspects the unit
- **THEN** it MUST disable core dumps so the in-memory signing key cannot be written to disk by a crash

### Requirement: Install runbook documents OS setup
The repository MUST include `docs/runbooks/install-systemd.md` that documents creating the OS user/group, installing the binary, installing and enabling the unit, verifying the service, updating, and uninstalling. The runbook MUST state that in-process read policy remains required even when systemd is not used.

The runbook MUST additionally document creating the `mcp-admin` group, adding operators to it, and the fact that group membership only takes effect after the user opens a new session. The update procedure MUST warn that the new unit has to be installed before restarting, because the previous `ExecStart` without a subcommand no longer starts the service.

#### Scenario: Runbook covers user and enable steps
- **WHEN** an operator follows `docs/runbooks/install-systemd.md`
- **THEN** the document MUST include commands to create `mcp-agent`, install the unit under `/etc/systemd/system/`, and `enable --now` the service

#### Scenario: Runbook covers the administrative group
- **WHEN** an operator follows the runbook
- **THEN** it MUST include commands to create the `mcp-admin` group and add a user to it, and MUST state that the user has to re-login for the membership to apply

#### Scenario: Troubleshooting covers permission denied on the socket
- **WHEN** an operator hits a permission error running `linux-mcp auth`
- **THEN** the runbook troubleshooting section MUST list stale group membership as a cause and re-login as the remedy

## ADDED Requirements

### Requirement: Runbook documents obtaining a token and connecting a client
The install runbook MUST document how an operator obtains a token with `linux-mcp auth`, how to reach the server through an SSH tunnel to loopback, and how to supply the token to an MCP client as an `Authorization: Bearer` header. It MUST state that tokens are invalidated when the service restarts.

#### Scenario: Operator can go from install to connected client
- **WHEN** an operator follows the runbook end to end
- **THEN** it MUST show issuing a token, establishing the SSH tunnel to the loopback port, and configuring the client with the bearer header

#### Scenario: Restart behaviour documented
- **WHEN** an operator reads the runbook
- **THEN** it MUST state that restarting the service invalidates every previously issued token
