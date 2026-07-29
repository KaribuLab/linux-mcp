# Tool: `grep`

Busca un patrón en un archivo o, recursivamente, en un directorio, con recorrido seguro acotado. Solo lectura: no ejecuta comandos ni modifica el filesystem.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `grep` |
| Código | `internal/tool/grep.go` |
| Registro | `AddGrepTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.GrepToolDescription` (meta `[grep …]`, filas de texto crudo, `[blocked …]`) |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `path` | `string` | sí | Archivo o directorio a buscar |
| `pattern` | `string` | sí | Texto literal (default) o regex RE2 (`extended=true`) |
| `extended` | `bool` | no | Si `true`, compila `pattern` como regex RE2 en vez de texto literal |
| `ignoreCase` | `bool` | no | Búsqueda insensible a mayúsculas |
| `maxDepth` | `int` | no | Profundidad máxima de recursión cuando `path` es un directorio. `0`/omitido = sin límite |

## Modos de patrón

- **Literal (default, `extended=false`)**: `pattern` se busca como texto plano (`regexp.QuoteMeta` + `regexp.Compile`, equivalente a `grep -F`). Metacaracteres como `.` o `*` no tienen significado especial.
- **Extendido (`extended=true`)**: `pattern` se compila tal cual con el paquete `regexp` de Go — motor **RE2**, tiempo lineal garantizado, sin backtracking catastrófico. No es PCRE; no se soporta `-P`.
- `ignoreCase=true` agrega `(?i)` en ambos modos.

## Recursividad

- `path` puede ser un archivo o un directorio.
- Si es directorio, recorre recursivamente con el mismo helper que `find` (`internal/policy.Walk`): `CheckPath` por nodo, nunca sigue symlinks, presupuesto duro de 50.000 nodos.

## Contenido sensible durante la búsqueda

El sniff de contenido (`policy.ClassifyPrefix`, el mismo que usa `cat`) se aplica por archivo visitado, con trato **distinto por clase**:

| Clase | Archivo único | Recursivo |
|-------|----------------|-----------|
| **Binario** (NUL byte) | `[blocked class=binary path=...]`, no busca contenido | Se saltea silenciosamente, sin contador en el header, el recorrido continúa |
| **Private-key** (header PEM/OpenSSH/PuTTY) | **No se bloquea** — se busca el patrón igual que cualquier archivo | **No se saltea** — se busca igual que cualquier archivo |

Para archivos private-key, cada línea que matchea el patrón se devuelve con el contenido reemplazado por el placeholder fijo `[private-key content redacted]` en vez del texto real, y se cuenta en `redacted=<n>` del header. Esto permite detectar claves privadas mal ubicadas (ej. buscando `BEGIN.*PRIVATE KEY` en un árbol) sin que la tool exponga el contenido de la clave.

## Caps (v1)

- Máx. **1000** filas devueltas.
- Cada fila de contenido (no redactada) se trunca a **64 KiB** (mismo cap que `cat`), sin abortar la fila completa.
- Recorrido recursivo acotado por el presupuesto de 50.000 nodos del walk compartido.

## Respuesta

Un solo `TextContent`. El body **no** es tabla markdown (el contenido de línea puede tener `|`).

### Éxito

```text
[grep path=<abs> matches=<returned>/<total> truncated=<bool> filesScanned=<n> redacted=<n>]
<path>:<line>:<content>
```

- `redacted` cuenta las filas cuyo `<content>` fue reemplazado por `[private-key content redacted]`. Los archivos binarios salteados no cuentan en ningún contador del header (siguen siendo completamente silenciosos).
- `truncated=true` cuando se corta por el cap de filas, por el cap de bytes de una línea, o por el presupuesto de nodos del walk (directorio).

### Bloqueo (IsError)

```text
[blocked class=<class> path=<abs>]
```

Aplica al path raíz denegado por policy, o a un archivo único clasificado como binario. Un archivo único private-key **ya no** usa esta forma — ver arriba.

## Comportamiento

1. `policy.CheckPath` sobre `path`; si es denegado, responde `[blocked ...]` sin buscar nada.
2. Si `path` es directorio: `policy.Walk` visita cada archivo regular (saltea directorios y symlinks), aplicando el sniff por archivo.
3. Si `path` es archivo: sniff directo; binario bloquea, private-key busca y redacta, cualquier otro caso busca normal.
4. El patrón se compila una vez (`compileGrepPattern`) y se aplica línea por línea (`bufio.Reader.ReadBytes('\n')`, soporta líneas arbitrariamente largas sin abortar por límite de buffer).

## Ejemplos

**Búsqueda literal en un archivo**

```json
{ "path": "/var/log/syslog", "pattern": "ERROR" }
```

**Regex extendido, insensible a mayúsculas, en un árbol**

```json
{ "path": "/home/user/proyecto", "pattern": "TODO|FIXME", "extended": true, "ignoreCase": true }
```

**Auditar claves privadas mal ubicadas**

```json
{ "path": "/home", "pattern": "BEGIN.*PRIVATE KEY", "extended": true }
```

Devuelve `path:line:[private-key content redacted]` por cada hit, con `redacted=<n>` en el header — suficiente para localizar el archivo sin exponer su contenido.

## Notas / límites

- No es POSIX BRE real; "literal" es texto plano, no backslash-escaping de metacaracteres.
- `find`/`grep` comparten el mismo helper de recorrido seguro (`internal/policy.Walk`).
- Relacionada: `internal/policy`, `internal/toolmeta`, [`docs/tools/find.md`](find.md), [`docs/tools/cat.md`](cat.md).
