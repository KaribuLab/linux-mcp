## Purpose

Server-side row-filter compose tool `ps_grep` over the `ps` inventory.

## Requirements

### Requirement: Ps grep filters ps rows server-side
The `ps_grep` tool MUST apply the same process selection and `show*` column rules as `ps`, then filter **data rows** by `pattern` without returning the unfiltered intermediate list to the caller. It MUST NOT spawn or execute host binaries. Pattern modes MUST follow the same spirit as `list_grep` (`extended`, `ignoreCase`, and documented auto-glob vs literal rules against identity/`Comm` or full row as specified in `docs/tools/ps_grep.md`).

#### Scenario: Pattern reduces returned rows
- **WHEN** `ps_grep` is invoked with a `pattern` that matches a subset of processes
- **THEN** the response table MUST contain only matching data rows and MUST use a `[ps_grep …]` metadata line

### Requirement: Ps grep preserves caps and truncation semantics
`ps_grep` MUST cap output at 1000 rows. `truncated` MUST be true if the underlying `ps` selection was truncated or if matched rows exceed the output cap.

#### Scenario: Truncation signaled when base inventory truncated
- **WHEN** the underlying process inventory hits the cap before filtering
- **THEN** the `[ps_grep …]` metadata MUST set `truncated=true` even if few rows match the pattern

### Requirement: Ps grep accepts the same show flags as ps
`ps_grep` MUST accept the same `show*` flags and `includeKernel` as `ps`, with the same defaults and identity columns `Pid` and `Comm` always present.

#### Scenario: Show flags apply after filter
- **WHEN** `ps_grep` is invoked with `showMem=false` and a matching `pattern`
- **THEN** matching rows MUST be returned without the `Mem` column

### Requirement: Ps grep docs include agent example prompt
`docs/tools/ps_grep.md` MUST document behavior consistent with the implementation and MUST include `## Prompt de ejemplo (agente)` with at least one Spanish-neutral prompt exercising `ps_grep` via linux-mcp. The MCP tool description MUST cover the compose contract (filter server-side, `[ps_grep …]`, caps).

#### Scenario: Docs include example agent prompt
- **WHEN** an operator opens `docs/tools/ps_grep.md`
- **THEN** the document MUST contain `## Prompt de ejemplo (agente)` and at least one usable example prompt referencing `ps_grep`
