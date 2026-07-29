## ADDED Requirements

### Requirement: Find limits predicates to read-only metadata tests
The `find` tool MUST only support predicates that query filesystem metadata (name/pattern match, entry type, depth bounds). It MUST NOT support any predicate or option that executes a subprocess or mutates the filesystem (equivalents of `-exec`, `-execdir`, `-ok`, `-okdir`, `-delete`, `-fprint`/`-fprintf`).

#### Scenario: Only metadata filters accepted
- **WHEN** a caller invokes `find` with `name`/`iname`, `type`, `maxDepth`, and/or `minDepth`
- **THEN** the tool MUST filter matches using only those metadata predicates and MUST NOT expose any argument that runs a command or writes/deletes filesystem entries

### Requirement: Find applies the shared path policy per visited node
The `find` tool MUST reject a denylisted root path before starting the walk, using the same in-process path policy as `cat` and `list`. During the recursive walk, any node that matches the path denylist MUST be skipped without appearing in results, and the walk MUST continue for sibling and remaining nodes.

#### Scenario: Denied root path
- **WHEN** a caller invokes `find` with a root `path` that matches the denylist
- **THEN** the tool MUST return a `[blocked class=... path=...]` response and MUST NOT start the walk

#### Scenario: Denied node inside the walk
- **WHEN** the recursive walk reaches a node that matches the path denylist but the root path is allowed
- **THEN** that node MUST be excluded from results and the walk MUST continue visiting other nodes

### Requirement: Find never follows symlinks during the walk
The `find` tool MUST NOT follow symbolic links while walking the directory tree. A symlink entry MAY be reported as a match (subject to `type` filtering) but its target MUST NOT be descended into.

#### Scenario: Symlink to a directory is not descended
- **WHEN** the walk encounters a symlink pointing to a directory
- **THEN** the tool MUST NOT recurse into the symlink target and MUST continue the walk at the same depth for other entries

### Requirement: Find caps the walk by a node budget and results by a match cap
The `find` tool MUST use the shared walk helper's hard node-visited budget to bound the cost of the search independent of how many matches are found. It MUST also bound the number of returned matches. When either cap is hit, the response metadata MUST indicate truncation.

#### Scenario: No matches on a huge tree still stops
- **WHEN** `find` is invoked with a `name` pattern that matches nothing under a root larger than the node budget
- **THEN** the walk MUST stop once the node budget is reached and the response MUST indicate `truncated=true`

#### Scenario: Many matches capped in output
- **WHEN** a `find` invocation matches more entries than the result cap
- **THEN** the response MUST include at most the capped number of rows and MUST indicate `truncated=true`

### Requirement: Find response starts with a metadata header
The `find` tool MUST return a single text payload whose first line is a compact metadata header in the form `[find path=... matches=returned/total truncated=bool visited=...]`, followed by a markdown table body. When the root path is blocked by policy, the tool MUST return a short `[blocked class=... path=...]` line without listing matches.

#### Scenario: Successful find includes metadata line
- **WHEN** `find` successfully completes a walk
- **THEN** the first line of the text result MUST be a `[find ...]` metadata line and subsequent content MUST be the markdown table of matches

### Requirement: Find lets the caller select returned columns via boolean flags
The `find` tool MUST accept optional boolean arguments `showPath`, `showType`, `showSize`, and `showModTime`, each defaulting to `true`. The result table MUST include exactly the columns whose flag is `true`, in the fixed order `Path`, `Type`, `Size`, `ModTime`, regardless of the order the flags were supplied. If all four flags are explicitly set to `false`, the tool MUST still return the `Path` column alone rather than an empty table.

#### Scenario: Default returns all columns
- **WHEN** a caller invokes `find` without specifying any `show*` flag
- **THEN** the result table MUST include all four columns: `Path`, `Type`, `Size`, `ModTime`

#### Scenario: Caller restricts to a subset of columns
- **WHEN** a caller invokes `find` with `showPath=true` and `showType=true`, leaving `showSize` and `showModTime` as `false`
- **THEN** the result table MUST include only the `Path` and `Type` columns, in that order, and MUST NOT include `Size` or `ModTime`

#### Scenario: All column flags false still returns Path
- **WHEN** a caller invokes `find` with `showPath=false`, `showType=false`, `showSize=false`, and `showModTime=false`
- **THEN** the result table MUST fall back to returning the `Path` column alone

### Requirement: Find MCP tool description explains the full response contract
The MCP `Tool.Description` registered for `find` MUST describe: the supported metadata predicates, the absence of any execute/delete/write predicate, the `[find ...]` metadata line and markdown table body, the node-budget and match-cap truncation behavior, and the `[blocked class=... path=...]` rejection form.

#### Scenario: Agent-facing description covers response contract
- **WHEN** a client lists tools from the MCP server
- **THEN** the `find` tool description MUST mention the `[find ...]` metadata line, the markdown table body, truncation semantics, and the `[blocked ...]` form

### Requirement: Find documentation matches behavior
`docs/tools/find.md` MUST describe the supported predicates, the shared walk policy (no symlink following, node budget, denylist per node), the `[find ...]` / `[blocked ...]` response formats, and truncation semantics, consistent with the implementation and the MCP tool description.

#### Scenario: Docs updated for find
- **WHEN** this capability is implemented
- **THEN** `docs/tools/find.md` MUST document the predicates, walk safety controls, and response formats
