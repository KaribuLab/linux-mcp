## Why

En modo detallado (`list=true`) la tool `list` siempre devuelve las 9 columnas fijas (`Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath`), aunque el agente solo necesite una o dos (por ejemplo, solo `Size` para calcular espacio ocupado). Esto infla el output en tokens de forma innecesaria en directorios con muchas entradas. Permitir que el agente oculte columnas que no necesita reduce el consumo de tokens sin perder la información relevante.

## What Changes

- Se agregan 8 parámetros opcionales booleanos a la tool `list`, uno por cada columna del modo detallado excepto `Name`: `showSize`, `showMode`, `showOwner`, `showGroup`, `showModTime`, `showIsDir`, `showIsSymlink`, `showSymlinkPath`. Solo tienen efecto cuando `list=true`.
- Cada flag es opcional y por defecto `true` (columna visible). El agente solo necesita mandar los flags que quiere poner en `false` para ocultar columnas.
- La columna `Name` no tiene flag: siempre se incluye, sin excepción, porque una fila sin nombre de archivo no es utilizable.
- El orden de columnas es siempre el orden fijo actual (`Name, Size, Mode, Owner, Group, ModTime, IsDir, IsSymlink, SymlinkPath`); los flags solo deciden inclusión/omisión, no reordenan.
- Se usa `*bool` (puntero) en cada flag en Go para distinguir "no enviado" (default `true`) de "enviado en `false`", evitando el problema de zero-value de `bool` plano.
- No hay validación de "nombre de columna inválido": al ser flags tipados del JSON Schema, no existe la posibilidad de mandar un nombre de columna que no exista (a diferencia de un array de strings libre).
- La línea de metadatos `[list …]` agrega el campo `columns=<c1,c2,...>` cuando `list=true`, listando las columnas efectivamente visibles en orden, para que el agente sepa qué recibió sin tener que inferirlo de los flags que mandó.
- Los flags `show*` se ignoran silenciosamente cuando `list=false` (modo solo nombres, tabla `|File|` sin cambios).
- Se actualiza `docs/tools/list.md` con los nuevos parámetros, su default, y el nuevo campo `columns=` en la meta.

## Capabilities

### New Capabilities

(ninguna; es una extensión de una capability existente)

### Modified Capabilities

- `list-safe-read`: la tool `list` gana flags booleanos `show<Columna>` para ocultar columnas en modo detallado, con `Name` siempre presente y un campo `columns=` en la meta `[list …]`.

## Impact

- Código: `internal/tool/list.go` (`ListFilesArgs` gana 8 campos `*bool`, resolución de columnas visibles, construcción dinámica de header/filas), `internal/toolmeta/meta.go` (`ListHeader` gana campo `Columns`).
- Tests: `internal/tool/list_test.go` (default sin flags = 9 columnas, combinaciones de flags en `false`, flags ignorados con `list=false`, orden fijo preservado).
- Documentación: `docs/tools/list.md` (parámetros `show*`, default `true`, formato de meta actualizado).
- Sin impacto en `cat`, autenticación, políticas de path ni transporte HTTP.
- No es un cambio disruptivo para clientes existentes: todos los flags son opcionales y el comportamiento por defecto (sin flags) no cambia.

## Non-goals

- No se agrega selección/ocultamiento de columnas al modo `list=false` (tabla `|File|`).
- No se agrega reordenamiento de columnas: el orden siempre es el fijo actual.
- No se agrega ordenamiento (`sort`) ni filtros de contenido (glob/regex) de entradas; solo visibilidad de columnas.
- No se cambian los caps existentes (1000 entradas) ni la política de paths.
- No se permite ocultar la columna `Name`.
