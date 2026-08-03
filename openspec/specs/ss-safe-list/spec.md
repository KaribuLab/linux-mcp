## Purpose

Safe, bounded `ss` tool: netlink sock_diag, show* columns, state/family filters, caps, and agent docs with example prompt.

## Requirements

### Requirement: Ss lists sockets with wide model and narrow defaults
The `ss` tool MUST expose a wide socket inventory (not listen-only as a separate tool) using in-process netlink (sock_diag / inet_diag or equivalent). It MUST NOT spawn or execute the host `ss` binary or any other host binary. Default `state` MUST be `LISTEN` and default `family` MUST be `inet` (IPv4+IPv6 as documented). Callers MUST be able to widen via `state` values including at least `LISTEN`, `ESTAB`, and `all`, and via `family` values including at least `inet`, `inet4`, `inet6`, `unix`, and `all`. Results MUST be capped at 1000 rows with `truncated=true` when exceeded.

#### Scenario: Default is listening inet sockets
- **WHEN** `ss` is invoked with no `state` or `family` arguments
- **THEN** the result MUST only include listening sockets in the documented `inet` families (subject to the cap)

#### Scenario: Agent widens to established connections
- **WHEN** `ss` is invoked with `state=ESTAB`
- **THEN** the result MUST include established sockets matching the active `family` filter and MUST NOT be limited to LISTEN-only

### Requirement: Ss supports show flags and always includes Proto and Local
Identity columns `Proto` and `Local` MUST always be present first (fixed order). Optional flags defaulting to visible: `showState`, `showPeer`, `showPid`, `showProcess`, `showUser`, `showFamily`. Flags MUST NOT hide identity columns or reorder columns. Metadata MUST include `columns=`.

#### Scenario: Hide peer to reduce tokens on listen scan
- **WHEN** `ss` is invoked with `showPeer=false` and default state LISTEN
- **THEN** the table MUST omit `Peer` but MUST still include `Proto` and `Local`

#### Scenario: Peer visible for established audit
- **WHEN** `ss` is invoked with `state=ESTAB` and default `showPeer`
- **THEN** the `Peer` column MUST be present so remote endpoints are visible

### Requirement: Ss response metadata and backend failure clarity
On success, first line MUST be `[ss …]` with counts, `truncated`, and `columns=`, followed by a markdown table. Backend/permission failures MUST fail closed without fabricated sockets.

#### Scenario: Successful ss includes metadata
- **WHEN** `ss` successfully lists sockets
- **THEN** the first line MUST be `[ss …]` including `columns=` and the body MUST be a markdown table

### Requirement: Ss MCP description and docs include agent example prompt
The MCP description MUST state that the tool is wide (`ss`), defaults are narrow (`LISTEN`/`inet`), and `show*`/filters control tokens. `docs/tools/ss.md` MUST include `## Prompt de ejemplo (agente)` with at least one Spanish-neutral prompt exercising `ss` via linux-mcp.

#### Scenario: Docs include example agent prompt
- **WHEN** an operator opens `docs/tools/ss.md`
- **THEN** the document MUST contain `## Prompt de ejemplo (agente)` and at least one usable example prompt referencing `ss`

### Requirement: Ss resolves Pid and Process for foreign owners under reference deploy
When `showPid` and/or `showProcess` are enabled (default true), the `ss` tool MUST resolve socket owners by matching netlink inode to `/proc/<pid>/fd` targets of the form `socket:[inode]` using in-process reads only (MUST NOT spawn host binaries). Under the reference systemd unit (capabilities and ProtectProc as specified in `systemd-install`), this resolution MUST succeed for listening (and other listed) sockets whose owning process runs as a different uid than `mcp-agent`, when the kernel permits the read. The tool MUST NOT invent Pid or Process values when resolution fails; empty or placeholder cells remain acceptable outside that deploy or when the kernel denies access.

`docs/tools/ss.md` MUST describe this resolution path and the reference-deploy expectation, and MUST keep `## Prompt de ejemplo (agente)` with at least one Spanish-neutral prompt that exercises obtaining a socket's Pid (or crossing port to process) via linux-mcp `ss`.

#### Scenario: Foreign listener shows Pid under reference privileges
- **WHEN** linux-mcp runs with the reference unit privileges and `ss` is invoked with default or explicit `showPid=true` against a LISTEN socket owned by a non-`mcp-agent` process that is visible via `/proc`
- **THEN** the corresponding table row MUST include a non-empty Pid for that socket

#### Scenario: Docs describe Pid resolution and example prompt
- **WHEN** an operator opens `docs/tools/ss.md`
- **THEN** the document MUST mention inode/`/proc/*/fd` resolution and the reference systemd expectation, and MUST contain `## Prompt de ejemplo (agente)` with a usable prompt referencing `ss`
