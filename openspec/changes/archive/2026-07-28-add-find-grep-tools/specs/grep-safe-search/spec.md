## ADDED Requirements

### Requirement: Grep supports literal and extended-regex modes only
The `grep` tool MUST support exactly two pattern modes: a default literal/basic mode where the pattern is matched as plain text (no metacharacter interpretation), and an extended mode (`extended=true`) where the pattern is compiled as a regular expression using Go's RE2-based `regexp` engine. The tool MUST NOT support a backtracking regex engine (for example PCRE/`-P` semantics) or any option that executes a subprocess.

#### Scenario: Default mode treats pattern as literal text
- **WHEN** a caller invokes `grep` without `extended` (or `extended=false`) and the pattern contains regex metacharacters such as `.` or `*`
- **THEN** the tool MUST match the pattern as literal text, not as a regular expression

#### Scenario: Extended mode compiles as RE2 regex
- **WHEN** a caller invokes `grep` with `extended=true`
- **THEN** the tool MUST compile the pattern with Go's `regexp` package and match accordingly

### Requirement: Grep searches a single file or recursively walks a directory
The `grep` tool MUST accept a `path` argument that may be a regular file or a directory. When `path` is a directory, the tool MUST search matching content recursively using the shared walk helper, subject to the same path policy, symlink, and node-budget rules as `find`.

#### Scenario: Single file search
- **WHEN** `grep` is invoked with `path` pointing to a regular file
- **THEN** the tool MUST search only that file for the pattern

#### Scenario: Directory search walks recursively
- **WHEN** `grep` is invoked with `path` pointing to a directory
- **THEN** the tool MUST recursively search files under that directory using the shared walk helper (denylist per node, no symlink following, node budget)

### Requirement: Grep applies the shared path policy
The `grep` tool MUST reject a denylisted root path before searching, using the same in-process path policy as `cat`, `list`, and `find`. During a recursive search, any node that matches the path denylist MUST be skipped without its contents appearing in results.

#### Scenario: Denied root path
- **WHEN** a caller invokes `grep` with a root `path` that matches the denylist
- **THEN** the tool MUST return a `[blocked class=... path=...]` response and MUST NOT search any content

### Requirement: Grep silently skips binary content during recursive search
When recursively searching, the `grep` tool MUST classify each file's content prefix using the same sniff used by `cat` (binary via NUL byte). If a file is classified as binary, the tool MUST skip that file's content without emitting a blocked line for it and MUST continue the search on remaining files. This skip MUST NOT be reported as a count in the response metadata.

#### Scenario: Binary file skipped during recursive grep
- **WHEN** a recursive `grep` walk reaches a file whose content prefix contains a NUL byte
- **THEN** the tool MUST skip that file's content and continue the search

### Requirement: Grep searches private-key-classified content but redacts matched lines
The `grep` tool MUST NOT skip or block a file classified as private-key by the content sniff (header via the same classifier used by `cat`), whether encountered as a single-file target or during a recursive search. The tool MUST still search that file's content for the pattern, but for every matching line it MUST replace the returned `<content>` with a fixed redaction placeholder (`[private-key content redacted]`) instead of the real line text. The response metadata MUST report the total number of redacted rows via a `redacted=<n>` field.

#### Scenario: Private-key file matched during recursive grep
- **WHEN** a recursive `grep` walk reaches a file whose content sniff matches a private-key header and the pattern matches one or more lines in that file
- **THEN** the tool MUST include a row per match with `<content>` replaced by `[private-key content redacted]`, MUST count those rows in `redacted`, and MUST continue searching remaining files

#### Scenario: Single-file grep target is a private-key file
- **WHEN** `grep` is invoked with `path` pointing directly to a file whose content sniff matches a private-key header
- **THEN** the tool MUST search that file's content for the pattern instead of returning a `[blocked ...]` response, and MUST redact any matching row's content as above

#### Scenario: Redacted count reported even with zero matches
- **WHEN** a private-key-classified file is scanned and the pattern does not match any of its lines
- **THEN** the file contributes zero rows and zero to `redacted`, and the search continues normally

### Requirement: Grep caps output and per-line length
The `grep` tool MUST bound the total number of returned match rows and MUST bound the length of each returned line using the same byte cap as `cat`'s per-read limit. When a recursive search exceeds the shared walk node budget, the response MUST indicate truncation.

#### Scenario: Long matching line is truncated per row
- **WHEN** a matching line exceeds the per-line byte cap
- **THEN** the tool MUST truncate that row's content to the cap and MUST NOT fail the whole request

#### Scenario: Many matches capped in output
- **WHEN** a search produces more matching lines than the result cap
- **THEN** the response MUST include at most the capped number of rows and MUST indicate `truncated=true`

### Requirement: Grep response starts with a metadata header
The `grep` tool MUST return a single text payload whose first line is a compact metadata header in the form `[grep path=... matches=returned/total truncated=bool filesScanned=... redacted=...]`, followed by raw text rows in the form `<path>:<line>:<content>`. When the root path or a single-file binary target is blocked by policy, the tool MUST return a short `[blocked class=... path=...]` line without any match rows.

#### Scenario: Successful grep includes metadata line
- **WHEN** `grep` successfully completes a search
- **THEN** the first line of the text result MUST be a `[grep ...]` metadata line including a `redacted` count and subsequent lines MUST be `<path>:<line>:<content>` rows

### Requirement: Grep MCP tool description explains the full response contract
The MCP `Tool.Description` registered for `grep` MUST describe: the literal vs extended-regex modes and that extended mode uses RE2 (not PCRE), single-file vs recursive-directory behavior, the silent skip of binary content, the search-and-redact treatment of private-key-classified content (including the `redacted` count), the `[grep ...]` metadata line and row format, truncation behavior, and the `[blocked class=... path=...]` rejection form.

#### Scenario: Agent-facing description covers response contract
- **WHEN** a client lists tools from the MCP server
- **THEN** the `grep` tool description MUST mention both pattern modes, the private-key redaction behavior and `redacted` count, the metadata line and row format, truncation semantics, and the `[blocked ...]` form

### Requirement: Grep documentation matches behavior
`docs/tools/grep.md` MUST describe both pattern modes, single-file and recursive-directory behavior, the shared walk policy, the silent skip of binary content, the search-and-redact treatment of private-key content with its `redacted` count, output caps, and the `[grep ...]` / `[blocked ...]` response formats, consistent with the implementation and the MCP tool description.

#### Scenario: Docs updated for grep
- **WHEN** this capability is implemented
- **THEN** `docs/tools/grep.md` MUST document both pattern modes, recursive behavior, and response formats
