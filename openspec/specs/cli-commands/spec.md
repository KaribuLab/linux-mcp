## Purpose

CLI structure, behaviour, and documentation for `linux-mcp serve` and `linux-mcp auth`.

## Requirements

### Requirement: Binary exposes serve and auth subcommands
The binary MUST provide a `serve` subcommand that runs the MCP server and an `auth` subcommand that obtains a token. Command wiring MUST live under `internal/command` with one file per command, and `cmd/linux-mcp/main.go` MUST only delegate to the command root and propagate the exit code.

#### Scenario: Bare invocation shows help
- **WHEN** the binary is invoked with no arguments
- **THEN** it MUST print help and MUST NOT start a network listener

#### Scenario: Serve starts the MCP server
- **WHEN** `linux-mcp serve` is invoked
- **THEN** the MCP endpoint and the issuance socket MUST both be started

### Requirement: Socket path shared by both commands
The issuance socket path MUST be configurable through a single persistent flag available to both `serve` and `auth`, so both processes resolve the same rendezvous point by default.

#### Scenario: Custom socket path honored by both commands
- **WHEN** `serve` and `auth` are given the same non-default socket path
- **THEN** `auth` MUST obtain a token from that `serve` instance

### Requirement: Auth command has no subject flag
The `auth` command MUST NOT accept a flag that sets the token subject, username, or uid. The requester identity is determined by the server from kernel peer credentials.

#### Scenario: No way to request a token for another user
- **WHEN** an operator inspects `linux-mcp auth --help`
- **THEN** there MUST be no flag that selects the subject of the issued token

### Requirement: Auth output separates token from diagnostics
The `auth` command MUST write the token, and nothing else, to standard output. Warnings, the effective expiration, and errors MUST go to standard error, so the token can be captured by command substitution.

#### Scenario: Token captured cleanly in a script
- **WHEN** an operator runs `TOKEN=$(linux-mcp auth --ttl 8h)`
- **THEN** the variable MUST contain only the token value

#### Scenario: Capped lifetime reported without polluting stdout
- **WHEN** the requested lifetime is reduced by the server maximum
- **THEN** the notice MUST be written to standard error

### Requirement: Binary reports its version
The root command MUST support `--version`, printing a version value that the build can inject at link time, with a usable fallback for local builds that do not inject it.

#### Scenario: Version flag available
- **WHEN** the binary is invoked with `--version`
- **THEN** it MUST print its version and exit without starting any listener

### Requirement: Every command is documented
Each CLI command MUST have a document under `docs/commands/<name>.md` describing its flags with their defaults, its output, and its error conditions. The documentation index MUST link to those documents. A command MUST NOT be considered complete while its documentation disagrees with the actual behaviour of the code.

#### Scenario: Commands documented and indexed
- **WHEN** a reader opens the documentation index
- **THEN** there MUST be links to `docs/commands/serve.md` and `docs/commands/auth.md`

#### Scenario: Flags and defaults described
- **WHEN** an operator reads a command document
- **THEN** it MUST list every flag of that command with its default value and describe the command's output and error conditions

### Requirement: Serve binds an explicit loopback address
The `serve` command MUST default its MCP listen address to an explicit loopback IP and port rather than a hostname, so the bound address does not depend on name resolution.

#### Scenario: Default listen address is explicit
- **WHEN** `serve` starts without an address flag
- **THEN** it MUST listen on `127.0.0.1:5000`
