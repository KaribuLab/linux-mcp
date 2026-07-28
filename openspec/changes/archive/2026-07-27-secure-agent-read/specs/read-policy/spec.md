## ADDED Requirements

### Requirement: Path denylist enforced in-process
The system MUST evaluate absolute or resolved paths against an in-process denylist before any tool reads or lists filesystem content. Enforcement MUST NOT depend on systemd or external sandboxing.

#### Scenario: Denied sensitive path
- **WHEN** a tool requests a path that matches the denylist (for example `/etc/shadow` or `/etc/gshadow`)
- **THEN** the tool MUST reject the request without returning file contents or directory entries for that path

#### Scenario: Allowed operational path
- **WHEN** a tool requests a non-denied path used for ops visibility (for example under `/proc`, `/sys`, or `/etc` configs not on the denylist)
- **THEN** the path policy MUST allow the request to proceed to subsequent checks (type, content sniff, output limits)

### Requirement: Dangerous file types rejected
The system MUST refuse to open for content reading device nodes, sockets, and other non-regular special files that are not explicitly treated as readable text sources. Paths known to expose raw process memory (for example `/proc/*/mem`, `kcore`) MUST be denied.

#### Scenario: Block device rejected
- **WHEN** `cat` is asked to read a block device path
- **THEN** the request MUST be rejected without streaming device data

### Requirement: Private key content sniff on first useful line
Before returning file body bytes to the MCP client, the system MUST inspect the first non-empty line of a bounded prefix (after skipping BOM and leading blank lines). If that line matches a private-key header (OpenSSH, PEM `PRIVATE KEY` variants, PGP private key block, or PuTTY private key file header), the system MUST block the read and MUST NOT return the file body.

#### Scenario: PEM private key blocked by content
- **WHEN** the first useful line of a file is `-----BEGIN RSA PRIVATE KEY-----` (or another listed private-key header)
- **THEN** the tool MUST return a blocked result with class indicating private key and MUST NOT include key material in the response body

#### Scenario: Example header not at start does not block
- **WHEN** a text file contains a private-key header only after other non-empty content (for example a README)
- **THEN** the private-key sniff MUST NOT block solely for that mid-file occurrence

#### Scenario: Public key allowed by sniff
- **WHEN** the first useful line indicates a public key (for example `-----BEGIN PUBLIC KEY-----` or an `ssh-ed25519` public line)
- **THEN** the private-key sniff MUST NOT block the read

### Requirement: Shared policy reusable by tools
Path checks, type checks, content classification, and output-limit helpers MUST live in a shared internal package usable by `cat`, `list`, and future read-oriented tools.

#### Scenario: Same path deny for list and cat
- **WHEN** a path is denied by policy
- **THEN** both `cat` and `list` MUST reject that path under the same rules
