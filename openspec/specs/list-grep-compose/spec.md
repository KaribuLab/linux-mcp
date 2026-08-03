## Purpose

Server-side row-filter compose tool `list_grep` over the `list` directory inventory (listing lines only; never file contents).

## Requirements

### Requirement: list_grep filters list output rows by pattern
The `list_grep` tool MUST list a directory using the same path policy, `all`/`list`/`show*` flags, and entry cap as `list`, then MUST filter data rows of the markdown table by a pattern. The tool MUST NOT open file contents. Matching MUST apply to each full data row line (as `ls | grep` matches the listing line), not only the `Name` column. Markdown header rows (`|Col|` and separator `|---|`) and the tool metadata line MUST NOT be treated as filterable data rows that can appear as matches by themselves.

#### Scenario: Pattern keeps matching names in simple mode
- **WHEN** a caller invokes `list_grep` with `list=false` (or omitted) and a `pattern` that matches a subset of entry names under `path`
- **THEN** the response body MUST be a markdown table in the same simple shape as `list`, containing only data rows whose line matches the pattern

#### Scenario: Pattern applies to full detailed row
- **WHEN** a caller invokes `list_grep` with `list=true` and a `pattern` that appears only in a non-Name column of a detailed row
- **THEN** that row MUST be included in the filtered result

#### Scenario: No file content is read
- **WHEN** `list_grep` runs successfully against a directory of regular files
- **THEN** the tool MUST NOT read the contents of those files as part of matching

### Requirement: list_grep pattern modes support glob names and grep-like text
When `extended=false` (or omitted), if `pattern` contains glob metacharacters `*`, `?`, or `[`, the `list_grep` tool MUST match it with Go `filepath.Match` against the entry Name/File column only (not file contents). Otherwise it MUST treat `pattern` as literal substring text against the full data row. When `extended=true`, the tool MUST compile `pattern` as RE2 against the full data row and MUST NOT auto-detect globs. The tool MUST support `ignoreCase` in all modes.

#### Scenario: Glob star matches entry names
- **WHEN** `list_grep` is invoked with `extended=false` and `pattern` `*.txt`
- **THEN** the tool MUST include rows whose Name/File matches that glob (e.g. `foo.txt`) and MUST NOT read file contents to decide the match

#### Scenario: Literal mode without glob metacharacters
- **WHEN** `list_grep` is invoked with `extended=false` and a `pattern` without `*`, `?`, or `[` that contains `.`
- **THEN** the tool MUST match that pattern as literal substring text in the row line

#### Scenario: Extended mode uses RE2
- **WHEN** `list_grep` is invoked with `extended=true` and a valid RE2 `pattern`
- **THEN** the tool MUST filter rows using that regular expression against the full data row

### Requirement: list_grep response format mirrors list with distinct metadata tag
On success, `list_grep` MUST return a single text payload whose first line is metadata tagged `[list_grep ...]` (not `[list ...]`) followed by a markdown table using the same column rules as `list` for the effective `list`/`show*` flags. On a denied root path, the tool MUST return `[blocked class=... path=...]` with no table, same policy classes as `list`.

#### Scenario: Success header uses list_grep tag
- **WHEN** `list_grep` succeeds
- **THEN** the first line MUST start with `[list_grep ` and the body MUST be a markdown table consistent with `list` for the same listing flags

#### Scenario: Denied path blocks like list
- **WHEN** `list_grep` is invoked with a denylisted `path`
- **THEN** the tool MUST return a single `[blocked class=... path=...]` line and MUST NOT return a table

### Requirement: list_grep bounds output and reports truncation honestly
The `list_grep` tool MUST NOT return more data rows than the shared list entry cap. If the underlying directory listing was truncated before filtering, or if filtered matches exceed the output cap, the metadata MUST indicate `truncated=true`.

#### Scenario: Truncated base listing is signaled
- **WHEN** the directory has more visible entries than the list entry cap and `list_grep` filters the capped listing
- **THEN** the response metadata MUST set `truncated=true` so the caller knows unseen entries were not considered

### Requirement: list_grep is documented for agents
The public behavior of `list_grep` MUST be documented in `docs/tools/list_grep.md` and indexed from `docs/README.md`, aligned with the MCP tool description and these requirements.

#### Scenario: Docs exist for the tool
- **WHEN** the change is complete
- **THEN** `docs/tools/list_grep.md` MUST describe parameters, semantics, caps, and response shape, and `docs/README.md` MUST list the tool
