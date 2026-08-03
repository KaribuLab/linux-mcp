## Context

Hoy `list`, `find` y `grep` son tools MCP independientes. Un agente que quiere el equivalente de `ls | grep` o `find | xargs grep` debe:

1. Llamar `list`/`find` y meter hasta 1000 filas en contexto, o
2. Llamar `grep` recursivo sin predicados de nombre/tipo (ruido + trabajo extra).

No existe composición server-side. Un DSL con `|` genérico se descartó por complejidad y riesgo de alucinación de sintaxis. `cat | grep` se descartó porque `grep path=<archivo>` ya cubre ese caso.

Fuente de verdad pública prevista: `docs/tools/list_grep.md`, `docs/tools/find_grep.md`.

## Goals / Non-Goals

**Goals:**
- Dos meta-tools nuevas: `list_grep` y `find_grep`, semántica alineada a Linux (`ls | grep`, `find | xargs grep`).
- Ahorrar tokens: el stage intermedio no llega al contexto del agente.
- Reutilizar implementación y policy de `list`/`find`/`grep` (caps, denylist, walk, sniff, redaction).
- Mantener formatos de respuesta: `list_grep` → contrato `list`; `find_grep` → contrato `grep`.
- Registro en `NewHandler` igual que el resto; sin cambios de transporte/auth/CORS.

**Non-Goals:**
- Meta-tool `pipe` / parser de expresiones con `|`.
- `cat_grep`.
- Invocar shell/`xargs` reales.
- Cambiar contratos de `list`/`find`/`grep`/`cat`.
- Pipelines de más de dos stages.

## Decisions

### D1 — Dos tools nombradas, no un DSL

Nombres MCP: `list_grep`, `find_grep`.

Alternativa: una sola `pipe` con expresión. Rechazada — más tokens de schema/docs, retries por sintaxis inválida, semántica ambigua del segundo stage.

### D2 — `list_grep` = filtro sobre filas de salida de `list`

Flujo in-process:

1. Ejecutar la misma lógica de listado que `list` (path policy, `all`, `list`, flags `show*`, cap `MaxListEntries`).
2. Compilar `pattern` con las mismas reglas que `grep` (`extended`, `ignoreCase`).
3. Filtrar **filas de datos** de la tabla (no las dos líneas de header markdown `|Col|` / `|---|`, no la línea meta `[list …]`).
4. Recalcular meta: `entries=returned/total_matched` (o equivalente claro) y `truncated` si el listado base ya truncó **o** si el filtro deja más filas que el cap de salida (mismo tope 1000).
5. Emitir formato idéntico a `list` (meta `[list …]` o meta propia `[list_grep …]` — ver D2b).

Modos de patrón (`extended=false`):

- Si `pattern` contiene `*`, `?` o `[` → glob (`filepath.Match`) contra la columna **Name/File** (UX de agente: `*.txt` filtra nombres, no contenido).
- Si no → subcadena literal sobre la **línea de fila completa** (como `ls | grep`; con `list=true` puede matchear owner/mode).
- `extended=true` → siempre RE2 sobre la fila completa (sin auto-glob).

No abre archivos; no usa walk. (Agentes confunden “grep” en el nombre con búsqueda de contenido — la descripción MCP debe decir explícitamente NEVER reads file contents.)


### D2b — Header meta de `list_grep`

Usar `[list_grep path=... entries=... truncated=...]` (y `columns=` cuando aplique), no reutilizar el tag `[list …]`, para que el agente distinga la tool. El body (tabla markdown) permanece idéntico en forma a `list`.

Alternativa: reutilizar `[list …]`. Rechazada — dificulta auditoría y tests e2e de registro.

### D3 — `find_grep` = find predicates → grep de contenido (xargs)

Flujo in-process:

1. Walk con predicados de `find` (`path`, `name`, `iname`, `type`, `maxDepth`, `minDepth`) — misma policy: denylist por nodo, no seguir symlinks, presupuesto de nodos.
2. Para cada match que sea archivo regular (o symlink a archivo según el mismo criterio que `grep` single-file / `CheckReadableType`), ejecutar `grepScan` con `pattern`/`extended`/`ignoreCase`.
3. Directorios matcheados por find **no** se grepean como archivo; solo aportan si el walk ya los usó para descender. Igual espíritu que `find … -type f | xargs grep` cuando el caller filtra `type=f`; si `type` se omite, grepear solo entradas legibles como archivo (saltar dirs), no re-walk contenido bajo un dir match.
4. Salida: formato **grep** — meta `[find_grep …]` + filas `path:line:content`. Caps: `MaxGrepMatches`, truncado de línea 64 KiB, `redacted=`, skip binario, redact private-key — igual que `grep`.
5. `truncated=true` si se corta por match cap, node budget del find, o ambos.

Args de columnas `show*` de `find` **no** aplican: la salida no es tabla find.

### D4 — Extraer helpers internos sin cambiar API pública de list/find/grep

Preferir refactor mínimo:

- Compartir compilación de patrón (`compileGrepPattern`) y `grepScan` (ya en `grep.go`).
- Para `list_grep`: o bien factorizar “construir filas de list” a un helper interno, o listar + filtrar en el mismo paquete `tool` reutilizando funciones no exportadas tras un split pequeño.
- No exponer los helpers como tools MCP nuevas.

Alternativa: llamar a los handlers MCP unos desde otros. Rechazada — acopla a `CallToolRequest` y complica tests.

### D5 — Registro y docs

```go
tool.AddListGrepTool(server)
tool.AddFindGrepTool(server)
```

en `internal/handler/server.go` junto a las existentes.

Docs: `docs/tools/list_grep.md`, `docs/tools/find_grep.md` + filas en `docs/README.md`. Descripciones MCP en constantes `ListGrepToolDescription` / `FindGrepToolDescription` (contrato agent-facing).

### D6 — Firmas (resumen)

**`list_grep`**

| Arg | Tipo | Notas |
|-----|------|-------|
| `path` | string | requerido |
| `pattern` | string | requerido |
| `extended` | bool | default false (literal) |
| `ignoreCase` | bool | opcional |
| `all`, `list`, `show*` | igual que `list` | mismos defaults |

**`find_grep`**

| Arg | Tipo | Notas |
|-----|------|-------|
| `path` | string | requerido |
| `pattern` | string | requerido |
| `extended`, `ignoreCase` | bool | igual que `grep` |
| `name`, `iname`, `type`, `maxDepth`, `minDepth` | igual que `find` | sin `show*` |

## Risks / Trade-offs

- **[Risk] Filtrar fila markdown completa matchea columnas no-Name** → Mitigación: documentar semántica `ls | grep` (línea completa); agent usa `list=false` o patrón anclado si solo quiere nombres.
- **[Risk] `find_grep` sin `type=f` grepea symlinks/raros** → Mitigación: solo entradas que pasen `CheckReadableType` / mismo gate que `grepScan`; dirs skip.
- **[Risk] Dos tools más en el schema MCP = más tokens de tool list** → Mitigación: aceptable vs devolver 1000 filas; descripciones concisas.
- **[Risk] Refactor de list para reutilizar filas introduce regresiones** → Mitigación: tests existentes de `list` deben seguir verdes; preferir extracción interna mínima.
- **[Trade-off] Header `[list_grep]`/`[find_grep]` vs reusar tags** → claridad > compat parsers que asuman solo cuatro tags.

## Migration Plan

- Deploy: binario nuevo registra dos tools; clients antiguos las ignoran.
- Rollback: quitar registro; sin migración de datos.
- Sin **BREAKING** para callers de tools existentes.

## Open Questions

- Ninguna bloqueante. Detalle menor de implementación: si el meta de `list_grep` reporta `entries` como matches tras filtro vs entries del list pre-filtro — preferencia: `entries=<filas_devueltas>/<total_que_matchearon_el_patrón>` y `truncated` si el list base truncó (puede haber matches no vistos) o si se cortó por cap post-filtro.
