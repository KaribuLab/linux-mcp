## ADDED Requirements

### Requirement: Ss resolves Pid and Process for foreign owners under reference deploy
When `showPid` and/or `showProcess` are enabled (default true), the `ss` tool MUST resolve socket owners by matching netlink inode to `/proc/<pid>/fd` targets of the form `socket:[inode]` using in-process reads only (MUST NOT spawn host binaries). Under the reference systemd unit (capabilities and ProtectProc as specified in `systemd-install`), this resolution MUST succeed for listening (and other listed) sockets whose owning process runs as a different uid than `mcp-agent`, when the kernel permits the read. The tool MUST NOT invent Pid or Process values when resolution fails; empty or placeholder cells remain acceptable outside that deploy or when the kernel denies access.

`docs/tools/ss.md` MUST describe this resolution path and the reference-deploy expectation, and MUST keep `## Prompt de ejemplo (agente)` with at least one Spanish-neutral prompt that exercises obtaining a socket's Pid (or crossing port to process) via linux-mcp `ss`.

#### Scenario: Foreign listener shows Pid under reference privileges
- **WHEN** linux-mcp runs with the reference unit privileges and `ss` is invoked with default or explicit `showPid=true` against a LISTEN socket owned by a non-`mcp-agent` process that is visible via `/proc`
- **THEN** the corresponding table row MUST include a non-empty Pid for that socket

#### Scenario: Docs describe Pid resolution and example prompt
- **WHEN** an operator opens `docs/tools/ss.md`
- **THEN** the document MUST mention inode/`/proc/*/fd` resolution and the reference systemd expectation, and MUST contain `## Prompt de ejemplo (agente)` with a usable prompt referencing `ss`
