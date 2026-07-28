## Purpose

Documentation for connecting supported MCP clients to linux-mcp with SSH tunneling and bearer tokens.

## Requirements

### Requirement: One integration document per MCP client
The repository MUST provide a document per supported MCP client under `docs/agents/<client>.md`, covering at least Claude, Codex, and OpenCode. Each document MUST contain a copyable configuration block for that client, the server URL reached through the SSH tunnel, and how the token is supplied as an `Authorization: Bearer` header.

#### Scenario: Operator configures a client from a single page
- **WHEN** an operator opens the document for the client they use
- **THEN** it MUST contain a configuration block they can copy without consulting the documents of other clients

#### Scenario: Expired token path documented
- **WHEN** an operator reads a client document
- **THEN** it MUST state that tokens expire and that a new one is obtained with `linux-mcp auth`

### Requirement: Client transport support verified, not assumed
Each client document MUST reflect the client's actual support for Streamable HTTP with custom headers, verified against that client's current documentation at the time of writing. If a client cannot supply an `Authorization` header natively, the document MUST describe the required bridge instead of presenting a configuration that does not work.

#### Scenario: Unsupported client documented honestly
- **WHEN** a client does not support custom headers over Streamable HTTP
- **THEN** its document MUST describe the bridge or workaround rather than an untested configuration block

### Requirement: Local development workflow documented
The repository MUST provide `docs/runbooks/local-development.md` describing how to run the server with live reload, obtain a token against the local instance, and connect the MCP Inspector. It MUST explain that local development requires no administrative group, because the developer owns the socket.

#### Scenario: Developer reaches a working local setup
- **WHEN** a developer follows the local development runbook
- **THEN** it MUST show starting the server with reload, generating a token in a second terminal, and connecting the Inspector with that token

#### Scenario: Group requirement explained as production-only
- **WHEN** a developer reads the local development runbook
- **THEN** it MUST explain that the administrative group applies only when the service runs as a different user

### Requirement: Documentation index links clients and local workflow
`docs/README.md` and the project `README.md` MUST link to every client integration document and to the local development runbook.

#### Scenario: Indexes expose the new documents
- **WHEN** a reader opens the documentation index or the project README
- **THEN** there MUST be links to each `docs/agents/<client>.md` and to `docs/runbooks/local-development.md`
