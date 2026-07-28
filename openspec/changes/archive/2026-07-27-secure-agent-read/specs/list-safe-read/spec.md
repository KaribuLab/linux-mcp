## ADDED Requirements

### Requirement: List applies the shared path policy
The `list` tool MUST apply the same in-process path denylist as other read tools before calling `ReadDir`. Denied paths MUST NOT return directory listings. Behavior MUST be documented in `docs/tools/list.md`.

#### Scenario: List denied path
- **WHEN** a caller invokes `list` on a denylisted path
- **THEN** the tool MUST reject the request without listing entries

### Requirement: List caps directory entries
The `list` tool MUST bound returned entries to at most 1000. When the directory has more entries than the cap, the tool MUST indicate truncation and MUST NOT attempt to render an unbounded listing.

#### Scenario: Huge directory truncated
- **WHEN** `list` is invoked on a directory with more than 1000 entries
- **THEN** the response MUST include at most 1000 entry rows and MUST signal truncation

### Requirement: List response starts with a metadata header
The `list` tool MUST return a single text payload whose first line is a compact metadata header in the form `[list …]` (including at least `path`, entry counts, and `truncated`), followed by the markdown table body. When the request is blocked by path policy, the tool MUST return a short `[blocked class=… path=…]` line without listing entries.

#### Scenario: Successful list includes metadata line
- **WHEN** `list` successfully lists a directory
- **THEN** the first line of the text result MUST be a `[list …]` metadata line and subsequent content MUST be the markdown table

#### Scenario: Truncated list sets truncated in metadata
- **WHEN** `list` hits the entry cap
- **THEN** the `[list …]` metadata line MUST set `truncated=true` (and SHOULD include enough fields for the agent to know more entries exist)

### Requirement: List MCP tool description explains the full markdown response pattern
The MCP `Tool.Description` for `list` MUST describe the full agent-facing response contract: `[list …]` metadata line, that the body is a **markdown table** after that line (including that `list=false` vs `list=true` change columns), entry cap / truncation when applicable, and `[blocked class=… path=…]` without rows. A one-line description such as only "List the files in a directory in markdown format" is NOT sufficient unless it also covers meta and blocked forms.

#### Scenario: Agent-facing description covers list response contract
- **WHEN** a client lists tools from the MCP server
- **THEN** the `list` tool description MUST mention the `[list …]` metadata line, the markdown table body that follows, and the `[blocked …]` rejection form

### Requirement: List resolves symlink targets from the listed directory
When resolving symlink targets for detailed listings, the tool MUST join the listed directory path with the entry name before `Readlink`. It MUST NOT call `Readlink` with only the basename relative to the process CWD.

#### Scenario: Symlink target under listed path
- **WHEN** `list` is invoked with `list=true` on a directory containing a symlink and the process CWD is not that directory
- **THEN** the tool MUST resolve the symlink using `filepath.Join(dir, name)` (or equivalent) so the target resolution is relative to the listed directory

### Requirement: List resolves group names from GID
When detailed listing includes group ownership, the tool MUST look up the group using the GID (group lookup API), not the UID lookup API with a GID value.

#### Scenario: Group column uses group database
- **WHEN** `list` is invoked with `list=true` on a directory the process can stat
- **THEN** the Group column MUST come from a group-id lookup for the entry GID

### Requirement: List documentation matches behavior
`docs/tools/list.md` MUST describe path policy, entry caps, the `[list …]` metadata header, blocked responses, truncation signaling, symlink resolution, and group lookup consistent with the implementation and the MCP tool description. `docs/README.md` MUST remain consistent if the tool summary changes.

#### Scenario: Docs updated for safe list
- **WHEN** this capability is implemented
- **THEN** `docs/tools/list.md` MUST document path denial, entry caps, `[list …]` / `[blocked …]` formats, and the corrected symlink/group behavior
