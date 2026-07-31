# Tool: `find_grep`

Descubre entradas con predicados de `find` y busca un patrón en el **contenido** de los archivos regulares hallados (semántica `find | xargs grep`). La salida es formato `grep`, no tabla `find`.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `find_grep` |
| Código | `internal/tool/find_grep.go` |
| Registro | `AddFindGrepTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.FindGrepToolDescription` (meta `[find_grep …]`, filas `path:line:content`, `[blocked …]`) |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `path` | `string` | sí | Raíz del recorrido |
| `pattern` | `string` | sí | Texto literal (default) o regex RE2 (`extended=true`) sobre el **contenido** |
| `extended` | `bool` | no | Si `true`, patrón RE2 |
| `ignoreCase` | `bool` | no | Match insensible a mayúsculas |
| `name` | `string` | no | Glob contra el basename (case-sensitive) |
| `iname` | `string` | no | Glob case-insensitive; tiene precedencia sobre `name` |
| `type` | `string` | no | `f` / `d` / `l` (solo archivos regulares se escanean en contenido) |
| `maxDepth` | `int` | no | Profundidad máxima (raíz = 0) |
| `minDepth` | `int` | no | Profundidad mínima para considerar un match |

No hay flags `show*` de columnas: esta tool no emite tabla `find`.

### Semántica

1. Walk seguro igual que `find` (denylist por nodo, sin seguir symlinks, presupuesto de nodos).
2. Predicados `name`/`iname`/`type`/`depth` seleccionan candidatos.
3. Solo entradas **regulares** se pasan a `grepScan`; directorios y no-regulares no se leen como archivo.
4. Binario: skip silencioso en el escaneo multi-archivo.
5. Private-key: se busca y se redacta el contenido de las filas match (`redacted=` en meta).

## Caps (v1)

- Presupuesto de walk: **50.000** nodos.
- Máx. **1000** filas de match (igual que `grep`).
- Contenido de línea no redactada truncado a **64 KiB**.

## Respuesta

### Éxito

```text
[find_grep path=<abs> matches=<returned>/<total> truncated=<bool> filesScanned=<n> redacted=<n> visited=<n>]
<path>:<line>:<content>
```

### Bloqueo (raíz denegada)

```text
[blocked class=... path=...]
```

## Ejemplos

```json
{ "path": "/etc", "name": "*.conf", "pattern": "TODO" }
```

```json
{ "path": "/opt/app", "type": "f", "iname": "*.env", "pattern": "api[_-]?key", "extended": true, "ignoreCase": true }
```

## Relación con otras tools

- `find` solo: metadata, sin leer contenido.
- `grep` recursivo: busca en un árbol sin predicados `name`/`type` de find.
- `list_grep`: filtra filas de un listing; no abre archivos.
