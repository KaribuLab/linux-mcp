## ADDED Requirements

### Requirement: Ps lists processes from proc with a hard row cap
The `ps` tool MUST enumerate processes via `/proc` (or equivalent kernel interface) in-process. It MUST NOT spawn or execute host binaries (including `ps`, shells, or wrappers). It MUST return at most 1000 data rows. When more processes match the selection than the cap, the response MUST set `truncated=true` in the metadata line and MUST NOT return unbounded output.

#### Scenario: Process list truncated at cap
- **WHEN** more than 1000 processes would be returned under the active filters
- **THEN** the tool MUST return at most 1000 rows and the metadata line MUST include `truncated=true`

### Requirement: Ps excludes kernel threads by default
By default, `ps` MUST omit kernel threads (e.g. names typically shown in brackets). When `includeKernel` is true, those processes MUST be eligible for inclusion.

#### Scenario: Default omits kernel threads
- **WHEN** `ps` is invoked without `includeKernel` (or with `includeKernel=false`)
- **THEN** the returned rows MUST NOT include kernel-thread processes as defined in the tool documentation

#### Scenario: includeKernel adds kernel threads
- **WHEN** `ps` is invoked with `includeKernel=true`
- **THEN** kernel-thread processes MUST be eligible to appear in the result (subject to the row cap)

### Requirement: Ps supports show flags for optional columns
The `ps` tool MUST always include identity columns `Pid` and `Comm` as the first columns in fixed order. It MUST accept optional boolean flags defaulting to visible (`true` when omitted): `showPpid`, `showUser`, `showStat`, `showCpu`, `showMem`, `showCmdline`, `showExe`. Setting a flag to `false` MUST omit that column. Flags MUST NOT reorder columns. There MUST NOT be a flag that hides `Pid` or `Comm`.

#### Scenario: Default shows all optional columns
- **WHEN** `ps` is invoked with no `show*` arguments
- **THEN** the table MUST include `Pid`, `Comm`, and all optional columns in the documented fixed order

#### Scenario: Hiding optional columns
- **WHEN** `ps` is invoked with `showCmdline=false` and `showExe=false`
- **THEN** the table MUST omit `Cmdline` and `Exe` but MUST still include `Pid` and `Comm`

### Requirement: Ps response metadata and markdown table
On success, `ps` MUST return a single text payload whose first line is `[ps …]` including at least process counts and `truncated`, plus `columns=<c1,c2,...>` listing effective columns in table order, followed by a markdown table body.

#### Scenario: Successful ps includes metadata and columns
- **WHEN** `ps` successfully lists processes
- **THEN** the first line MUST be a `[ps …]` metadata line including `columns=` and subsequent content MUST be the markdown table

### Requirement: Ps MCP description and docs include agent example prompt
The MCP `Tool.Description` for `ps` MUST describe the agent-facing contract (metadata, markdown table, caps, `show*`, `includeKernel`). `docs/tools/ps.md` MUST document parameters, caps, columns, and MUST include a section `## Prompt de ejemplo (agente)` with at least one Spanish-neutral prompt a human can paste to an agent to exercise `ps` via linux-mcp (e.g. starting with `Usa el tool linux-mcp \`ps\` para …`).

#### Scenario: Docs include example agent prompt
- **WHEN** an operator opens `docs/tools/ps.md`
- **THEN** the document MUST contain `## Prompt de ejemplo (agente)` and at least one usable example prompt referencing the `ps` tool
