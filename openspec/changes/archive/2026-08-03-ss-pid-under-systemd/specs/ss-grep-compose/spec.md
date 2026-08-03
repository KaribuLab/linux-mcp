## ADDED Requirements

### Requirement: Ss grep inherits foreign Pid resolution under reference deploy
The `ss_grep` tool MUST use the same socket inventory and Pid/Process resolution path as `ss`. Under the reference systemd unit, when `showPid`/`showProcess` are enabled and a filtered row corresponds to a socket owned by a non-`mcp-agent` process visible via `/proc`, the row MUST include the resolved Pid/Process just as `ss` would. `docs/tools/ss_grep.md` MUST note that Pid/Process follow `ss` (including reference-deploy expectations) and MUST keep `## Prompt de ejemplo (agente)` with at least one Spanish-neutral prompt that filters by port or address and expects process identity when available.

#### Scenario: Grep hit on foreign listener includes Pid
- **WHEN** linux-mcp runs with the reference unit privileges and `ss_grep` matches a LISTEN socket owned by a non-`mcp-agent` process with `showPid` enabled
- **THEN** the matching row MUST include a non-empty Pid

#### Scenario: Ss grep docs mention Pid and example prompt
- **WHEN** an operator opens `docs/tools/ss_grep.md`
- **THEN** the document MUST state that Pid/Process resolution follows `ss` under the reference deploy, and MUST contain `## Prompt de ejemplo (agente)` with a usable prompt referencing `ss_grep`
