## Why

Un agente que hace `list` o `find` amplio y luego filtra en contexto gasta tokens en filas inútiles y en round-trips extra. En Linux eso se resuelve con composición (`ls | grep`, `find | xargs grep`). Hoy cada tool MCP es una llamada aislada; hace falta composición **server-side** que devuelva solo el resultado útil, sin exponer un shell ni un DSL de pipes genérico.

## What Changes

- Nueva tool `list_grep`: semántica `ls | grep` — lista un directorio y filtra las **filas de salida** por patrón (literal o RE2). No abre el contenido de archivos. Respuesta en el **mismo formato que `list`** (meta + tabla markdown), solo con las filas que matchean.
- Nueva tool `find_grep`: semántica `find | xargs grep -E` — aplica predicados de `find` (name/iname/type/depth) y busca el patrón en el **contenido** de los paths hallados. Respuesta en el **mismo formato que `grep`** (`path:line:content` + meta), con la misma política de binario/private-key/caps.
- Reutiliza motores existentes de `list`, `find` y `grep` (policy walk, sniff, toolmeta); no cambia el contrato de las tools actuales.
- Documentación nueva: `docs/tools/list_grep.md`, `docs/tools/find_grep.md`; filas nuevas en `docs/README.md`.

### Non-goals

- No hay meta-tool `pipe` ni DSL con `|` arbitrario.
- No se añade `cat_grep` (un `grep` sobre un archivo ya cubre ese caso).
- No se invoca `/bin/sh` ni `xargs` real; la composición es in-process.
- No se cambia el formato default de `list`, `find`, `grep` ni `cat`.
- No se añaden stages extra (`| head`, `| sort`, etc.) en esta v1.

## Capabilities

### New Capabilities

- `list-grep-compose`: comportamiento de la tool `list_grep` (args de list + patrón de filtro sobre filas, formato de respuesta, caps, descripción MCP).
- `find-grep-compose`: comportamiento de la tool `find_grep` (predicados find + búsqueda de contenido grep, formato de respuesta, caps, descripción MCP).

### Modified Capabilities

- (ninguno — `list`, `find` y `grep` conservan sus requisitos; las nuevas tools los componen sin alterar sus contratos).

## Impact

- Código nuevo: `internal/tool/list_grep.go`, `internal/tool/find_grep.go` (nombres finales a fijar en design).
- Código modificado: `internal/handler/server.go` (registro de las dos tools).
- Reutiliza: `internal/tool/list.go`, `find.go`, `grep.go`, `internal/policy` (walk, CheckPath, sniff), `internal/toolmeta`.
- Docs: `docs/tools/list_grep.md`, `docs/tools/find_grep.md` (nuevos), `docs/README.md` (tabla de tools).
- Sin dependencias externas nuevas.
- Breaking: ninguno (solo tools nuevas).
