## ADDED Requirements

### Requirement: Origin policy is fail-closed and explicitly configured
The HTTP layer MUST receive its allowed origins as explicit configuration rather than as an optional variadic argument, so that omitting the configuration is a compile-time error instead of a silent disabling of the check. A request carrying an `Origin` header that is not in the allowlist MUST be rejected with HTTP 403 and MUST NOT receive an `Access-Control-Allow-Origin` header.

#### Scenario: Unknown origin rejected
- **WHEN** a browser request arrives with an `Origin` that is not in the allowlist
- **THEN** the server MUST respond 403 and MUST NOT set `Access-Control-Allow-Origin`

#### Scenario: Allowed origin echoed
- **WHEN** a browser request arrives with an `Origin` present in the allowlist
- **THEN** the server MUST set `Access-Control-Allow-Origin` to that origin and MUST set `Vary: Origin`

#### Scenario: No global wildcard available
- **WHEN** an operator inspects the origin configuration options
- **THEN** there MUST be no supported value that allows all origins

### Requirement: Port wildcard allowed for loopback origins
The origin allowlist MUST accept entries whose port is the wildcard `*` when the host is a loopback host, for example `http://localhost:*` and `http://127.0.0.1:*`. Such an entry MUST match that scheme and host on any port. A wildcard port MUST NOT be accepted for a non-loopback host, and the host component MUST NOT be wildcarded.

#### Scenario: Any local port matches
- **WHEN** `serve` is configured with `http://127.0.0.1:*` and a request arrives with `Origin: http://127.0.0.1:41235`
- **THEN** the request MUST be allowed

#### Scenario: External host still rejected under a port wildcard
- **WHEN** `serve` is configured with `http://127.0.0.1:*` and a request arrives with an `Origin` on an external host
- **THEN** the request MUST be rejected with 403

#### Scenario: Wildcard host refused at startup
- **WHEN** `serve` is started with an origin entry that wildcards the host rather than the port
- **THEN** startup MUST fail with a configuration error

### Requirement: Rejected origins are logged
When a request is rejected because its `Origin` is not in the allowlist, the server MUST log the received origin value, so an operator integrating a new client can add the exact origin instead of guessing or widening the policy.

#### Scenario: Operator learns the required origin from the log
- **WHEN** a client is rejected because of its origin
- **THEN** the service log MUST contain the rejected origin value

### Requirement: Origins configurable as a comma-separated list
The `serve` command MUST accept allowed origins through a flag that takes a comma-separated list and that may also be repeated. Values MUST be full origins including scheme, host, and port. The default MUST be empty, so no browser origin is allowed unless an operator configures one.

#### Scenario: Comma-separated list accepted
- **WHEN** `serve` is started with two origins separated by a comma
- **THEN** both origins MUST be allowed

#### Scenario: Default allows no browser origin
- **WHEN** `serve` is started without the origin flag
- **THEN** every request carrying an `Origin` header MUST be rejected, while requests without `Origin` MUST still be served

#### Scenario: Inspector works without configuring origins
- **WHEN** the MCP Inspector is used against a `serve` instance started without the origin flag
- **THEN** the connection MUST succeed, because the Inspector reaches the server through its own proxy process, which sends no `Origin` header

### Requirement: Requests without Origin are not subject to the policy
A request that carries no `Origin` header MUST proceed to authentication without an origin check, because such clients are not browsers and are not governed by the browser same-origin policy. Access control for those clients is provided by bearer authentication.

#### Scenario: Non-browser client served
- **WHEN** a non-browser MCP client sends a request with a valid bearer token and no `Origin` header
- **THEN** the request MUST be handled normally

### Requirement: Host header validated against DNS rebinding
The server MUST reject requests whose `Host` header does not match the configured loopback host and port, with HTTP 403. This check MUST be independent of the origin allowlist, because a DNS rebinding request is treated as same-origin by the browser and therefore bypasses CORS entirely.

#### Scenario: Rebound host rejected
- **WHEN** a request arrives with a `Host` header naming an external domain
- **THEN** the server MUST respond 403 regardless of the `Origin` header

#### Scenario: Loopback host accepted
- **WHEN** a request arrives with a `Host` header of `127.0.0.1:5000` or `localhost:5000`
- **THEN** the host check MUST allow the request to proceed

### Requirement: Preflight answered before authentication
CORS handling MUST wrap bearer authentication so that an `OPTIONS` preflight is answered without a token. The advertised allowed request headers MUST include `Authorization`.

#### Scenario: Preflight succeeds without a token
- **WHEN** a browser sends an `OPTIONS` preflight with no `Authorization` header from an allowed origin
- **THEN** the server MUST respond with a success status and MUST NOT respond 401

#### Scenario: Authorization header permitted by preflight
- **WHEN** a browser reads the preflight response
- **THEN** `Access-Control-Allow-Headers` MUST include `Authorization`
