# Tool: `cat`

Lee un archivo de texto con salida acotada (stream; nunca carga el archivo completo en un `ReadFile` unbounded) y lo devuelve al cliente MCP.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `cat` |
| Código | `internal/tool/cat.go` |
| Registro | `AddCatFileTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.CatToolDescription` (meta `[cat …]`, body texto crudo, `offset`/`next` en bytes, `[blocked …]`) |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `path` | `string` | sí | Ruta absoluta o relativa del archivo |
| `offset` | `int64` | no | Cursor de **byte** para reanudar (`next` de una respuesta truncada previa). `0`/omitido = desde el inicio |

## Caps (v1)

- Máx. **100 líneas** y **64 KiB** (corta el que se alcance primero).
- Sin cache del archivo completo entre llamadas.
- Resume: `Seek` al `offset` cuando el fd es seekable.

## Respuesta

Un solo `TextContent`.

### Éxito

```text
[cat path=<abs> lines=<n> truncated=<bool> next=<byte>]
<body texto crudo>
```

- Si `truncated=false`, se omite `next`.
- Si `truncated=true`, volver a llamar con el mismo `path` y `offset=<next>`.
- El body **no** es tabla markdown ni JSON por línea.

### Bloqueo (IsError)

```text
[blocked class=<class> path=<abs>]
```

Clases: `path_denied`, `private_key`, `binary`, `type_denied`.

## Comportamiento

1. `policy.CheckPath` (denylist in-process; no depende de systemd).
2. Abre el archivo; rechaza devices/sockets/dirs (`type_denied`).
3. Sniff desde el **inicio** del archivo (primera línea útil): headers de private key → `private_key`; NUL → `binary`.
4. `Seek(offset)` si `offset > 0`; si Seek falla → error de resume.
5. Stream hasta caps; meta vía `toolmeta.CatHeader` + body en `strings.Builder`.

Path deny extra (además del sniff): `/etc/shadow`, `/etc/gshadow`, `mem`/`kcore`, `.ssh/id_*` (no `.pub`), `*.pem`.

## Ejemplos

**Primera página**

```json
{ "path": "/etc/hostname" }
```

**Página siguiente**

```json
{ "path": "/var/log/syslog", "offset": 4096 }
```

## Notas / límites

- Pensada para texto; binarios se rechazan.
- `/proc`/`/sys` text-like permitidos con los mismos caps; resume puede fallar si no son seekable.
- Relacionada: `internal/policy`, `internal/toolmeta`.
