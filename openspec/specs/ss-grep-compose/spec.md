## Purpose

Server-side row-filter compose tool `ss_grep` over the `ss` inventory.

## Requirements

### Requirement: Ss grep filters ss rows server-side
The `ss_grep` tool MUST apply the same `state`/`family` selection and `show*` rules as `ss`, then filter data rows by `pattern` server-side. It MUST NOT spawn or execute host binaries. Pattern modes MUST follow the `list_grep` spirit as documented in `docs/tools/ss_grep.md`. Response metadata MUST use `[ss_grep …]`.

#### Scenario: Pattern finds wildcard binds
- **WHEN** `ss_grep` is invoked with a pattern intended to match addresses such as `0.0.0.0` or a port and matching sockets exist under the active filters
- **THEN** only matching rows MUST appear in the table

### Requirement: Ss grep preserves caps and show flags
Output MUST be capped at 1000 rows with truncation semantics aligned to `ss` + post-filter cap. The same `show*` flags as `ss` MUST apply; `Proto` and `Local` always present.

#### Scenario: Filtered established with peer column
- **WHEN** `ss_grep` is invoked with `state=ESTAB`, a matching `pattern`, and default `showPeer`
- **THEN** matching rows MUST include `Peer`

### Requirement: Ss grep docs include agent example prompt
`docs/tools/ss_grep.md` MUST document the compose contract and MUST include `## Prompt de ejemplo (agente)` with at least one Spanish-neutral prompt exercising `ss_grep` via linux-mcp. The MCP tool description MUST cover server-side filtering and metadata.

#### Scenario: Docs include example agent prompt
- **WHEN** an operator opens `docs/tools/ss_grep.md`
- **THEN** the document MUST contain `## Prompt de ejemplo (agente)` and at least one usable example prompt referencing `ss_grep`

### Requirement: Ss grep inherits foreign Pid resolution under reference deploy
The `ss_grep` tool MUST use the same socket inventory and Pid/Process resolution path as `ss`. Under the reference systemd unit, when `showPid`/`showProcess` are enabled and a filtered row corresponds to a socket owned by a non-`mcp-agent` process visible via `/proc`, the row MUST include the resolved Pid/Process just as `ss` would. `docs/tools/ss_grep.md` MUST note that Pid/Process follow `ss` (including reference-deploy expectations) and MUST keep `## Prompt de ejemplo (agente)` with at least one Spanish-neutral prompt that filters by port or address and expects process identity when available.

#### Scenario: Grep hit on foreign listener includes Pid
- **WHEN** linux-mcp runs with the reference unit privileges and `ss_grep` matches a LISTEN socket owned by a non-`mcp-agent` process with `showPid` enabled
- **THEN** the matching row MUST include a non-empty Pid

#### Scenario: Ss grep docs mention Pid and example prompt
- **WHEN** an operator opens `docs/tools/ss_grep.md`
- **THEN** the document MUST state that Pid/Process resolution follows `ss` under the reference deploy, and MUST contain `## Prompt de ejemplo (agente)` with a usable prompt referencing `ss_grep`
