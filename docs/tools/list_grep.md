# Tool: `list_grep`

Lista un directorio y filtra las **filas del listing** por un patrón (semántica `ls | grep`). **Nunca lee el contenido de archivos** — para eso están `grep` / `find_grep`. Sirve para ahorrar tokens frente a un `list` amplio seguido de filtrado en el agente.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `list_grep` |
| Código | `internal/tool/list_grep.go` |
| Registro | `AddListGrepTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.ListGrepToolDescription` (meta `[list_grep …]`, tabla markdown, `[blocked …]`) |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `path` | `string` | sí | Directorio a listar |
| `pattern` | `string` | sí | Filtro de filas (ver modos abajo). **No** busca dentro de archivos |
| `extended` | `bool` | no | Si `true`, `pattern` es regex RE2 sobre la fila completa (desactiva auto-glob) |
| `ignoreCase` | `bool` | no | Match insensible a mayúsculas |
| `all` | `bool` | no | Incluye ocultos (prefijo `.`) |
| `list` | `bool` | no | Tabla detallada (`true`) o solo nombres (`false`) |
| `showSize` … `showSymlinkPath` | `bool` | no | Igual que `list`: ocultan columnas en modo detallado (default `true`) |

### Modos de `pattern` (`extended=false`)

| Patrón | Comportamiento |
|--------|----------------|
| Contiene `*`, `?` o `[` (ej. `*.txt`) | Glob (`filepath.Match`) contra la columna **Name/File** |
| Sin metacaracteres de glob (ej. `.txt`, `readme`) | Subcadena **literal** sobre la fila completa de datos |
| `extended=true` | Siempre RE2 sobre la fila completa (ej. `\.txt$`) |

Ejemplo que los agentes suelen querer:

```json
{ "path": "/home/patricio", "pattern": "*.txt" }
```

Eso filtra **nombres** que matchean el glob, no el contenido de los `.txt`.

### Semántica del filtro

- Las líneas de cabecera markdown (`|File|` / `|Name|…|` y `|---|`) **no** se consideran filas filtrables.
- No se lee el contenido de los archivos del directorio.
- En modo literal (sin glob), el patrón puede matchear cualquier columna de la fila detallada (como `ls | grep` sobre la línea).

## Caps (v1)

- Misma ventana de listado que `list`: máx. **1000** entradas visibles antes del filtro.
- Máx. **1000** filas de salida tras el filtro.
- Misma path policy que `list`/`cat`.

## Respuesta

Un solo `TextContent`.

### Éxito

```text
[list_grep path=<abs> entries=<returned>/<matched> truncated=<bool>]
|File|
|---|
|nombre|
```

Con `list=true`, misma forma de tabla detallada que `list`, incluyendo `columns=` en el meta.

`truncated=true` cuando el listado base ya truncó (puede haber entradas no consideradas) o cuando los matches superan el cap de salida.

### Bloqueo

```text
[blocked class=... path=...]
```

## Ejemplos

```json
{ "path": "/tmp/proj", "pattern": "*.txt" }
```

```json
{ "path": "/tmp/proj", "pattern": ".txt" }
```

```json
{ "path": "/tmp/proj", "list": true, "pattern": "root", "ignoreCase": true }
```

## Relación con otras tools

- Preferir `list_grep` cuando solo se necesita un subconjunto del listing por nombre/fila.
- Para buscar **dentro** de archivos bajo un árbol con filtros de nombre/tipo, usar `find_grep`.
- Un archivo concreto: `grep` (no hace falta `cat` + filtro).
