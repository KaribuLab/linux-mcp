## Why

`linux-mcp` solo expone `cat` (leer un archivo) y `list` (listar un directorio); ambas son operaciones acotadas de un solo nodo. Un agente que necesita ubicar archivos por nombre/tipo o buscar texto dentro de un árbol hoy no tiene forma segura de hacerlo dentro del servidor. Se necesitan versiones de solo lectura de `find` y `grep` que reutilicen la política de paths existente y agreguen los controles que un recorrido recursivo requiere y que `cat`/`list` no necesitaron (presupuesto de nodos, symlinks, contenido sensible por archivo).

## What Changes

- Nueva tool `find`: busca entradas por nombre/patrón (`name`/`iname`), tipo (`f`/`d`/`l`) y profundidad (`maxdepth`/`mindepth`) bajo un path. Sin predicados de acción (`-exec`, `-execdir`, `-ok`, `-delete`, `-fprint*`); solo *tests* de metadata. El agente elige qué columnas quiere de vuelta con flags booleanos por columna (`showPath`, `showType`, `showSize`, `showModTime`), todos `true` por default — ahorra tokens cuando solo necesita path o path+type.
- Nueva tool `grep`: busca un patrón en un archivo o, recursivamente, en un directorio. Modo `extended=false` (default) usa coincidencia de texto literal; `extended=true` compila el patrón como regex (motor RE2 de Go, sin backtracking catastrófico). Soporta `ignoreCase`.
- Nuevo helper compartido de recorrido recursivo (`internal/policy` walk): aplica `CheckPath` a cada nodo visitado (no solo a la raíz), nunca sigue symlinks, y corta con un presupuesto duro de 50.000 nodos visitados independiente de cuántos resultados se devuelvan.
- `grep` reutiliza el sniff de contenido existente (binario / private-key), con trato distinto por clase: un archivo **binario** se salta silenciosamente (recursivo) o bloquea (`[blocked ...]`, archivo único) y el recorrido continúa; un archivo **private-key** se busca igual que cualquier otro, pero cada línea que matchea se devuelve con el contenido redactado (`[private-key content redacted]`) y se cuenta en `redacted=<n>` del header — así el agente puede detectar claves privadas mal ubicadas sin exponer su contenido.
- Ambas tools responden con el mismo patrón `[tool key=val ...]` + body acotado ya usado por `cat`/`list`, incluyendo forma `[blocked class=... path=...]` cuando el path raíz es denegado.
- Documentación nueva: `docs/tools/find.md`, `docs/tools/grep.md`; fila nueva en `docs/README.md` (tabla de tools).

### Non-goals

- No se soporta ningún predicado/flag que ejecute código o modifique el filesystem (`-exec`, `-delete`, `-ok`, escritura de resultados a archivo).
- No se implementa POSIX BRE real; "básico" en `grep` es texto literal, no backslash-escaping de metacaracteres.
- No se sigue symlinks durante el recorrido (ni en `find` ni en `grep -r`) en esta v1.
- No hay timeout de reloj adicional al presupuesto de nodos.

## Capabilities

### New Capabilities

- `find-safe-search`: comportamiento seguro y acotado de la tool `find` (predicados soportados, caps, formato de respuesta, descripción MCP).
- `grep-safe-search`: comportamiento seguro y acotado de la tool `grep` (modo literal/regex, recursividad, caps, formato de respuesta, descripción MCP).

### Modified Capabilities

- `read-policy`: se agrega el requirement de recorrido recursivo compartido — `CheckPath` por nodo visitado, no seguir symlinks, presupuesto duro de nodos — reusable por `find`, `grep` y tools futuras que caminen árboles.

## Impact

- Código nuevo: `internal/tool/find.go`, `internal/tool/grep.go`, `internal/policy/walk.go`.
- Código modificado: `internal/handler/server.go` (registro de las dos tools nuevas).
- Reutiliza sin cambios: `internal/policy` (`CheckPath`, `ClassifyPrefix`, `CheckReadableType`), `internal/toolmeta` (header/body pattern).
- Docs: `docs/tools/find.md`, `docs/tools/grep.md` (nuevos), `docs/README.md` (tabla de tools actualizada).
- Sin dependencias externas nuevas (regex vía `regexp` de la stdlib de Go).
