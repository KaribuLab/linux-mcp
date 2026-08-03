## Context

`ss` / `ss_grep` listan sockets con netlink (`florianl/go-diag`) y, si `showPid`/`showProcess`, resuelven dueño con `buildSocketInodeMap()` caminando `/proc/*/fd` → `socket:[inode]` (`internal/tool/ss.go`). No hay Pid en sock_diag.

La unit `deploy/systemd/linux-mcp.service` corre como `mcp-agent` con:

- `ProtectProc=invisible` → otros uids no aparecen en `/proc` del servicio
- solo `CAP_DAC_READ_SEARCH` → leer `fd/` ajeno falla (chequeo tipo `ptrace_may_access`)

Resultado: columnas Pid/Process vacías en el deploy de referencia; el agente no puede cruzar puerto↔proceso con fiabilidad.

Perfil de producto acordado: agente **root solo-lectura**; el operador asume riesgo de Bearer filtrado (emisión sigue exigiendo host + `mcp-admin`).

Firma pública de tools (sin cambio de schema JSON):

| Tool | Args relevantes | Respuesta |
|------|-----------------|-----------|
| `ss` | `state`, `family`, `showPid`, `showProcess`, … | `[ss … columns=…]` + tabla; Pid/Process cuando resolución ok |
| `ss_grep` | igual + `pattern` | `[ss_grep …]` + filas filtradas |

Registro: sin cambio (`AddSsTool` / `AddSsGrepTool` en `NewHandler`). Transporte: Streamable HTTP + Bearer + CORS intactos.

Docs de verdad de comportamiento: `docs/tools/ss.md`, `docs/tools/ss_grep.md` (incl. `## Prompt de ejemplo (agente)`).

## Goals / Non-Goals

**Goals:**

- Bajo unit de referencia, `ss`/`ss_grep` rellenan Pid/Process para sockets de procesos de **cualquier** uid visible al kernel vía `/proc/*/fd`.
- Documentar amenaza y procedimiento de update de unit.
- Validar en host real (o checklist) que un LISTEN ajeno muestra Pid ≠ vacío.

**Non-Goals:**

- Tool MCP de lectura de `/proc/<pid>/mem` o attach debugger.
- Cambiar denylist FS, escritura, o spawn de `ss`/`lsof`.
- Garantizar Pid en installs sin systemd/caps (ahí sigue fail-soft).

## Decisions

### D1 — Relajar ProtectProc + añadir CAP_SYS_PTRACE (no servicio aparte)

- **Qué:** En la unit de referencia: quitar `ProtectProc=invisible` (default) y añadir `CAP_SYS_PTRACE` a `AmbientCapabilities` y `CapabilityBoundingSet` junto a `CAP_DAC_READ_SEARCH`.
- **Por qué:** Sin ver `/proc` ajeno no hay mapa inode; sin cap ptrace-class no hay `Readlink` de `fd/` ajeno. Un solo proceso mantiene el modelo actual de tools.
- **Alternativas rechazadas:**
  - Solo `ProtectProc=default` sin ptrace → `ps` mejora, Pid en `ss` sigue vacío.
  - Segundo servicio privilegiado solo-ss → más ops, misma superficie vía API si se expone igual.
  - Heurística User+cmdline → ya falló en costo de tokens; no cumple “Pid como corresponde”.

### D2 — Sin cambios de código Go salvo docs/descripciones si hace falta

- **Qué:** Reusar `buildSocketInodeMap` / `attachProcOwners`. Ajustar texto de `SsToolDescription` / docs si el “when visible” debe mencionar el deploy de referencia.
- **Por qué:** El bug es de jaula, no de algoritmo.
- **Alternativa:** Reescribir con eBPF — fuera de alcance.

### D3 — No exponer mem; cap es poder del proceso, no API nueva

- **Qué:** Ningún tool nuevo lee `mem`/`maps`. La cap habilita la resolución inode usada hoy.
- **Por qué:** Producto = inventario read-only, no debugger remoto. Riesgo residual: compromiso del binario o tool futura abusando la cap — aceptar bajo perfil “root read-only” + documentar.

### D4 — Docs y prompts: flujo puerto → Pid → ps

- **Qué:** Runbook + `docs/tools/ss.md` / `ss_grep.md`: expectativa Pid bajo unit nueva; prompt de ejemplo que pida dueño de un puerto y cruce con `ps`/`ps_grep` usando Pid.
- **Por qué:** El valor del change es el agente deje de adivinar.

## Risks / Trade-offs

- [Bearer + API inspecciona procs ajenos (fd; cap permite más si se abusa del proceso)] → Mitigación: TTL corto, reinicio invalida tokens, bind loopback, `mcp-admin` chico; documentar en runbook. No tool de mem.
- [Yama `ptrace_scope` u otras políticas del host bloquean igual] → Mitigación: runbook troubleshooting; verificar caps efectivas con `getpcaps` / prueba `ss` showPid.
- [Update olvidado: unit vieja en `/etc`] → Mitigación: runbook ya exige copiar unit antes de restart; añadir nota explícita de caps/ProtectProc.
- [Tests CI no corren como mcp-agent con caps] → Mitigación: tests unitarios del mapa con fake `/proc` siguen; checklist manual o doc de verificación en host.

## Migration Plan

1. Merge change (unit + docs + specs).
2. En cada host: copiar `deploy/systemd/linux-mcp.service` → `/etc/systemd/system/`, `daemon-reload`, `restart linux-mcp` (tokens previos mueren).
3. Verificar: `ss` LISTEN con `showPid=true` muestra Pid de un servicio de otro uid (p.ej. correo/ssh).
4. Rollback: restaurar unit anterior (solo `CAP_DAC_READ_SEARCH` + `ProtectProc=invisible`) y restart — comportamiento ciego vuelve.

## Open Questions

- Ninguno bloqueante: perfil “root solo-lectura” + Pid obligatorio bajo referencia ya decidido por el operador.
