## Why

El perfil objetivo del agente es **root solo-lectura**: inventariar puertos y cruzarlos con procesos sin adivinar. Hoy `ss` ya obtiene sockets por netlink (puerto, User, inode), pero bajo la unit de referencia (`User=mcp-agent`, `ProtectProc=invisible`, solo `CAP_DAC_READ_SEARCH`) no puede resolver inode→Pid vía `/proc/*/fd` de procesos ajenos. El agente queda ciego al cruzar con `ps` y gasta tokens en heurísticas (caso real: MTA antiguo en :25).

## What Changes

- Ampliar la unit systemd de referencia para que `mcp-agent` pueda **ver** procesos ajenos y **resolver** dueños de sockets (inode → Pid/Process) como hace `ss -p` privilegiado, sin spawn de binarios.
- Documentar el tradeoff de amenaza: Bearer válido + API = inspección de procs ajenos al nivel fd/ptrace-class; sin exponer tool de dump de memoria.
- Actualizar runbook y docs de `ss` / `ss_grep` para que operadores y agentes esperen Pid/Process poblados bajo el deploy de referencia.
- **No** cambiar el modelo de tools (sigue netlink + `buildSocketInodeMap`); el gap es de **runtime/hardening**, no de contrato MCP ausente.

### Non-goals

- No añadir tool MCP que lea `/proc/<pid>/mem` ni dumps de memoria.
- No ejecutar el binario host `ss` / `lsof`.
- No relajar denylist de lectura de archivos ni escribir en el sistema.
- No exigir Pid en entornos no-systemd o sin las caps (fail soft: columnas vacías como hoy).

## Capabilities

### New Capabilities

- (ninguna)

### Modified Capabilities

- `systemd-install`: la unit de referencia MUST incluir `CAP_SYS_PTRACE` además de `CAP_DAC_READ_SEARCH`, y MUST NO usar `ProtectProc=invisible` (usar default o equivalente que permita ver entradas `/proc` de otros uids). El runbook MUST documentar por qué y el riesgo asumido.
- `ss-safe-list`: MUST dejar explícito que, con el deploy de referencia, `showPid`/`showProcess` resuelven dueños de sockets de procesos de otros usuarios cuando el kernel lo permite vía `/proc/*/fd` (sin fabricar Pids).
- `ss-grep-compose`: misma expectativa de resolución Pid/Process al filtrar, alineada con `ss`.

## Impact

- `deploy/systemd/linux-mcp.service` — caps + `ProtectProc`
- `docs/runbooks/install-systemd.md` — amenaza, verificación Pid, update de unit
- `docs/tools/ss.md`, `docs/tools/ss_grep.md` — notas de resolución bajo systemd; prompts de ejemplo si hace falta
- `openspec/specs/systemd-install`, `ss-safe-list`, `ss-grep-compose` — requisitos
- Código Go de `ss`: sin cambio funcional esperado; validar en host con unit nueva
- Operadores: **BREAKING** para quien copie la unit anterior — tras update hay que reinstalar unit y reiniciar; superficie de token aumenta (inspección de procs)
