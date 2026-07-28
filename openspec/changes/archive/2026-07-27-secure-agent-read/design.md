## Context

linux-mcp expone `cat` y `list` vía MCP (Streamable HTTP, `localhost:5000`). El agente objetivo es ops/audit con lectura amplia del FS (incl. `/proc`, `/sys`), a menudo bajo systemd con `CAP_DAC_READ_SEARCH`. Hoy:

- `cat` usa `os.ReadFile` sin límites ni policy.
- `list` no aplica path policy; `Readlink` usa solo el basename (CWD); grupo se resuelve con `LookupId`.
- No hay capa compartida para futuras tools (`ps`, `grep`, `find`).

La policy de lectura MUST vivir en el proceso Go. Systemd es defensa opcional del operador, no dependencia.

Documentación pública de comportamiento: `docs/tools/cat.md`, `docs/tools/list.md`.

## Goals / Non-Goals

**Goals:**

- Default-allow de lectura con denylist de paths/tipos peligrosos en código.
- `cat` con stream + tope líneas ∩ bytes; sniff private key en primera línea útil; meta mínima una línea + body.
- `list` con misma path policy, tope de entradas, fixes symlink/grupo.
- Paquete reutilizable (`internal/policy`) para tools futuras.
- Docs alineadas al comportamiento real.
- Unit systemd de referencia + runbook de instalación en el repo.

**Non-Goals:**

- Tools `ps`/`grep`/`find` en este change.
- Sniff de tokens cloud.
- Allowlist de roots.
- Auth HTTP / cambios CORS.

## Decisions

### D1: Paquete `internal/policy` como única puerta de lectura

- **Qué:** APIs compartidas, p.ej. `CheckPath(path)`, `ClassifyPrefix(prefix []byte)`, `OpenLimited(path, limits)`, constantes de deny.
- **Por qué:** Evita duplicar reglas en cada tool; `grep`/`find` nacen ya acotados.
- **Alternativa:** Logic inline en `cat.go`/`list.go` → rechazada (duplicación).

### D2: Default-allow + denylist (no allowlist de roots)

- **Qué:** Permitir gran parte del FS; denegar paths/tipos sensibles conocidos.
- **Deny inicial (mínimo):** `/etc/shadow`, `/etc/gshadow`, `**/mem`, `kcore`, device nodes; dirs/patrones de keys por path como capa extra (`.ssh/id_*`, `*.pem` opcional); `/proc/*/mem` y similares.
- **Por qué:** Rol ojos-de-servidor; allowlist estrecha ciega al agente.
- **Alternativa:** Allowlist roots → rechazada para este producto.

### D3: Sniff de private key en primera línea útil

- **Qué:** Tras abrir, leer prefijo acotado; saltar BOM/líneas vacías; evaluar primera línea con contenido.
- **Match → block** (sin devolver body): `BEGIN OPENSSH PRIVATE KEY`, `BEGIN * PRIVATE KEY`, `BEGIN ENCRYPTED PRIVATE KEY`, `BEGIN PGP PRIVATE KEY BLOCK`, header `PuTTY-User-Key-File`.
- **No block:** public keys / `BEGIN PUBLIC KEY`.
- **Error barato en tokens:** una línea `[blocked class=private_key path=…]`.
- **Por qué:** Nombres de archivo arbitrarios; README con ejemplo en el medio no dispara.
- **Alternativa:** Solo path patterns → insuficiente.

### D4: Caps de salida en `cat` (stream, nunca `ReadFile` completo)

- **Límites v1 (cerrados):** max **100 líneas** **y** max **64 KiB** (el que corte primero); `list` max **1000** entradas. Constantes en código (`policy.Limits`); sin caps dinámicos por RAM del host (el cuello es tokens del agente, no recursos del server).
- **Tipos:** preferir archivos regulares; `/proc`/`/sys` text-like permitidos con los mismos caps (no confiar en `stat` size).
- **Binario:** null en prefijo → **reject** con `[blocked class=binary …]`.
- **Sin cache/buffer del archivo completo** en proceso entre calls: contradice stream+caps, stale en logs/`/proc`, riesgo RAM/secretos. Cada call abre, lee acotado, cierra.

### D5: Formato de respuesta `cat` — un `TextContent`, meta mínima + resume por Seek

Firma tool (sin cambio de nombre):

| Campo | Tipo | Notas |
|-------|------|--------|
| `path` | string | requerido |
| `offset` | int (opcional) | **cursor de byte** (no línea). `0`/omitido = desde el inicio. En resume, el agente pasa el `next` de la meta previa. |

**Resume (v1):** tras abrir, si `offset > 0` y el fd es seekable → `Seek(offset)` y leer solo la página (líneas ∩ bytes). Evita skip-líneas O(n²) y evita cache full-file. Si Seek falla (p.ej. algunos `/proc`) → leer desde el inicio descartando hasta el cursor solo si es barato/documentado, o devolver error claro de resume no soportado; no inventar sesión stateful en server.

**Description MCP** (`mcp.Tool.Description`): MUST explicar al agente el **contrato completo de salida** — no solo “Read the contents of a file”. Incluir:

1. Lectura acotada (100 líneas ∩ 64 KiB); no archivo completo garantizado.
2. Forma éxito: primera línea meta `[cat …]` **y debajo el body en texto crudo** (no tabla markdown; no JSON por línea).
3. Campos meta: `path`, rango/conteo de líneas de la página, `truncated`, y si aplica `next=<byte>` para re-llamar con `offset`.
4. Bloqueo: una línea `[blocked class=… path=…]` sin body; clases al menos `path_denied`, `private_key`, `binary`.
5. Cómo paginar: si `truncated=true`, volver a llamar con el mismo `path` y `offset=<next>`.

Borrador de tono (ajustar en código; mantener corto pero completo):

```text
Read a text file with bounded output (max 100 lines and 64KiB). On success the
first line is metadata: [cat path=... lines=... truncated=bool next=<byte-or-empty>]
followed by the raw text body (not markdown). Pass next as offset to resume via
Seek when truncated. On policy/content block returns a single line
[blocked class=... path=...] with no file body (classes include path_denied,
private_key, binary). Does not dump full large or binary files; no server-side
file cache.
```

Respuesta éxito (un string):

```text
[cat path=<abs> lines=<n-or-range> truncated=<bool> next=<byte-or-empty>]
<body>
```

- Si `truncated=false`, meta puede omitir `next`.
- `next` / arg `offset` = **bytes**, no número de línea (las líneas en meta son informativas de la página).
- Sin JSON por línea; sin array structured de filas; sin cache entre requests.
- Fuente de verdad pública: `docs/tools/cat.md` (debe coincidir con Description MCP).

Error policy/sniff: un string `[blocked class=… path=…]` (tool error o content IsError según convención SDK existente — preferir error de tool claro y documentado en Description).

### D6: `list` — misma path policy + caps + fixes + meta header

Firma sin cambio de args (`path`, `all`, `list`).

- Path deny antes de `ReadDir`.
- Cap de entradas: **1000** (cerrado v1).
- **Misma convención que `cat`:** primera línea siempre meta `[list …]`; debajo la **tabla markdown** (modo simple `|File|` o detallado con columnas Name/Size/…).
- Symlink: `os.Readlink(filepath.Join(dir, name))`.
- Grupo: `user.LookupGroupId` (o API correcta), no `LookupId` con GID.
- Docs: `docs/tools/list.md` + **Description MCP** que explique **todo el patrón de respuesta**.

**Description MCP** MUST incluir (no basta “List files in markdown”):

1. Éxito: línea `[list …]` y **después** tabla markdown (mencionar modos `list=false` vs `list=true` a alto nivel).
2. Truncado/`next` de entradas si aplica; cap 1000.
3. `[blocked class=… path=…]` sin filas.

Respuesta éxito:

```text
[list path=<abs> entries=<returned>/<total-or-unknown> truncated=<bool> next=<entry-offset-or-empty>]
|Name|...
```

- `truncated=false` → meta corta OK (`entries=<n>` sin `next`).
- `truncated=true` → `truncated=true` + `next` (offset de **entrada** para página siguiente; distinto del `offset` byte de `cat`).
- Deny: `[blocked class=path_denied path=…]` (misma familia que `cat`).
- Pocos tokens: una línea meta, no JSON por fila.

### D9: Headers tipados + body con `strings.Builder`

- **Paquete:** `internal/toolmeta` (presentación; no mezclar con `internal/policy`).
- **Contrato:** cada tool define un struct de header que implementa `fmt.Stringer` (`String() string` → una línea `[cmd k=v …]`). `Blocked` compartido para denegaciones.
- **Body:** las tools acumulan el cuerpo en `strings.Builder` (filas de `list`, líneas de `cat`). No concatenar con `+` ni pasar el body como `string` intermedio mientras se construye.
- **Ensamblado:** helper del estilo:

```go
func Render(h fmt.Stringer, body *strings.Builder) string {
    // prefijo h.String() + "\n" + body.String() si body tiene contenido
}
```

  o `WriteTo(w io.Writer)` si en implementación conviene un solo buffer: escribir header, `\n`, luego `body` — evita segundo string grande innecesario cuando el SDK acepte `[]byte`/writer; v1 puede devolver `string` final al MCP tras un único `Builder` que ya incluya header+body.
- **Preferencia v1:** un `strings.Builder` de respuesta: `WriteString(header.String())`, `WriteByte('\n')`, volcar/append del body builder (o escribir body directo en el mismo builder tras el header).
- **Por qué:** menos allocs en listados/cats truncados; formato de meta testeable por struct; tools futuras reutilizan el patrón.
- **Alternativa rechazada:** `Render(h, body string)` con body ya materializado — fácil, pero peores copias en hot path.

### D7: Registro y transporte

- Tools siguen registrándose en `NewHandler` vía `AddCatFileTool` / `AddListFilesTool`.
- Sin cambio de transporte Streamable HTTP / CORS en este change.
- Config de límites: constantes fijas v1 (100 / 64 KiB / 1000); flags CLI opcionales solo si no alargan el change.

### D8: Unit systemd + runbook en el repo

- **Unit:** `deploy/systemd/linux-mcp.service`
  - `User=`/`Group=mcp-agent`
  - `ExecStart=/usr/local/bin/linux-mcp`
  - `AmbientCapabilities=` / `CapabilityBoundingSet=CAP_DAC_READ_SEARCH`
  - Hardening de escritura (`ProtectSystem`, `ProtectHome`, `PrivateTmp`, syscall filter, etc.)
  - `InaccessiblePaths` grueso (`/etc/shadow`, `/etc/gshadow`, `/root`) — complemento, no sustituto de policy app
- **Runbook:** `docs/runbooks/install-systemd.md` — crear usuario, build/install binario, install unit, enable/start, update, uninstall, troubleshooting
- **Por qué en repo:** operador no inventa la unit; alguien sin systemd igual tiene policy en código
- README / `docs/README.md` enlazan el runbook

## Risks / Trade-offs

| Riesgo | Mitigación |
|--------|------------|
| Falso positivo sniff en docs que empiezan con header PEM | Raro; aceptar; primera línea útil reduce ruido |
| Falso negativo (key ofuscada / sin header) | Path deny + ops systemd `InaccessiblePaths` opcionales |
| `/proc` size 0 / files que bloquean | Stream + max bytes + timeout; no usar size como gate único |
| Denylist incompleta | Extensible; audit futuro; no pretender cobertura total |
| **BREAKING** para clientes que esperan body completo | Documentar en docs/tools; meta `truncated`/`next` |
| Skip-líneas O(n²) si `offset` fuera por línea | **Mitigado:** `offset`/`next` = byte + `Seek`; sin cache full-file |
| Resume en fs no seekable (`/proc`) | Fallback documentado / error claro; no sesión stateful |

## Migration Plan

1. Implementar `internal/policy` + `internal/toolmeta` (headers/`Render` con Builder) + tests.
2. Cablear `cat` y `list`; actualizar docs.
3. Verificar a mano con MCP Inspector / cliente: archivo chico, archivo grande, PEM, path deny, list dir enorme.
4. Rollback: revertir change; no hay migración de datos.

## Open Questions

Ninguna abierta. Cerrado:

- **Resume `cat`:** arg `offset` + meta `next` = **byte** con `Seek` cuando seekable; no offset por línea; no buffer/cache del archivo completo en proceso.
- **Defaults v1:** 100 líneas ∩ 64 KiB (`cat`); 1000 entradas (`list`); documentar en Description MCP + `docs/tools/*`.
- **Patrón de respuesta en la tool:** sí — `mcp.Tool.Description` MUST describir el contrato completo (meta + body/`tabla markdown` + `[blocked …]` + paginación). Tasks 2.2 / 3.2 y specs `cat-safe-read` / `list-safe-read` lo exigen.
