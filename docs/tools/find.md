# Tool: `find`

Busca entradas en un árbol de directorios por metadata (nombre/patrón, tipo, profundidad), con recorrido seguro acotado. Nunca ejecuta comandos ni modifica el filesystem.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `find` |
| Código | `internal/tool/find.go` |
| Registro | `AddFindFilesTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.FindToolDescription` (meta `[find …]`, tabla markdown de columnas variables, `[blocked …]`) |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `path` | `string` | sí | Directorio raíz a recorrer |
| `name` | `string` | no | Glob contra el nombre base, sensible a mayúsculas (ej. `*.go`) |
| `iname` | `string` | no | Glob contra el nombre base, insensible a mayúsculas; tiene precedencia sobre `name` si ambos vienen |
| `type` | `string` | no | `f` (archivo regular), `d` (directorio), `l` (symlink) |
| `maxDepth` | `int` | no | Profundidad máxima a descender (raíz = 0). `0`/omitido = sin límite |
| `minDepth` | `int` | no | Profundidad mínima para reportar un match (raíz = 0). `0`/omitido = incluye la raíz |
| `showPath` | `bool` | no | Incluir columna `Path`. Default `true` |
| `showType` | `bool` | no | Incluir columna `Type`. Default `true` |
| `showSize` | `bool` | no | Incluir columna `Size`. Default `true` |
| `showModTime` | `bool` | no | Incluir columna `ModTime`. Default `true` |

No existe ningún predicado de acción (`-exec`, `-execdir`, `-ok`, `-okdir`, `-delete`, `-fprint`/`-fprintf`) — solo *tests* de metadata.

### Selección de columnas

Si las cuatro flags `show*` vienen `false`, la tool no devuelve una tabla vacía: cae a la columna `Path` sola. El orden de columnas en la tabla es siempre `Path`, `Type`, `Size`, `ModTime` (solo las presentes), independiente del orden en que se pasaron las flags. Útil para ahorrar tokens de salida cuando solo interesa, por ejemplo, la ruta.

## Caps (v1)

- Recorrido acotado por un presupuesto duro de **50.000 nodos visitados** (`internal/policy.Walk`), independiente de cuántos matches se encuentren.
- Máx. **1000** filas devueltas en la tabla.
- Misma path policy que `cat`/`list` (`internal/policy.CheckPath`), aplicada a **cada nodo visitado**, no solo a la raíz.
- Nunca sigue symlinks al descender; un symlink puede reportarse como entrada (sujeto al filtro `type`) pero su destino no se recorre.

## Respuesta

Un solo `TextContent`.

### Éxito

```text
[find path=<abs> matches=<returned>/<total> truncated=<bool> visited=<n>]
|Path|Type|Size|ModTime|
|---|---|---|---|
|...|
```

Con columnas restringidas (ej. solo `showPath=true`):

```text
[find path=<abs> matches=<returned>/<total> truncated=<bool> visited=<n>]
|Path|
|---|
|...|
```

- `truncated=true` cuando se corta por el cap de filas (1000) o por el presupuesto de nodos del walk (50.000), lo que ocurra primero.
- `visited` es el total de nodos que el walk efectivamente recorrió (útil para distinguir "no hay más resultados" de "se cortó el recorrido").

### Bloqueo (IsError)

```text
[blocked class=path_denied path=<abs>]
```

Solo aplica al path raíz. Un nodo denegado **dentro** del árbol se saltea silenciosamente y el recorrido continúa (no aparece en resultados, no genera `[blocked ...]`).

## Comportamiento

1. `policy.CheckPath` sobre la raíz (mismo denylist que `cat`/`list`); si es denegada, responde `[blocked ...]` sin recorrer nada.
2. `policy.Walk` recorre el árbol: aplica `CheckPath` a cada nodo, nunca sigue symlinks, poda por `minDepth`/`maxDepth` durante el descenso (no post-filtro), corta a los 50.000 nodos.
3. Cada nodo visitado dentro del rango de profundidad se evalúa contra `name`/`iname` (glob sobre el nombre base) y `type`.
4. Los matches se acotan a 1000 filas; la tabla se arma solo con las columnas pedidas.

## Ejemplos

**Buscar archivos Go**

```json
{ "path": "/home/user/proyecto", "name": "*.go" }
```

**Solo directorios, hasta 2 niveles**

```json
{ "path": "/var/log", "type": "d", "maxDepth": 2 }
```

**Solo la ruta (ahorra tokens en árboles grandes)**

```json
{ "path": "/home/user/proyecto", "showType": false, "showSize": false, "showModTime": false }
```

## Notas / límites

- `find`/`grep` comparten el mismo helper de recorrido seguro (`internal/policy.Walk`).
- No hay timeout de reloj adicional al presupuesto de nodos.
- Relacionada: `internal/policy`, `internal/toolmeta`, [`docs/tools/grep.md`](grep.md).
