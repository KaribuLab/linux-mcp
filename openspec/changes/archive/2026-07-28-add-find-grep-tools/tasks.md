## 1. Walk compartido (`internal/policy`)

- [x] 1.1 Crear `internal/policy/walk.go` con `WalkLimits` (default `MaxNodes = 50_000`), `WalkFunc` y `Walk(root, limits, minDepth, maxDepth, fn)`.
- [x] 1.2 Aplicar `CheckPath` a cada nodo visitado (no solo la raíz); si un nodo es denegado, saltearlo y continuar el walk.
- [x] 1.3 No seguir symlinks al descender (detectar `os.ModeSymlink` y no recursar en el target); el propio symlink puede reportarse como entrada según filtros de tipo del caller.
- [x] 1.4 Cortar el walk al alcanzar `MaxNodes` visitados y devolver una señal de truncamiento independiente del conteo de resultados del caller.
- [x] 1.5 Aplicar `minDepth`/`maxDepth` durante el walk (podar antes de descender cuando corresponda), no como post-filtro.
- [x] 1.6 Tests unitarios: nodo denegado se saltea y el walk continúa; symlink a directorio no se desciende; walk se corta exactamente en `MaxNodes`; `minDepth`/`maxDepth` podan correctamente.

## 2. Tool `find`

- [x] 2.1 Definir `FindFilesArgs` (`path`, `name`/`iname`, `type` `f|d|l`, `maxDepth`, `minDepth`, `showPath`/`showType`/`showSize`/`showModTime` bool con default `true`) en `internal/tool/find.go`.
- [x] 2.2 Implementar `FindFiles`: `CheckPath` sobre la raíz (mismo patrón `[blocked ...]` que `cat`/`list`), luego usar el walk compartido (tarea 1) para recolectar matches.
- [x] 2.3 Formatear resultados como tabla markdown con columnas variables según `show*` (orden fijo `Path|Type|Size|ModTime`, solo las columnas con flag `true`; si las cuatro vienen `false`, forzar columna `Path` sola), con cap de filas devueltas y `truncated` cuando se corte por cap de resultados o por presupuesto de nodos del walk.
- [x] 2.4 Definir `toolmeta.FindHeader` (`[find path=... matches=returned/total truncated=bool visited=...]`) en `internal/toolmeta` siguiendo el patrón de `CatHeader`/`ListHeader`.
- [x] 2.5 Escribir `FindToolDescription` explicando: predicados soportados, ausencia de acciones (`-exec`/`-delete`/etc.), flags `show*` de selección de columnas (default todos `true`), formato `[find ...]` + tabla, truncamiento, y `[blocked ...]`.
- [x] 2.6 `AddFindFilesTool(server)` registrando la tool `find`.
- [x] 2.7 Tests: predicados `name`/`iname`/`type`/`maxDepth`/`minDepth`; path raíz denegado; nodo denegado dentro del árbol excluido; truncamiento por cap de resultados y por presupuesto de nodos; columnas por default (las cuatro); subconjunto de columnas (ej. solo `showPath`+`showType`); las cuatro `show*` en `false` cae a columna `Path` sola.

## 3. Tool `grep`

- [x] 3.1 Definir `GrepArgs` (`path`, `pattern`, `extended bool`, `ignoreCase bool`, `maxDepth`) en `internal/tool/grep.go`.
- [x] 3.2 Implementar compilación de patrón: `extended=false` → `regexp.QuoteMeta` + `regexp.Compile` (match literal); `extended=true` → `regexp.Compile` directo sobre el patrón (RE2). Aplicar `ignoreCase` con el flag `(?i)` o equivalente en ambos modos.
- [x] 3.3 Caso archivo único: abrir, aplicar sniff (`policy.ClassifyPrefix`) igual que `cat`. Si clasifica **binario**, responder `[blocked class=binary ...]` sin buscar contenido. Si clasifica **private-key**, NO bloquear — buscar el patrón igual que un archivo normal (ver 3.4b).
- [x] 3.4 Caso directorio: usar el walk compartido (tarea 1); por cada archivo visitado, aplicar el mismo sniff. Si es **binario**, **saltear silenciosamente** (sin contador en el header), continuando el walk. Si es **private-key**, no saltear — buscar el patrón igual que 3.4b.
- [x] 3.4b Tratamiento private-key (archivo único o dentro del walk): correr la búsqueda del patrón normalmente sobre el contenido; por cada línea que matchea, emitir la fila `<path>:<line>:<content>` con `<content>` reemplazado por el placeholder fijo `[private-key content redacted]`; incrementar el contador `redacted`.
- [x] 3.5 Recolectar matches como filas `<path>:<line>:<content>` (o el placeholder redactado de 3.4b), truncando cada fila no redactada al cap de bytes por línea (reusar `policy.MaxBytes`) sin abortar la fila completa.
- [x] 3.6 Cap de filas totales devueltas; `truncated=true` si se corta por ese cap o por el presupuesto de nodos del walk (directorio) o por el cap de bytes/líneas (archivo único).
- [x] 3.7 Definir `toolmeta.GrepHeader` (`[grep path=... matches=returned/total truncated=bool filesScanned=... redacted=...]`) en `internal/toolmeta`.
- [x] 3.8 Escribir `GrepToolDescription` explicando: modo literal vs `extended=true` (RE2, no PCRE), archivo único vs directorio recursivo, salteo silencioso de binario, búsqueda-y-redacción de contenido private-key (con contador `redacted`), formato `[grep ...]` + filas, truncamiento, y `[blocked ...]`.
- [x] 3.9 `AddGrepTool(server)` registrando la tool `grep`.
- [x] 3.10 Tests: modo literal vs extendido; `ignoreCase`; archivo único binario bloqueado por sniff; archivo único private-key buscado y redactado (no bloqueado); directorio con archivo binario salteado sin abortar; directorio con archivo private-key buscado y filas redactadas, contador `redacted` correcto; truncamiento por cap de filas, por línea larga, y por presupuesto de nodos.

## 4. Registro e integración

- [x] 4.1 Registrar `tool.AddFindFilesTool(server)` y `tool.AddGrepTool(server)` en `internal/handler/server.go` junto a `cat`/`list`.
- [x] 4.2 Test de integración (`internal/e2e`) que liste las tools del servidor MCP y confirme que `find` y `grep` aparecen con sus descripciones.

## 5. Documentación

- [x] 5.1 Crear `docs/tools/find.md` (parámetros incluyendo flags `show*` de columnas, caps, walk seguro, formato de respuesta, ejemplos) siguiendo el formato de `docs/tools/list.md`.
- [x] 5.2 Crear `docs/tools/grep.md` (parámetros, modos literal/extendido, recursividad, salteo silencioso de contenido sensible, caps, formato de respuesta, ejemplos) siguiendo el formato de `docs/tools/cat.md`.
- [x] 5.3 Agregar filas de `find` y `grep` a la tabla de tools en `docs/README.md` y en la sección de tools de `README.md` (raíz).
- [x] 5.4 Actualizar `internal/policy` doc/comentarios si aplica para reflejar el nuevo helper `Walk` como parte de la política compartida.

## 6. Cierre

- [x] 6.1 Ejecutar `go test ./...` y linters del proyecto.
- [x] 6.2 Ejecutar `graphify --update .` para reflejar los nuevos archivos/tools en el grafo.
- [x] 6.3 Guardar en Engram (`topic_key: project:linux-mcp:current-state`) que `find`/`grep` fueron implementadas, con decisiones clave (walk compartido, cap de nodos, modos de grep).
