## Purpose

Unix-socket token issuance, signing key lifecycle, and issuance audit.

## Requirements

### Requirement: Tokens issued over a filesystem unix socket
The `serve` command MUST expose a token issuance endpoint on a unix domain socket at a filesystem path (default `/run/linux-mcp/issue.sock`). The socket MUST NOT use the Linux abstract namespace, because abstract sockets carry no permission checks. The socket MUST be created with mode `0660`, owned by the service user and by the administrative group (default `mcp-admin`), and the permissions MUST be in effect from the moment the socket becomes connectable.

#### Scenario: Socket created with restrictive permissions
- **WHEN** `serve` starts successfully
- **THEN** the issuance socket MUST exist as a filesystem path with mode `0660` and group ownership set to the configured administrative group

#### Scenario: Non-member cannot connect
- **WHEN** a local user who is not a member of the administrative group runs `linux-mcp auth`
- **THEN** the connection MUST fail with a permission error and no token MUST be issued

#### Scenario: Member obtains a token
- **WHEN** a local user who is a member of the administrative group runs `linux-mcp auth`
- **THEN** the server MUST return a signed token for that user

#### Scenario: Administrative group is optional
- **WHEN** `serve` is started with an empty administrative group, as in local development where the server runs as the developer's own user
- **THEN** the socket MUST be left at mode `0600` owned by that user and no group ownership change MUST be attempted, so that issuance still works without relaxing any authorization check

### Requirement: Subject derived from kernel peer credentials
The server MUST determine the identity of the requester from `SO_PEERCRED` on the accepted connection. The issuance protocol MUST NOT accept a client-supplied subject, username, or uid. The token subject MUST be derived from the peer uid reported by the kernel.

#### Scenario: Subject cannot be spoofed
- **WHEN** a member of the administrative group requests a token
- **THEN** the `sub` claim MUST correspond to the uid reported by `SO_PEERCRED` for that connection, regardless of any value the client sends

#### Scenario: Numeric uid retained alongside username
- **WHEN** a token is issued
- **THEN** the token MUST carry the numeric uid in addition to the human-readable username, so that identity survives username reuse

### Requirement: Signing key generated in memory and never persisted
The server MUST generate the token signing key with a cryptographically secure random source at startup and MUST keep it only in process memory. The key MUST NOT be written to disk, derived from a persisted value, or reloaded across restarts.

#### Scenario: Restart invalidates previously issued tokens
- **WHEN** the service is restarted
- **THEN** a new signing key MUST be generated and every token issued before the restart MUST be rejected

#### Scenario: No key material on disk
- **WHEN** an operator inspects the filesystem after the service has been running
- **THEN** there MUST be no file containing the signing key created by the service

### Requirement: Requested lifetime capped by the server
The client MAY request a token lifetime. The server MUST enforce a maximum lifetime (default 24h) and MUST issue the token with the smaller of the requested and maximum lifetime. Every issued token MUST carry an expiration.

#### Scenario: Excessive lifetime is capped
- **WHEN** a client requests a lifetime longer than the configured maximum
- **THEN** the issued token MUST expire at the configured maximum and the effective expiration MUST be reported to the user

#### Scenario: Token always expires
- **WHEN** any token is issued
- **THEN** it MUST contain an expiration claim in the future

### Requirement: Issuance is audited
The server MUST log one issuance record containing at least the token identifier (`jti`), the peer uid, the resolved subject, the peer pid, and the expiration. The record MUST be written to the service log, not to a file the service manages as state.

#### Scenario: Issuance produces a correlatable record
- **WHEN** a token is issued
- **THEN** the service log MUST contain a record with the `jti` of that token, allowing later correlation with usage records
