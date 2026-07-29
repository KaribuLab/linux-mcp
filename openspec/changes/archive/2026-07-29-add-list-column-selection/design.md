## Context

`list` (`internal/tool/list.go`) tiene modo detallado (`list=true`) con 9 columnas fijas: `Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath`. El header/filas se generan con `fileInfo.detailedRow()` (formato fijo vía `fmt.Sprintf`) y la meta usa `toolmeta.ListHeader.String()`. No hay forma de pedir un subconjunto: un agente que solo quiere `Size` igual paga el costo en tokens de las 9 columnas por cada fila.

Objetivo: dejar que el agente oculte columnas del modo detallado, minimizando el riesgo de que "alucine" un parámetro mal formado, y manteniendo compatibilidad total cuando no especifica nada.

Se descartó un diseño previo basado en `columns []string` (array de nombres de columna libres) porque exponía una superficie de error nueva: el agente debía recordar exactamente los mismos literales del header markdown (case-sensitive), y un typo producía un error de tool call en vez de una lista. El approach actual reemplaza eso por flags booleanos tipados, uno por columna, visibles como campos individuales en el JSON Schema de la tool.

## Goals / Non-Goals

**Goals:**
- Un flag booleano opcional por columna del modo detallado, excepto `Name`: `showSize`, `showMode`, `showOwner`, `showGroup`, `showModTime`, `showIsDir`, `showIsSymlink`, `showSymlinkPath`.
- Default `true` (columna visible) para todos los flags cuando no se envían.
- `Name` siempre presente, sin flag que la controle.
- Comportamiento por defecto (sin flags) idéntico al actual: 9 columnas, mismo orden.
- Eliminar la clase de error "nombre de columna inválido" por construcción (no existe un string libre que validar).
- Meta `[list ...]` refleja las columnas efectivamente visibles cuando `list=true`.

**Non-Goals:**
- No afecta `list=false` (los flags `show*` se ignoran ahí).
- No permite reordenar columnas: el orden es siempre el fijo actual.
- No agrega orden/sort de filas ni filtros de contenido.
- No cambia caps de entradas (1000) ni policy de paths.
- No permite ocultar `Name`.

## Decisions

### 1. Flags booleanos por columna en vez de `columns []string`
Se agregan 8 campos opcionales a `ListFilesArgs`, uno por columna (todas menos `Name`), en vez de un array de strings libre. Cada flag aparece como un campo independiente y tipado (`boolean`) en el JSON Schema que el cliente MCP le muestra al modelo, lo que reduce drásticamente el riesgo de que el agente "alucine" un nombre de columna que no existe: con un array de strings el agente debe recordar y escribir bien un literal; con flags, el schema mismo enumera las opciones válidas.
Alternativa descartada (versión previa de este design): `columns []string` con match exacto contra los nombres del header — funcional, pero delega en el agente la responsabilidad de escribir el string correcto y obliga a diseñar/mantener un requisito completo de rechazo de nombres inválidos.

### 2. Tipo `*bool` (puntero), no `bool` plano — semántica de default `true`
Go, al decodificar JSON, deja un campo `bool` ausente en su zero-value (`false`). Si el default deseado es `true` y usáramos `bool` plano, un agente que omite `showSize` terminaría ocultando la columna sin querer (`false` por zero-value, indistinguible de "el agente pidió false explícitamente"). Por eso cada flag es `*bool`: `nil` significa "no enviado → default `true` → visible"; `&false` oculta la columna explícitamente; `&true` es equivalente al default pero válido de enviar.
Alternativa descartada: `bool` plano con default `true` "documentado" solo en la descripción — no es seguro, porque el decoder de Go no tiene forma de distinguir "omitido" de "enviado false" sin el puntero.

### 3. `Name` no tiene flag: siempre presente, sin excepción
No existe `showName`. La columna `Name` se agrega siempre como primera columna de la tabla. Esto reemplaza la lógica anterior de "forzar Name si falta en la selección": ahora no hay forma de pedir su ausencia, así que no hay nada que forzar en runtime.
Alternativa descartada: agregar `showName *bool` e ignorarlo si viene en `false` — agrega un campo al schema que nunca tiene efecto real, lo cual es confuso para el agente que lo intente usar.

### 4. No existe validación de "columna inválida"
Al ser campos tipados (`boolean`) y no un string libre, no hay una clase de error "nombre de columna desconocido" que diseñar: un valor no-booleano en uno de estos campos es simplemente un error de tipo de JSON Schema que el propio cliente MCP rechaza antes de invocar la tool, y una clave JSON desconocida es ignorada por el decoder de Go (comportamiento estándar de `encoding/json`, sin necesidad de código adicional). Esto elimina un requisito completo (y sus tests) respecto al diseño anterior.

### 5. Orden de columnas fijo; construcción dinámica de filas por inclusión
El orden de la tabla siempre es `Name, Size, Mode, Owner, Group, ModTime, IsDir, IsSymlink, SymlinkPath`. Internamente se arma una lista ordenada de pares `(nombre canónico, visible bool, getter func(fileInfo) string)` resuelta una vez por request (no por fila): `Name` siempre `visible=true`; el resto según `flag == nil || *flag`. El header y cada fila se construyen iterando solo las entradas con `visible=true`.
Alternativa descartada: mantener `detailedRow()` con `Sprintf` fijo y post-procesar el string para quitar columnas — más frágil (depende de separar por `|`) y no ahorra el cómputo de campos no usados (ej. `user.LookupId` para `Owner` si `showOwner=false`); la resolución ordenada permite además saltar cómputo innecesario por columna oculta.

### 6. Meta `[list ...]` gana campo `columns=<c1,c2,...>` cuando `list=true`
Igual que en el diseño anterior: `toolmeta.ListHeader` gana un campo `Columns string` (ya formateado, ej. `"Name,Size"`), agregado al final antes del `]` solo cuando no está vacío. Con `list=false` el campo queda vacío y no se imprime. Con `list=true` y sin flags, `Columns` lleva las 9 columnas por defecto. Esto le da al agente una confirmación explícita de qué columnas recibió, sin tener que recordar qué flags mandó.

## Risks / Trade-offs

- [Riesgo] Perder la capacidad de reordenar columnas (existía en el diseño con array) → **Mitigación**: no era un requisito real del pedido original (ahorrar tokens), y el orden fijo es más predecible para el agente y más simple de documentar.
- [Riesgo] 8 campos booleanos infla el tamaño del JSON Schema de la tool (se envía una vez por sesión/list de tools, no por llamada) → **Mitigación**: costo fijo y único, insignificante comparado con el ahorro por llamada en directorios grandes; se mantienen descripciones cortas por campo.
- [Riesgo] Un agente podría no saber que el default es `true` y mandar los 8 flags en `true` explícitamente en cada llamada, sin ahorro → **Mitigación**: documentar el default `true` en la descripción `jsonschema` de cada campo y en `ListToolDescription`.
- [Riesgo] Cambiar `ListHeader.String()` altera el string exacto de la meta en modo detallado (agrega `columns=...`) → **Mitigación**: es un campo nuevo al final, aditivo; ningún test existente parsea la meta completa por igualdad estricta (solo `HasPrefix`/`Contains`), y se documenta el nuevo formato en `docs/tools/list.md` y en el delta spec.

## Migration Plan

Sin migración de datos ni breaking change. Despliegue normal: build + release de binario. Clientes que no usan los flags `show*` no ven cambios de comportamiento (salvo el campo aditivo `columns=` en la meta de modo detallado, que es solo informativo).

## Open Questions

Ninguna pendiente; alcance acotado a `list=true`.
