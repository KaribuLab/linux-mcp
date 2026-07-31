## Why

Hoy linux-mcp solo da ojos de filesystem (`cat`/`list`/`find`/`grep` y composiciones). Un agente que audita seguridad o configuración de procesos/red en ejecución no tiene inventario runtime: empieza a barrer `/etc` y gasta tokens. Hace falta el mismo patrón de tools acotadas + `*_grep` + columnas `show*` que ya existe en `list`, leyendo el kernel **sin ejecutar binarios del OS**.

## What Changes

- Cuatro tools MCP nuevas (dos bases + dos composiciones server-side):
  - `ps` / `ps_grep` — inventario de procesos vía `/proc`, tabla markdown + filtro de filas.
  - `ss` / `ss_grep` — sockets vía netlink (inet diag) en proceso, tabla amplia (no solo LISTEN) + filtro; defaults tipados estrechos (`state`/`family`) para poco ruido; el agente amplía cuando quiere.
- Columnas opcionales con flags `show*` al estilo `list` (default visible; columna identidad siempre presente; meta `columns=`; orden fijo).
- Caps duros, headers `[ps …]` / `[ss …]` y variantes `_grep`, sin `os/exec` de binarios del host, sin shell, sin DSL `pipe`.
- Documentación nueva en `docs/tools/` para las cuatro tools + filas en `docs/README.md`.
- Estándar de docs: cada `docs/tools/<nombre>.md` incluye sección `## Prompt de ejemplo (agente)` con prompt(s) pegables (ej. `Usa el tool linux-mcp \`ps\` para …`). Queda en `openspec/config.yaml`; este change lo aplica en las cuatro docs nuevas y actualiza la sección Convención de `docs/README.md`.

### Non-goals

- **No tools `services` / `services_grep` en esta v1.** Inventario fiel de units systemd sin `systemctl` implica D-Bus al manager (scope, deps, permisos de `mcp-agent`). Se deja para una propuesta futura solo-D-Bus; con `ps` (+ `ss`) el agente ya ancla procesos/red y sigue con `cat`/`grep` en configs.
- No ejecutar **nunca** binarios del OS (`systemctl`, `ss`, `ps`, `/bin/sh`, etc.) — ni como fallback.
- No `listen`-only como tool separada (la superficie LISTEN es default/`state` de `ss`).
- No `journalctl` / logs / packages / capabilities en esta v1.
- No meta-tool `pipe`.
- No regularizar docs de tools ya existentes (`cat`, `list`, …) en este change — eso va en una propuesta aparte.
- No cambiar contratos de tools FS actuales.

## Capabilities

### New Capabilities

- `ps-safe-list`: tool `ps` — listado acotado de procesos vía `/proc`, columnas `show*`, caps, meta, descripción MCP, doc + prompt de ejemplo.
- `ps-grep-compose`: tool `ps_grep` — filtro server-side sobre filas de `ps` (mismo formato), doc + prompt de ejemplo.
- `ss-safe-list`: tool `ss` — listado acotado de sockets vía netlink, predicados `state`/`family`, `show*`, doc + prompt de ejemplo.
- `ss-grep-compose`: tool `ss_grep` — filtro server-side sobre filas de `ss`, doc + prompt de ejemplo.

### Modified Capabilities

- (ninguno a nivel de requisitos de tools FS existentes; la convención de prompts vive en `openspec/config.yaml` + `docs/README.md` y se verifica en los specs nuevos de cada tool).

## Impact

- Código nuevo: `internal/tool/ps.go`, `ps_grep.go`, `ss.go`, `ss_grep.go` (+ tests); headers en `internal/toolmeta`.
- Dep nueva (solo `ss`): `github.com/florianl/go-diag` (+ transitivas `mdlayher/netlink`, `golang.org/x/sys`); `ps` sin deps de terceros. **Sin** invocar binario `ss`. Criterio: superficie estrecha y sin advisories directos conocidos (ver design D5b).
- Código modificado: `internal/handler/server.go` (registro de las cuatro tools).
- Docs nuevas: `docs/tools/ps.md`, `ps_grep.md`, `ss.md`, `ss_grep.md`.
- Docs tocadas: `docs/README.md` (tabla tools + convención de prompt de ejemplo), `README.md` (tabla tools si aplica).
- OpenSpec: `openspec/config.yaml` (estándar de prompt de ejemplo en docs de tools — ya aplicado).
- Permisos runtime: lectura `/proc` y netlink según el usuario del proceso; fallos → error/`[blocked]` claro, sin inventar filas.
- Breaking: ninguno (solo tools nuevas + convención de docs).
