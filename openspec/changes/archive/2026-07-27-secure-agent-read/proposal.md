## Why

El servidor MCP da ojos de lectura amplia a un agente de ops/audit (incl. `/proc`, `/sys`, configs). Hoy `cat` carga archivos enteros con `os.ReadFile` y no hay política de paths ni detección de secretos en código: riesgo de OOM/tokens y de filtrar llaves privadas o archivos sensibles. La policy debe vivir en la app (no depender de systemd), porque el binario también puede correr a mano.

## What Changes

- Añadir paquete compartido de política de lectura (`internal/policy` o equivalente): denylist de paths, tipos de archivo permitidos, resolución segura.
- Añadir `internal/toolmeta`: headers tipados (`Stringer`) por tool + ensamblado de respuesta con `strings.Builder` (body sin concatenación `string`).
- Endurecer `cat`: lectura por stream con tope 100 líneas ∩ 64 KiB; resume por `offset`/`next` en **bytes** vía `Seek` (sin cache full-file ni skip-líneas O(n²)); sniff de la primera línea útil para bloquear llaves privadas (PEM/OpenSSH/PPK); respuesta `[cat …]` + body texto crudo; `Tool.Description` MCP MUST documentar el contrato completo (meta, body, paginación, `[blocked …]`).
- Endurecer `list`: misma path policy; tope 1000 entradas; meta `[list …]` + tabla markdown; Description MCP MUST documentar meta + forma markdown + blocked; corregir symlinks (`Join` con el dir) y lookup de grupo.
- Actualizar docs públicas: `docs/tools/cat.md`, `docs/tools/list.md` (y fila en `docs/README.md` si cambia el resumen).
- Añadir unit systemd de referencia (`deploy/systemd/linux-mcp.service`) y runbook de instalación (usuario OS, binario, enable/start, límites de la unit vs policy en app).
- **BREAKING** (comportamiento): `cat` ya no devuelve el archivo completo ni contenido de paths/clases bloqueadas; `list` puede omitir o rechazar paths denegados y truncar listados grandes.

## Non-goals

- Tools nuevas (`ps`, `grep`, `find`) — solo dejar la policy reutilizable para ellas.
- Detección de tokens cloud (AWS/GitHub/etc.) en contenido — solo private keys en v1.
- Allowlist estrecha de roots (el rol es default-allow con denylist + caps).
- Auth HTTP / endurecimiento CORS (fuera de este change).

## Capabilities

### New Capabilities

- `read-policy`: Política de lectura en proceso (path deny, tipos de archivo, sniff de private keys en prefijo, límites de salida reutilizables).
- `cat-safe-read`: Comportamiento seguro y acotado de la tool `cat` (stream, truncado, meta mínima, block por contenido).
- `list-safe-read`: Comportamiento seguro y acotado de la tool `list` (misma path policy, caps, fixes de symlink/grupo).
- `systemd-install`: Unit systemd de referencia y runbook para instalar el servicio (usuario `mcp-agent`, capabilities, enable).

### Modified Capabilities

- (ninguna — no hay specs main previas de tools)

## Impact

- Código: `internal/tool/cat.go`, `internal/tool/list.go`, nuevo `internal/policy/` (o nombre acordado en design), posible wiring mínimo en `internal/handler` solo si hace falta inyectar config.
- Docs: `docs/tools/cat.md`, `docs/tools/list.md`, `docs/README.md`, `docs/runbooks/install-systemd.md`.
- Deploy: `deploy/systemd/linux-mcp.service`.
- Runtime: mismos transporte Streamable HTTP y registro MCP; cambia contrato de salida/errores de `cat`/`list`.
- Operación: unit opcional con `CAP_DAC_READ_SEARCH`; policy en app sigue siendo obligatoria sin systemd.
