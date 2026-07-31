## 0. Estándar de documentación (prompts de ejemplo)

- [x] 0.1 Actualizar `openspec/config.yaml` (contexto + rules) con el estándar `## Prompt de ejemplo (agente)` en docs de tools.
- [x] 0.2 Actualizar `docs/README.md` sección Convención: documentar el estándar de prompts de ejemplo por tool y listar las cuatro tools nuevas en la tabla cuando existan sus docs.

## 1. Helpers compartidos

- [x] 1.1 Añadir headers en `internal/toolmeta`: `PsHeader`, `PsGrepHeader`, `SsHeader`, `SsGrepHeader` (tags `[ps …]` / `[ps_grep …]` / `[ss …]` / `[ss_grep …]`, con `columns=` cuando aplique).
- [x] 1.2 Reutilizar lógica de patrón/filtro de filas de `list_grep` (literal / auto-glob / RE2 / `ignoreCase`) para `ps_grep` y `ss_grep`, sin DSL pipe.
- [x] 1.3 Helper interno compartido de resolución de flags `show*` (default true, orden fijo, columna(s) identidad siempre on) reutilizable por `ps` y `ss`.
- [x] 1.4 Garantizar en código/tests de estas tools que **no** se usa `os/exec` ni spawn de binarios del host.

## 2. Tool `ps` + `ps_grep`

- [x] 2.1 Implementar `ps` (`internal/tool/ps.go`): solo lectura `/proc`, `includeKernel`, columns `show*`, cap 1000, meta + tabla, `PsToolDescription`.
- [x] 2.2 Tests unitarios de `ps` (default sin kernel threads, `includeKernel`, hide columns, truncamiento).
- [x] 2.3 Implementar `ps_grep` (`internal/tool/ps_grep.go`): mismos args de selección/columnas + `pattern`/`extended`/`ignoreCase`, meta `[ps_grep …]`.
- [x] 2.4 Tests de `ps_grep` (filtro por Comm/cmdline, truncación base, show flags).
- [x] 2.5 Documentar `docs/tools/ps.md` y `docs/tools/ps_grep.md` con sección `## Prompt de ejemplo (agente)` cada una (prompts en español neutro, forma `Usa el tool linux-mcp \`ps\` para …` / `ps_grep`).
- [x] 2.6 Añadir filas en `docs/README.md` y tabla de tools del `README.md` raíz.

## 3. Tool `ss` + `ss_grep`

- [x] 3.1 Añadir dep `github.com/florianl/go-diag` (pin en `go.mod`); implementar backend **solo netlink vía esa lib** (sin binario `ss`); tool `ss` con defaults `state=LISTEN` / `family=inet`, ampliación a `ESTAB`/`all`, `show*`, cap 1000. Si la lib no cubre un caso v1, documentar y valorar respaldo D5b en la misma PR.
- [x] 3.2 Asegurar `govulncheck` (o equivalente del CI del repo) sobre el módulo tras añadir la dep.
- [x] 3.3 Tests de `ss` (defaults, widen state, hide Peer, truncamiento; fixtures donde sea posible).
- [x] 3.4 Implementar `ss_grep` + tests (p. ej. patrón `0.0.0.0` / puerto).
- [x] 3.5 Documentar `docs/tools/ss.md` y `docs/tools/ss_grep.md` con `## Prompt de ejemplo (agente)` cada una (indicar fuente netlink, no CLI).
- [x] 3.6 Actualizar `docs/README.md` y `README.md` raíz.

## 4. Registro e integración

- [x] 4.1 Registrar las cuatro tools en `internal/handler/server.go`.
- [x] 4.2 Test e2e/registro: las cuatro aparecen en la lista MCP.
- [x] 4.3 Verificar que tests existentes de tools FS siguen pasando.

## 5. Cierre

- [x] 5.1 `go test ./...` y linters del proyecto.
- [x] 5.2 `graphify --update .` si hubo cambios de código relevantes.
- [x] 5.3 Guardar en Engram (`topic_key: project:linux-mcp:current-state` y `openspec:add-ps-services-ss`) el estado (sin services en v1; cero exec de binarios).
