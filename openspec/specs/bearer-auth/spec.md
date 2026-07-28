## Purpose

Bearer token authentication and audit requirements for the MCP HTTP endpoint.

## Requirements

### Requirement: MCP endpoint requires a bearer token
Every MCP request MUST carry an `Authorization: Bearer <token>` header. Requests without a bearer token, or with a token that fails verification, MUST be rejected with HTTP 401 and MUST NOT reach any tool. The 401 response MUST include a `WWW-Authenticate: Bearer` header.

#### Scenario: Unauthenticated request rejected
- **WHEN** an MCP request arrives without an `Authorization` header
- **THEN** the server MUST respond 401 with `WWW-Authenticate: Bearer` and MUST NOT invoke `cat` or `list`

#### Scenario: Authenticated request proceeds
- **WHEN** an MCP request carries a token issued by this server and still valid
- **THEN** the request MUST be handled normally by the MCP server

### Requirement: Token claims fully validated
The token verifier MUST validate the signature against the in-memory signing key and MUST reject the token unless the issuer matches this service, the audience matches this server's resource URL, the not-before time has passed, and the expiration is in the future.

#### Scenario: Expired token rejected
- **WHEN** a request presents a token whose expiration has passed
- **THEN** the server MUST respond 401 and MUST NOT invoke any tool

#### Scenario: Wrong audience rejected
- **WHEN** a request presents an otherwise valid token whose audience is not this server's resource URL
- **THEN** the server MUST respond 401

#### Scenario: Tampered token rejected
- **WHEN** a request presents a token whose payload was modified after signing
- **THEN** signature verification MUST fail and the server MUST respond 401

### Requirement: Required scope enforced
The server MUST require the scope `mcp:read` on every MCP request. A verified token lacking the required scope MUST be rejected with HTTP 403. The required scope MUST be advertised in the `WWW-Authenticate` challenge.

#### Scenario: Missing scope rejected
- **WHEN** a request presents a valid token without the `mcp:read` scope
- **THEN** the server MUST respond 403 and MUST NOT invoke any tool

### Requirement: Authenticated identity bound to the MCP session
The verifier MUST expose the token subject as the authenticated user identifier so that the Streamable HTTP transport can bind an MCP session to a single user.

#### Scenario: Session bound to issuing user
- **WHEN** a token is verified successfully
- **THEN** the resulting token information MUST carry the subject as the user identifier

### Requirement: Tool usage is audited with the token identifier
The server MUST log one record per authenticated tool invocation containing at least the token identifier (`jti`), the subject, and the tool invoked, so that usage can be correlated with the issuance record.

#### Scenario: Usage correlates with issuance
- **WHEN** an authenticated client invokes a tool
- **THEN** the service log MUST contain a record carrying the same `jti` present in the issuance record for that token

### Requirement: No OAuth discovery metadata advertised
The server MUST NOT publish OAuth protected resource metadata and MUST NOT include a `resource_metadata` parameter in the `WWW-Authenticate` challenge, because no authorization server exists. Tokens are pre-issued out of band.

#### Scenario: Challenge omits resource metadata
- **WHEN** the server responds 401 to an unauthenticated request
- **THEN** the `WWW-Authenticate` header MUST NOT reference a resource metadata URL
