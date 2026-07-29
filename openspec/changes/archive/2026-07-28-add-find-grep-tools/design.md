## Context

`internal/tool` hoy tiene dos tools, ambas O(1) acotadas por nodo:

- `cat` (`internal/tool/cat.go`): lee un archivo, cap dual (100 líneas ∩ 64 KiB), resume por `Seek` de byte.
- `list` (`internal/tool/list.go`): lista un directorio, cap de 1000 entradas.

Ambas comparten `internal/policy` (`CheckPath` denylist, `CheckReadableType`, `ClassifyPrefix` sniff de binario/private-key) e `internal/toolmeta` (header `[tool key=val]` + body, forma `[blocked class=... path=...]`).

`find` y `grep -r` son recorridos recursivos de árbol: la superficie de abuso no es "un archivo grande" sino "un árbol grande" (ej. apuntar a `/`). `CheckPath` alcanza para el nodo raíz pero no protege nodos internos del walk, y no existe hoy ningún presupuesto de trabajo (nodos visitados) — solo caps de *salida*.

## Goals / Non-Goals

**Goals:**
- Exponer `find` y `grep` de solo lectura, sin ningún predicado/flag que ejecute código o mute el filesystem.
- Compartir una sola implementación de recorrido seguro (`internal/policy` walk) entre `find`, `grep -r` y tools futuras que caminen árboles.
- Mantener el mismo contrato de respuesta (header de una línea + body acotado + `[blocked ...]`) que `cat`/`list`.
- Acotar el costo de un walk por presupuesto de nodos visitados, no solo por resultados devueltos.

**Non-Goals:**
- Implementar predicados de acción de `find` (`-exec`, `-execdir`, `-ok`, `-delete`, `-fprint*`) o cualquier flag de `grep` que invoque un proceso externo.
- Implementar POSIX BRE real (backslash-escaping de metacaracteres); "básico" en `grep` es coincidencia de texto literal.
- Seguir symlinks durante el recorrido (evita loops y fuga fuera del subárbol pedido).
- Timeout de reloj: el presupuesto de nodos es el único corte de recorrido en v1.
- Paginación de resultados entre llamadas (igual que `list` v1: solo se señala `truncated`, no hay cursor de reanudación de walk).

## Decisions

### D1 — `find`: solo *tests* de metadata, nunca *actions*

Args: `path` (string, requerido), `name`/`iname` (glob, opcional, mutuamente con preferencia a `iname` si ambos vienen), `type` (`f`|`d`|`l`, opcional), `maxDepth`/`minDepth` (int, opcional).

Alternativa considerada: exponer `-printf`-like format string. Rechazada — mismo motivo que `list` fija sus columnas: una superficie de formato libre es más código a validar por cero beneficio (el agente ya recibe todas las columnas útiles en la tabla fija). En su lugar, selección de columnas vía flags booleanos fijos (D1b) — mismo espíritu de "sin formato libre" pero permite ahorrar tokens.

### D1b — `find`: selección de columnas de salida vía flags booleanos

Args adicionales: `showPath`, `showType`, `showSize`, `showModTime` (bool, opcionales, default `true` los cuatro). El agente decide qué columnas necesita de vuelta — ej. solo `showPath=true` y el resto `false` cuando únicamente le importa la ruta, ahorrando tokens de salida en árboles grandes.

Reglas:
- Si las cuatro flags vienen `false`, la tool responde igual (header + tabla) pero con una única columna `Path` forzada — nunca se devuelve una tabla sin ninguna columna, sería una respuesta inútil.
- El orden de columnas en la tabla es fijo (`Path`, `Type`, `Size`, `ModTime`) independiente del orden en que el caller puso las flags en `true`; solo cambia cuáles aparecen.
- No hay flag para agregar columnas nuevas (ej. permisos, owner) — mismo alcance de metadata que D1, esto es únicamente proyección de las columnas ya definidas.

Alternativa considerada: un array `columns: []string` con nombres de columna. Rechazada — flags booleanos individuales son más simples de describir en el schema MCP (JSON Schema `boolean` con default, sin necesidad de validar un enum de strings) y más fácil de razonar para el agente ("quiero esto sí/no" en vez de "arma un array con los nombres exactos").

### D2 — `grep`: "básico" = texto literal, "extendido" = regex RE2

`extended=false` (default) compila el patrón como texto literal (`regexp.QuoteMeta` + `regexp.Compile`, equivalente a `grep -F`). `extended=true` compila el patrón tal cual con el paquete `regexp` de Go (motor RE2).

Alternativa considerada: traducir POSIX BRE real (backslash-escapa metacaracteres) para "básico". Rechazada — con RE2 de por medio (tiempo lineal garantizado, sin backtracking) la distinción básica/extendida de grep tradicional existe por motivos de sintaxis histórica y de riesgo de ReDoS en motores con backtracking (PCRE); ninguno de los dos aplica acá. Texto literal como modo básico es más predecible para un agente y evita que caracteres de glob comunes en nombres de archivo/paths se interpreten como regex por accidente.

Alternativa considerada: exponer `-P` (PCRE). Rechazada explícitamente por el usuario — no se agrega ningún motor de backtracking.

### D3 — `grep` soporta archivo único y directorio (recursivo) en la misma tool

Un solo arg `path` que puede ser archivo o directorio; si es directorio, recorre recursivamente vía el walk compartido (D5). Evita tener `grep` y `grepRecursive` como tools separadas para un caso de uso que en la práctica siempre se pide junto.

### D4 — Contenido sensible durante `grep`: binario se saltea/bloquea, private-key se busca y se redacta

Reutiliza `policy.ClassifyPrefix` (binario / private-key) por archivo visitado, pero con trato distinto por clase:

- **Binario** (NUL byte): igual que antes — en recursivo se saltea silenciosamente sin aparecer en resultados ni contar en ningún contador del header; en archivo único, responde `[blocked class=binary path=...]` sin buscar contenido, igual que `cat`.
- **Private-key**: ya NO se saltea/bloquea. El archivo se busca igual que cualquier otro (recursivo o único), pero cada fila que matchea el patrón se devuelve con el contenido reemplazado por un placeholder fijo (`[private-key content redacted]`) en vez del texto real de la línea. El header agrega `redacted=<n>` con la cantidad de filas redactadas por esta razón.

Motivación del cambio (pedido explícito del usuario): un agente que audita "¿hay private keys con permisos incorrectos, en lugares que no deberían?" necesita poder confirmar que un archivo existe y matchea un patrón como `BEGIN.*PRIVATE KEY` — saltear/bloquear el archivo completo hace que `grep` no sirva para ese caso de uso de seguridad, que es justamente uno de los más útiles para esta tool.

Alternativa considerada: abortar todo el `grep` como hace `cat` con un archivo bloqueado. Rechazada — en un recorrido de N archivos, un solo private-key no debería tirar abajo resultados legítimos de los demás N-1.

Alternativa considerada: aplicar el mismo trato de redacción (en vez de salteo) también a binarios. Rechazada explícitamente por el usuario — los binarios mantienen el comportamiento original de D4 sin cambios; no hay caso de uso de auditoría análogo (el contenido no es texto legible de todas formas).

Alternativa considerada: en vez de redactar el contenido completo, devolver un prefijo corto (20-40 bytes) truncado, igual que el cap normal de línea pero más agresivo. Rechazada explícitamente por el usuario — un prefijo corto de una clave privada real igual puede ser información sensible (confirma tipo de clave, puede alcanzar para reconstrucción parcial); redacción total (`[private-key content redacted]`) es la opción segura, y `path:line` ya es la señal accionable que el agente necesita para seguir investigando (ej. pedir permisos del archivo con otra tool, o revisar manualmente fuera del agente).

Alternativa considerada: mantener el header silencioso (sin contador), igual criterio que la versión anterior de D4. Rechazada explícitamente por el usuario — se prefiere señalizar `redacted=N` en el header para que el agente sepa que hay hits ocultos y pueda decidir un siguiente paso (ej. pedir el path exacto vía `find`, o revisar permisos) sin necesidad de ver el contenido real.

### D5 — Walk compartido: `internal/policy/walk.go`

Nueva función (forma tentativa, a firmar en implementación):

```go
type WalkLimits struct {
    MaxNodes int // default 50_000
}

type WalkFunc func(path string, info fs.FileInfo, depth int) error

func Walk(root string, limits WalkLimits, minDepth, maxDepth int, fn WalkFunc) (visited int, truncated bool, err error)
```

Reglas:
- `CheckPath` se aplica a **cada** nodo visitado, no solo a la raíz; un nodo denegado se saltea (no aborta el walk).
- Nunca sigue symlinks (`fs.WalkDir`-style con `d.Type()&os.ModeSymlink != 0` → skip, no `Lstat`-follow).
- Corta al llegar a `MaxNodes` nodos visitados (default 50.000), señalando `truncated=true` en el header de la tool que lo use, independiente de cuántos resultados haya devuelto.
- `minDepth`/`maxDepth` se aplican en el walk mismo (no post-filtro), para no gastar presupuesto de nodos en subárboles fuera de rango cuando `maxDepth` lo permite podar antes.

Alternativa considerada: cap solo por resultados devueltos (como hoy hace `MaxListEntries`). Rechazada — un walk sin coincidencias (ej. `find / -name no-existe`) igual recorrería todo el filesystem sin ese cap; el presupuesto tiene que limitar *trabajo*, no solo *output*.

### D6 — Presupuesto de nodos: 50.000, sin timeout de reloj

50.000 nodos es el orden de magnitud elegido (50× el cap de `list`) — suficiente para árboles de proyecto típicos (`node_modules`/`vendor` incluidos) sin habilitar un recorrido completo de `/`. No se agrega timeout de reloj adicional: el cap de nodos ya acota el I/O en el peor caso, y un timeout de reloj sumaría una segunda fuente de "truncated" a explicar en la respuesta sin beneficio claro sobre el cap de nodos.

### D7 — Forma de respuesta: mismo patrón `[tool key=val ...]`

`find` (estilo `list`, tabla markdown; columnas variables según D1b, ejemplo con las cuatro activas):

```
[find path=<abs> matches=<returned>/<total> truncated=<bool> visited=<n>]
|Path|Type|Size|ModTime|
|---|---|---|---|
```

Si el caller pide solo `showPath=true` (resto `false`), la tabla se reduce a:

```
[find path=<abs> matches=<returned>/<total> truncated=<bool> visited=<n>]
|Path|
|---|
```

`grep` (estilo `cat`, líneas de texto crudo prefijadas, no tabla — el contenido de línea puede tener `|` y rompería markdown):

```
[grep path=<abs> matches=<returned>/<total> truncated=<bool> filesScanned=<n> redacted=<n>]
<path>:<line>:<content>
```

`redacted=<n>` cuenta las filas cuyo `<content>` fue reemplazado por `[private-key content redacted]` (ver D4); filas binarias salteadas no cuentan acá ni en ningún otro contador (siguen siendo completamente silenciosas).

Bloqueo de path raíz o de archivo único binario en ambas: `[blocked class=<class> path=<abs>]`, igual forma que `cat`/`list`. Un archivo único clasificado private-key ya no usa esta forma — ver D4 (se busca y se redacta en vez de bloquear).

Cap de longitud de línea en `grep`: se reutiliza el mismo umbral de `policy.MaxBytes` (64 KiB) por línea individual antes de truncar esa fila con una marca de truncamiento en la fila (no en el header general).

### D8 — Registro y transporte

Igual patrón que `cat`/`list`: `internal/tool/find.go` con `AddFindFilesTool(server)`, `internal/tool/grep.go` con `AddGrepTool(server)`, ambas registradas en `NewHandler` (`internal/handler/server.go`) junto a las existentes. Sin cambios de transporte (Streamable HTTP), CORS, ni auth — reusan el mismo `mcp.Server` y middleware de auditoría.

## Risks / Trade-offs

- [Riesgo] Un walk de 50.000 nodos en un disco lento (red, `/proc` con muchos PIDs) puede tardar segundos y bloquear el handler → Mitigación: el cap de nodos es el límite duro; si en la práctica resulta muy lento, ajustar el valor es un cambio de constante, no de arquitectura (ver Open Questions).
- [Riesgo] Texto literal como "básico" en `grep` puede sorprender a un agente que espera BRE real (ej. `grep 'foo.bar'` esperando `.` como comodín) → Mitigación: la descripción MCP del tool (`GrepToolDescription`) debe ser explícita: "básico = texto literal, sin metacaracteres; usar `extended=true` para regex".
- [Riesgo] No seguir symlinks puede hacer que `find`/`grep -r` "no encuentren" contenido que un usuario esperaría vía un symlink común (ej. `/etc/localtime` → zoneinfo) → Mitigación aceptada como trade-off explícito de seguridad; documentar en `docs/tools/find.md` y `docs/tools/grep.md`.
- [Riesgo] `filesScanned`/`visited` en el header no distinguen "cortado por nodos" de "cortado por resultados" → Mitigación: `truncated=true` ya es suficiente señal accionable para el agente (repetir con filtro más específico); no se requiere una razón de truncamiento separada en v1.

## Migration Plan

No hay estado persistente ni breaking changes — son dos tools nuevas más un helper interno nuevo. Pasos de implementación (detalle en `tasks.md`):

1. `internal/policy/walk.go` + tests unitarios (cap de nodos, no-follow symlinks, skip de nodos denegados).
2. `internal/tool/find.go` + `AddFindFilesTool` + tests.
3. `internal/tool/grep.go` + `AddGrepTool` + tests (modo literal, modo regex, salteo de binario/private-key).
4. Registro en `internal/handler/server.go`.
5. Docs (`docs/tools/find.md`, `docs/tools/grep.md`, fila en `docs/README.md`).
6. `graphify --update .` tras el merge (regla del proyecto).

Rollback: revertir el commit; no hay migración de datos ni de config involucrada.

## Open Questions

- Valor exacto de `MaxNodes` (50.000) es una estimación inicial — puede requerir ajuste una vez que haya uso real contra árboles grandes (`node_modules`, `vendor`, `/usr`).
- Si en el futuro se pide soporte de PCRE o BRE real, requeriría una nueva decisión de diseño (motor con backtracking reintroduce el riesgo de ReDoS que D2 evita a propósito).
- ¿Hacer `MaxNodes` configurable vía variable de ambiente (`FIND_MAX_NODES`), leída por una clase/struct de config con default conservador (más bajo que 50.000, dado que no se conoce el hardware del servidor donde corre `linux-mcp`)? Queda **fuera de esta v1** — no hay código nuevo en `tasks.md` para esto todavía. Evaluar si es necesario más adelante, una vez que haya uso real que muestre si el valor fijo de D6 es un problema en la práctica.
