## ADDED Requirements

### Requirement: Cat streams with dual output caps
The `cat` tool MUST NOT load an entire file via unbounded `ReadFile` and MUST NOT keep a full-file cache between calls. It MUST stream content subject to both a maximum of 100 lines and a maximum of 64 KiB (whichever limit is hit first MUST stop the read). Behavior MUST be documented in `docs/tools/cat.md`.

#### Scenario: Large text file truncated by line cap
- **WHEN** a caller invokes `cat` on a text file longer than 100 lines
- **THEN** the response body MUST contain at most 100 lines and MUST indicate truncation in the metadata line

#### Scenario: Wide content truncated by byte cap
- **WHEN** a caller invokes `cat` on content that exceeds 64 KiB before the line cap
- **THEN** the read MUST stop at the byte cap and MUST indicate truncation

### Requirement: Cat resumes with byte offset via Seek
The optional `offset` argument and metadata `next` value MUST be byte cursors. When `offset > 0` and the file is seekable, the implementation MUST `Seek` to that byte before reading the next bounded page. It MUST NOT implement resume by repeatedly skipping N lines from the start of the file on each call (O(n²) I/O). Line counts in metadata are informational for the returned page only.

#### Scenario: Second page uses byte seek
- **WHEN** a prior `cat` response set `truncated=true` and `next` to a byte position, and the caller invokes `cat` again with that value as `offset` on a seekable file
- **THEN** the tool MUST resume from that byte position without re-scanning the whole file as line-skip from offset zero

### Requirement: Cat response is a single low-token text payload
The `cat` tool MUST return a single text payload: one compact metadata line in the form `[cat …]` followed by the raw text body. It MUST NOT emit a markdown table, JSON-per-line, or a structured array of lines as the primary content format.

#### Scenario: Successful read shape
- **WHEN** `cat` successfully reads a non-blocked file
- **THEN** the first line of the text result MUST be a `[cat …]` metadata line including at least `path` and whether the result is truncated, and subsequent lines MUST be the raw body excerpt

#### Scenario: Blocked read shape
- **WHEN** `cat` blocks due to policy or private-key sniff
- **THEN** the client-visible result MUST be a short blocked indicator (for example `[blocked class=… path=…]`) without the sensitive body

### Requirement: Cat rejects binary content in v1
If the bounded prefix contains a NUL byte, `cat` MUST reject the read as binary and MUST NOT return a full binary dump.

#### Scenario: Binary file rejected
- **WHEN** `cat` reads a file whose prefix contains a NUL byte
- **THEN** the tool MUST return a blocked/binary rejection without dumping the binary contents

### Requirement: Cat MCP tool description explains the full response pattern
The MCP `Tool.Description` registered for `cat` MUST describe the full agent-facing response contract: success shape (`[cat …]` metadata line then raw body), caps (lines ∩ bytes), truncation/`next` as byte resume via `offset`, and blocked/error lines (`[blocked class=… path=…]` including the main classes). A one-line description such as only "Read the contents of a file" is NOT sufficient.

#### Scenario: Agent-facing description covers response contract
- **WHEN** a client lists tools from the MCP server
- **THEN** the `cat` tool description MUST mention the `[cat …]` metadata line on success, that the body is raw text after that line, byte `next`/`offset` resume when truncated, and the `[blocked …]` form for policy/content rejections

### Requirement: Cat documentation matches behavior
`docs/tools/cat.md` MUST describe parameters (including byte `offset`), caps, metadata line format, blocked classes, Seek resume, and private-key sniff rules consistent with the implementation and with the MCP tool description.

#### Scenario: Docs updated with caps and sniff
- **WHEN** this capability is implemented
- **THEN** `docs/tools/cat.md` MUST document the output caps, `[cat …]` metadata, byte resume, and content/path blocking behavior
