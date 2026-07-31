## 1. Preparación / helpers compartidos

- [x] 1.1 Extraer o reutilizar `compileGrepPattern` / lógica de filtrado de filas de texto en el paquete `tool` para que `list_grep` y `find_grep` no dupliquen reglas literal/RE2/`ignoreCase`.
- [x] 1.2 Decidir extracción mínima de “construir filas de list” (helper interno) vs implementar filtrado junto a la lógica existente en `list.go`, sin cambiar el contrato público de `list`.
- [x] 1.3 Añadir `toolmeta.ListGrepHeader` y `toolmeta.FindGrepHeader` (tags `[list_grep …]` / `[find_grep …]`) siguiendo el patrón de headers existentes.

## 2. Tool `list_grep`

- [x] 2.1 Definir `ListGrepArgs` (args de `list` + `pattern` + `extended` + `ignoreCase`) y `ListGrepToolDescription` en `internal/tool/list_grep.go`.
- [x] 2.2 Implementar `ListGrep`: listar con misma policy/caps/flags que `list`, filtrar solo filas de datos por patrón (línea completa), emitir meta `[list_grep …]` + tabla markdown.
- [x] 2.3 Señalar `truncated=true` si el listing base truncó o si los matches superan el cap de salida.
- [x] 2.4 `AddListGrepTool(server)` registrando la tool `list_grep`.
- [x] 2.5 Tests: filtro simple por nombre; filtro sobre fila detallada; literal vs extended; path denegado; truncamiento del listing base; headers markdown no cuentan como matches.
- [x] 2.6 Documentar en `docs/tools/list_grep.md` y añadir fila en `docs/README.md` (y tabla de tools en `README.md` raíz).

## 3. Tool `find_grep`

- [x] 3.1 Definir `FindGrepArgs` (predicados de `find` sin `show*` + `pattern`/`extended`/`ignoreCase`) y `FindGrepToolDescription` en `internal/tool/find_grep.go`.
- [x] 3.2 Implementar `FindGrep`: walk con predicados find → `grepScan` solo en entradas legibles como archivo; salida formato grep con meta `[find_grep …]`.
- [x] 3.3 Reutilizar skip binario / redaction private-key / caps de matches y de línea de `grep`; `truncated` por node budget o match cap.
- [x] 3.4 `AddFindGrepTool(server)` registrando la tool `find_grep`.
- [x] 3.5 Tests: name+pattern de contenido; archivo fuera de predicados no aparece; dir match no se escanea como archivo; private-key redactado; path denegado; truncamiento por cap/budget.
- [x] 3.6 Documentar en `docs/tools/find_grep.md` y añadir fila en `docs/README.md` (y tabla de tools en `README.md` raíz).

## 4. Registro e integración

- [x] 4.1 Registrar `AddListGrepTool` y `AddFindGrepTool` en `internal/handler/server.go`.
- [x] 4.2 Test e2e/registro que confirme que `list_grep` y `find_grep` aparecen en la lista de tools MCP.
- [x] 4.3 Verificar que tests existentes de `list`/`find`/`grep` siguen pasando tras cualquier refactor de helpers.

## 5. Cierre

- [x] 5.1 Ejecutar `go test ./...` y linters del proyecto.
- [x] 5.2 Ejecutar `graphify --update .` si hubo cambios de código relevantes.
- [x] 5.3 Guardar en Engram (`topic_key: project:linux-mcp:current-state` y `openspec:tool-pipes`) el estado implementado.
