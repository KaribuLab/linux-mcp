## Purpose

Compose tool `find_grep`: find-style discovery predicates plus grep-style content search on matching files (find | xargs grep semantics), read-only and bounded.

## Requirements

### Requirement: find_grep searches file contents of find matches
The `find_grep` tool MUST locate filesystem entries using the same read-only metadata predicates as `find` (`path`, `name`, `iname`, `type`, `maxDepth`, `minDepth`), then MUST search the given `pattern` in the **contents** of matching readable files (semantics of `find | xargs grep`). The tool MUST NOT return a find-style markdown table of matches; it MUST return grep-style match rows. Directory entries that match find predicates MUST NOT be content-scanned as files; only entries accepted for content search via the same readability gates as `grep`/`grepScan` MUST be scanned.

#### Scenario: Content matches under find name filter
- **WHEN** a caller invokes `find_grep` with `name` (or `iname`) selecting a subset of files and a `pattern` present in some of those files' contents
- **THEN** the response MUST include `path:line:content` rows only for content matches inside that subset

#### Scenario: Files outside find predicates are not searched
- **WHEN** files under `path` exist that do not satisfy the find predicates but would match `pattern` if grepped
- **THEN** those files MUST NOT contribute match rows

#### Scenario: Directory match is not content-scanned
- **WHEN** a directory entry satisfies the find predicates
- **THEN** the tool MUST NOT treat that directory path as a file to scan for content lines

### Requirement: find_grep reuses grep pattern and safety behavior
The `find_grep` tool MUST support the same pattern modes as `grep` (literal default, RE2 when `extended=true`, optional `ignoreCase`). During content search it MUST apply the same binary skip / private-key redaction rules and per-line byte cap as `grep`.

#### Scenario: Binary file among find matches is skipped in multi-file search
- **WHEN** a find match is a binary-classified file in a multi-file `find_grep` run
- **THEN** the tool MUST skip that file's content without aborting the overall search (same recursive binary policy as `grep`)

#### Scenario: Private-key match rows are redacted
- **WHEN** a find match is private-key-classified and the pattern matches lines in that file
- **THEN** those rows MUST use the redaction placeholder and count toward `redacted` in metadata

### Requirement: find_grep uses the shared safe walk
The `find_grep` discovery phase MUST use the shared walk helper: denylist check per visited node, no symlink following while descending, and the hard node-visited budget. A denied root path MUST yield `[blocked class=... path=...]` with no match rows.

#### Scenario: Denied root path
- **WHEN** `find_grep` is invoked with a denylisted root `path`
- **THEN** the tool MUST return `[blocked class=... path=...]` and MUST NOT search contents

#### Scenario: Node budget truncation
- **WHEN** the find walk hits the shared node budget before finishing the tree
- **THEN** the response metadata MUST indicate `truncated=true`

### Requirement: find_grep response format mirrors grep with distinct metadata tag
On success, `find_grep` MUST return metadata tagged `[find_grep ...]` followed by raw `path:line:content` rows (not markdown). Match rows MUST be capped by the same match-row cap as `grep`. The tool MUST NOT expose `find` column-selection flags (`showPath`/`showType`/`showSize`/`showModTime`).

#### Scenario: Success shape
- **WHEN** `find_grep` finds content matches
- **THEN** the first line MUST start with `[find_grep ` and subsequent lines MUST be `path:line:content` rows

#### Scenario: Match cap truncation
- **WHEN** content matches exceed the grep match-row cap
- **THEN** the tool MUST return at most that many rows and MUST set `truncated=true` in metadata

### Requirement: find_grep is documented for agents
The public behavior of `find_grep` MUST be documented in `docs/tools/find_grep.md` and indexed from `docs/README.md`, aligned with the MCP tool description and these requirements.

#### Scenario: Docs exist for the tool
- **WHEN** the change is complete
- **THEN** `docs/tools/find_grep.md` MUST describe parameters, semantics, caps, and response shape, and `docs/README.md` MUST list the tool
