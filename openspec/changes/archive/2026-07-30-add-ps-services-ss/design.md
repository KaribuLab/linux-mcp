## Context

linux-mcp hoy es “ojos de FS”: `cat`/`list`/`find`/`grep` + `list_grep`/`find_grep`. Patrón establecido: respuesta acotada (meta + tabla/filas), caps, descripción MCP rica, docs en `docs/tools/`, composición nombrada `*_grep` (no DSL `|`).

Falta inventario runtime (procesos + sockets) sin barrer discos y **sin ejecutar binarios del OS**. Este change añade `ps`/`ss` y sus `*_grep`, con `show*` como `list`.

`services`/`services_grep` quedan fuera de esta v1 (ver D5 / Non-Goals).

Fuente de verdad pública prevista: `docs/tools/{ps,ps_grep,ss,ss_grep}.md`.

## Goals / Non-Goals

**Goals:**
- Cuatro tools MCP registradas en `NewHandler`, mismo transporte/auth.
- Tablas markdown + meta `[tool …]` + `columns=` cuando hay `show*`.
- Predicados tipados baratos (`state`/`family`/`includeKernel`) + `*_grep` para filtro texto server-side.
- Defaults de `ss` estrechos (poco ruido) sin limitar el modelo mental a “solo listen”.
- Docs con sección estándar `## Prompt de ejemplo (agente)`.
- Solo interfaces kernel/FS en-proceso: `/proc` y netlink. **Prohibido** `os/exec` / spawn de binarios del host.

**Non-Goals:**
- Tools `services` / `services_grep` (systemd units) en esta v1.
- Cualquier fallback que invoque `systemctl`, `ss`, `ps`, shells u otros binarios.
- `journal`, packages, capabilities, `listen` como tool separada.
- DSL `pipe` / stages extra.
- Backfill de prompts en docs de tools FS antiguas.
- Cambiar contratos de tools existentes.

## Decisions

### D1 — Dos bases + dos `*_grep` (no “audit”, no services aún)

Nombres MCP: `ps`, `ps_grep`, `ss`, `ss_grep`.

Alternativa: incluir `services` vía `systemctl`. Rechazada — viola la regla de no ejecutar binarios.  
Alternativa: incluir `services` vía D-Bus en esta v1. Aplazada — factible sin exec, pero suma dependencia, permisos de bus para `mcp-agent` y superficie de tests; el usuario prioriza v1 con `ps` (+ `ss`). Futuro change: solo D-Bus, nunca `systemctl`.

### D2 — Columnas `show*` idénticas en espíritu a `list`

- Cada columna opcional: `showX` bool, default `true` (omitido = visible).
- Columna(s) identidad **siempre** visibles, sin flag para ocultarlas.
- Orden fijo; flags solo inclusión/omisión.
- Meta incluye `columns=<c1,c2,...>` con el set efectivo.
- `*_grep` acepta los mismos `show*` que su base.

### D3 — Identidad y columnas por tool (v1)

**`ps` / `ps_grep`**
- Siempre: `Pid`, `Comm` (nombre corto de `/proc/pid/comm` o equivalente).
- `showPpid`, `showUser`, `showStat`, `showCpu`, `showMem`, `showCmdline`, `showExe`.
- Default filas: procesos de usuario visibles vía `/proc` (excluir kernel threads salvo `includeKernel=true`).
- Cap: 1000 filas. Cmdline/Exe truncados con indicador si hace falta (documentar límite).
- `showMem`: una sola columna, valor **RSS en KiB** (entero); no columna `%mem` separada en v1.

**`ss` / `ss_grep`**
- Siempre: `Proto`, `Local`.
- `showState`, `showPeer`, `showPid`, `showProcess`, `showUser`, `showFamily`.
- Predicados: `state` (`LISTEN` default | `ESTAB` | `all` | lista documentada), `family` (`inet` default = IPv4+IPv6 | `inet4` | `inet6` | `unix` | `all`).
- Cap: 1000 sockets. No tool `listen` separada.

### D4 — Semántica de `*_grep` = filtro de filas (como `list_grep`)

Misma lógica de patrón que `list_grep` (`extended` / `ignoreCase` / auto-glob cuando aplica sobre columna identidad: `Comm` o parte relevante de `Local` — documentar por tool).

- Filtra **filas de datos** de la tabla, no la meta ni headers markdown.
- Meta propia: `[ps_grep …]`, `[ss_grep …]`.
- `truncated=true` si la base truncó o el post-filtro supera el cap.

### D5 — Fuentes de datos: cero binarios del OS

Regla dura: **ninguna** tool de este change (ni helpers) puede usar `os/exec` ni spawnear `systemctl`, `ss`, `ps`, `/bin/sh`, etc.

| Tool | Fuente única | Notas |
|------|--------------|-------|
| `ps` | Lectura `/proc` en Go (stdlib) | Sin lib de terceros; sin fallback CLI |
| `ss` | Netlink sock_diag vía dependencia Go (D8) | Sin fallback al binario `ss` |
| `services` | **Fuera de v1** | Futuro: solo D-Bus a systemd; nunca `systemctl` |

Errores de permiso/netlink → texto de error claro o `[blocked class=…]` cuando aplique; nunca inventar filas.

Por qué no “services vía cgroup FS” en v1: incompleto (failed/enabled/FragmentPath poco fiable solo con cgroup) y confunde al agente respecto a `systemctl`. Mejor omitir que entregar inventario mentiroso.

### D5b — Dependencias de terceros (criterio de riesgo)

Preocupación: CVEs en libs. Criterio acordado:

1. Preferir **cero deps** cuando el kernel/FS alcanza (`ps` → `/proc` + stdlib).
2. Si hace falta lib: elegir una con **superficie estrecha**, **sin advisories directos conocidos**, y de preferencia usada en ecosistema serio.
3. Nunca compensar con spawn de binarios del OS.

**Decisión `ss`:** usar `github.com/florianl/go-diag` (API C-binding-free sobre sock_diag / estadísticas de sockets; MIT). Motivos:

- Superficie **solo diag/dump** (no API de mutar rutas/ifaces como `vishvananda/netlink`).
- deps.dev: **sin advisories directos** (a fecha de esta decisión).
- Apoyada en `mdlayher/netlink` + `golang.org/x/sys` (stack netlink habitual en Go).

Alternativa rechazada v1: reimplementar inet_diag a mano — menos deps, pero alto riesgo de bugs propios en parsing netlink.  
Alternativa de respaldo en apply (solo si `go-diag` no cubre `family`/`unix`/`ESTAB` necesarios): `vishvananda/netlink` **solo** APIs de dump/socket diag (historial largo desde ~2014, sin advisories directos en deps.dev); documentar “read-only usage” y no llamar mutadores.

Pin de versión en `go.mod`; en CI/release revisar advisories (`govulncheck` / deps.dev) antes de subir.

### D6 — Registro y docs + prompts de ejemplo

```go
tool.AddPsTool(server)
tool.AddPsGrepTool(server)
tool.AddSsTool(server)
tool.AddSsGrepTool(server)
```

en `internal/handler/server.go`.

Cada `docs/tools/<nombre>.md` MUST incluir:

```markdown
## Prompt de ejemplo (agente)

Usa el tool linux-mcp `<nombre>` para …
```

Al menos un prompt por tool, en español neutro. `docs/README.md` documenta el estándar. Regularización de docs antiguas = change aparte.

### D7 — Firmas (resumen)

**`ps`**: `includeKernel?`, `showPpid?` … `showExe?`  
**`ps_grep`**: args de `ps` + `pattern` (req) + `extended?` + `ignoreCase?`  
**`ss`**: `state?`, `family?`, `showState?` … `showFamily?`  
**`ss_grep`**: args de `ss` + `pattern` + `extended?` + `ignoreCase?`

Descripciones MCP: constantes `*ToolDescription` con contrato completo (meta, tabla, caps, blocked/error, `show*`, y que la fuente es `/proc` o netlink — no CLI).

## Risks / Trade-offs

- **[Risk] Netlink/inet_diag complejo o permisos insuficientes** → Mitigación: fail-closed; tests con fixtures/netns si aplica; documentar qué ve `mcp-agent`.
- **[Risk] CVE futuro en `go-diag` / `mdlayher/netlink`** → Mitigación: criterio D5b; pin + `govulncheck` en CI; superficie estrecha vs lib “navaja suiza”.
- **[Risk] Sin `services`, agente confunde proceso ↔ unit** → Mitigación: docs/prompts enseñan puente `ps`/`ss` → paths de config conocidos + `find`/`grep`; future D-Bus change.
- **[Risk] Cuatro tools más en schema MCP** → Mitigación: aceptable vs dump de `/proc` al contexto.
- **[Risk] Cmdline/Exe largos queman tokens** → Mitigación: truncación dura + `show*=false`.
- **[Trade-off] Default `ss state=LISTEN`** → menos ruido; agente amplía a `ESTAB`/`all`.
- **[Trade-off] Excluir kernel threads por default en `ps`** → menos ruido; `includeKernel` para forense.
- **[Trade-off] Dep `go-diag` (joven) vs `vishvananda/netlink` (madura, API ancha)** → preferimos estrechez + sin advisories; respaldo documentado en D5b.

## Migration Plan

- Deploy: binario nuevo registra cuatro tools; clients viejos las ignoran.
- Rollback: quitar registro; sin migración de datos.
- Sin **BREAKING**.

## Open Questions

- Ninguna bloqueante. Detalle menor en apply: cobertura exacta de `go-diag` para unix sockets / mapeo state → si falta algo crítico, activar respaldo D5b (`vishvananda/netlink` dump-only) y documentar en la PR.
