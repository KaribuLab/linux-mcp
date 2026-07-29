## 1. Toolmeta: campo `columns` en la meta

- [x] 1.1 Agregar campo `Columns string` a `toolmeta.ListHeader` (`internal/toolmeta/meta.go`)
- [x] 1.2 Actualizar `ListHeader.String()` para agregar ` columns=<valor>` antes del `]` solo cuando `Columns != ""`
- [x] 1.3 Actualizar/agregar tests en `internal/toolmeta/meta_test.go` (con y sin `Columns`)

## 2. Tool `list`: flags de visibilidad por columna

- [x] 2.1 Definir el set canónico ordenado de columnas (`Name` primero, luego `Size`, `Mode`, `Owner`, `Group`, `ModTime`, `IsDir`, `IsSymlink`, `SymlinkPath`) en `internal/tool/list.go`
- [x] 2.2 Agregar 8 campos `*bool` a `ListFilesArgs`: `ShowSize`, `ShowMode`, `ShowOwner`, `ShowGroup`, `ShowModTime`, `ShowIsDir`, `ShowIsSymlink`, `ShowSymlinkPath`, con tag `json:"show*,omitempty"` y `jsonschema` indicando default `true`
- [x] 2.3 Implementar resolución de columnas visibles: `Name` siempre visible; cada otra columna visible si su flag es `nil` o `*flag == true`; producir la lista ordenada (nombre canónico, getter) filtrada por visibilidad
- [x] 2.4 Aplicar la resolución solo cuando `args.List == true`; ignorar todos los `Show*` cuando `args.List == false`
- [x] 2.5 Reemplazar `fileInfo.detailedRow()` fijo por construcción dinámica de header (`|Col1|Col2|...|`) y filas según la lista de columnas visibles resuelta
- [x] 2.6 Evitar cómputo innecesario por columna oculta cuando sea barato (ej. no resolver `Owner`/`Group` vía `user.Lookup*` si `ShowOwner`/`ShowGroup` son `false`); no es obligatorio para `Size`/`Mode`/`ModTime`/`IsDir` que ya vienen de `info` sin costo extra
- [x] 2.7 Poblar `toolmeta.ListHeader.Columns` con las columnas visibles (join `,`) cuando `list=true`; dejarlo vacío cuando `list=false`
- [x] 2.8 Actualizar `ListToolDescription` para mencionar los 8 flags `show*`, que default es `true`, que `Name` siempre se incluye y no tiene flag, y que el orden de columnas es siempre fijo

## 3. Tests

- [x] 3.1 Test: `list=true` sin ningún `show*` devuelve las 9 columnas en el orden actual (no-regresión)
- [x] 3.2 Test: `list=true` con `showSize=false, showModTime=false` devuelve `Name,Mode,Owner,Group,IsDir,IsSymlink,SymlinkPath` en ese orden
- [x] 3.3 Test: `list=true` con `showSize=true` explícito es equivalente a omitirlo (default)
- [x] 3.4 Test: `list=true` con todos los `show*` en `false` devuelve tabla con solo `Name`
- [x] 3.5 Test: `list=false` con `show*` en `false` ignora los flags y devuelve tabla `|File|` sin cambios
- [x] 3.6 Test: meta `[list ...]` incluye `columns=` cuando `list=true` (reflejando el subconjunto visible) y no lo incluye cuando `list=false`
- [x] 3.7 Ejecutar `go test ./...` y confirmar que los tests existentes de `list_test.go` siguen pasando sin modificación de expectativas no relacionadas

## 4. Documentación

- [x] 4.1 Actualizar `docs/tools/list.md`: los 8 parámetros `show*` (tipo, opcional, default `true`), que `Name` no tiene flag y siempre se incluye, orden fijo de columnas, y el nuevo campo `columns=` en la meta
- [x] 4.2 Revisar `docs/README.md` para confirmar que la descripción de la fila `list` sigue siendo consistente (sin cambios: el resumen es genérico y sigue vigente)

## 5. Cierre

- [x] 5.1 `go vet ./...` y `go build ./...` sin errores
- [x] 5.2 Revisar que `openspec validate --changes add-list-column-selection` siga en verde tras cualquier ajuste de specs durante la implementación
