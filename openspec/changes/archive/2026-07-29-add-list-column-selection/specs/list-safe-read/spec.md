## ADDED Requirements

### Requirement: List supports per-column visibility flags in detailed mode
When invoked with `list=true`, the `list` tool MUST accept eight optional boolean arguments, one per detailed column except `Name`: `showSize`, `showMode`, `showOwner`, `showGroup`, `showModTime`, `showIsDir`, `showIsSymlink`, `showSymlinkPath`. Each flag MUST default to visible (`true`) when omitted. When a flag is explicitly set to `false`, its corresponding column MUST be excluded from the response table. Column order MUST always be the fixed default order (`Name, Size, Mode, Owner, Group, ModTime, IsDir, IsSymlink, SymlinkPath`), filtered to only the visible columns; flags MUST NOT reorder columns. These flags MUST be ignored when `list=false`.

#### Scenario: Default detailed listing keeps all columns
- **WHEN** `list` is invoked with `list=true` and no `show*` arguments
- **THEN** the response table MUST contain all 9 columns in the existing default order

#### Scenario: Hiding a subset of columns
- **WHEN** `list` is invoked with `list=true`, `showSize=false`, and `showModTime=false`
- **THEN** the response table MUST contain `Name, Mode, Owner, Group, IsDir, IsSymlink, SymlinkPath` in that fixed order, without `Size` or `ModTime`

#### Scenario: Explicit true is equivalent to default
- **WHEN** `list` is invoked with `list=true` and `showSize=true`
- **THEN** the `Size` column MUST be present, identical to the case where `showSize` is omitted

#### Scenario: Visibility flags ignored in simple mode
- **WHEN** `list` is invoked with `list=false` and any `show*` argument set to `false`
- **THEN** the response table MUST remain `|File|` with one row per name, unaffected by the `show*` arguments

### Requirement: List always includes the Name column in detailed mode
When `list=true`, the `list` tool MUST always include the `Name` column as the first column of the response table. There MUST NOT be an argument that can hide the `Name` column.

#### Scenario: Name column present regardless of other flags
- **WHEN** `list` is invoked with `list=true` and every other `show*` flag set to `false`
- **THEN** the response table MUST still contain the `Name` column as its only column

## MODIFIED Requirements

### Requirement: List response starts with a metadata header
The `list` tool MUST return a single text payload whose first line is a compact metadata header in the form `[list …]` (including at least `path`, entry counts, and `truncated`), followed by the markdown table body. When `list=true`, the metadata header MUST also include a `columns=<c1,c2,...>` field listing the effective columns returned, in table order (the full default set when no `show*` flags were set to `false`, or the filtered subset otherwise). When `list=false`, the metadata header MUST NOT include the `columns` field. When the request is blocked by path policy, the tool MUST return a short `[blocked class=… path=…]` line without listing entries.

#### Scenario: Successful list includes metadata line
- **WHEN** `list` successfully lists a directory
- **THEN** the first line of the text result MUST be a `[list …]` metadata line and subsequent content MUST be the markdown table

#### Scenario: Truncated list sets truncated in metadata
- **WHEN** `list` hits the entry cap
- **THEN** the `[list …]` metadata line MUST set `truncated=true` (and SHOULD include enough fields for the agent to know more entries exist)

#### Scenario: Detailed list metadata reports effective columns
- **WHEN** `list` is invoked with `list=true` (with or without `show*` arguments)
- **THEN** the `[list …]` metadata line MUST include a `columns=` field listing the exact columns present in the table, in table order

#### Scenario: Simple list metadata omits columns field
- **WHEN** `list` is invoked with `list=false`
- **THEN** the `[list …]` metadata line MUST NOT include a `columns=` field

### Requirement: List documentation matches behavior
`docs/tools/list.md` MUST describe path policy, entry caps, the `[list …]` metadata header (including the `columns=` field when `list=true`), blocked responses, truncation signaling, symlink resolution, group lookup, and the eight `show*` visibility arguments (their default of `true`, that `Name` cannot be hidden, and that column order is always fixed), consistent with the implementation and the MCP tool description. `docs/README.md` MUST remain consistent if the tool summary changes.

#### Scenario: Docs updated for safe list
- **WHEN** this capability is implemented
- **THEN** `docs/tools/list.md` MUST document path denial, entry caps, `[list …]` / `[blocked …]` formats, the corrected symlink/group behavior, and the `show*` arguments (default `true`, fixed order, `Name` always present, and the `columns=` metadata field)
