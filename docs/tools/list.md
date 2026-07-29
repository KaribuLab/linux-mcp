# Tool: `list`

Lista entradas de un directorio con meta mínima y tabla markdown acotada (máx. 1000 filas).

| Campo | Valor |
|-------|-------|
| Nombre MCP | `list` |
| Código | `internal/tool/list.go` |
| Registro | `AddListFilesTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.ListToolDescription` (meta `[list …]`, tabla markdown, `[blocked …]`) |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `path` | `string` | sí | Directorio a listar |
| `all` | `bool` | no | Si `true`, incluye ocultos (prefijo `.`) |
| `list` | `bool` | no | Si `true`, tabla detallada; si `false`, solo nombres |
| `showSize` | `bool` (opcional, default `true`) | no | Solo aplica con `list=true`. Si `false`, oculta la columna `Size` |
| `showMode` | `bool` (opcional, default `true`) | no | Solo aplica con `list=true`. Si `false`, oculta la columna `Mode` |
| `showOwner` | `bool` (opcional, default `true`) | no | Solo aplica con `list=true`. Si `false`, oculta la columna `Owner` |
| `showGroup` | `bool` (opcional, default `true`) | no | Solo aplica con `list=true`. Si `false`, oculta la columna `Group` |
| `showModTime` | `bool` (opcional, default `true`) | no | Solo aplica con `list=true`. Si `false`, oculta la columna `ModTime` |
| `showIsDir` | `bool` (opcional, default `true`) | no | Solo aplica con `list=true`. Si `false`, oculta la columna `IsDir` |
| `showIsSymlink` | `bool` (opcional, default `true`) | no | Solo aplica con `list=true`. Si `false`, oculta la columna `IsSymlink` |
| `showSymlinkPath` | `bool` (opcional, default `true`) | no | Solo aplica con `list=true`. Si `false`, oculta la columna `SymlinkPath` |

### Selección de columnas (`show*`)

- Pensados para ahorrar tokens en directorios con muchas entradas cuando el agente no necesita todas las columnas del modo detallado.
- Todos son opcionales y por defecto `true` (columna visible); el agente solo necesita mandar los que quiere poner en `false`.
- `Name` **no tiene flag**: siempre se incluye como primera columna, sin excepción.
- El orden de columnas es siempre el fijo (`Name, Size, Mode, Owner, Group, ModTime, IsDir, IsSymlink, SymlinkPath`); los flags solo deciden inclusión/omisión, no reordenan.
- Se ignoran silenciosamente cuando `list=false`.

## Caps (v1)

- Máx. **1000** entradas visibles (tras filtro `all`).
- Misma path policy que `cat` (`internal/policy`).

## Respuesta

Un solo `TextContent`.

### Éxito

```text
[list path=<abs> entries=<returned>/<total> truncated=<bool> next=<entry-offset>]
|File|
|---|
|nombre|
```

Con `list=true`:

```text
[list path=<abs> entries=<returned>/<total> truncated=<bool> columns=<c1,c2,...>]
|Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath|
|---|---|---|---|---|---|---|---|---|
|...|
```

- `truncated=false` → se puede omitir `next`.
- v1 no expone arg de paginación de entradas; `truncated=true` señala que hay más.
- `columns=` solo aparece cuando `list=true`; lista, en el orden de la tabla, las columnas efectivamente devueltas (todas por defecto, o el subconjunto resultante de los flags `show*`). Con `list=false` este campo no aparece.

### Bloqueo (IsError)

```text
[blocked class=path_denied path=<abs>]
```

## Comportamiento

1. `policy.CheckPath` antes de `ReadDir`.
2. Filtra ocultos salvo `all=true`.
3. Corta a 1000 entradas; meta `ListHeader` + filas en `strings.Builder`.
4. Con `list=true`, resuelve las columnas visibles a partir de los flags `show*` (`Name` siempre incluida) antes de construir el header y las filas.
5. Symlink: `os.Readlink(filepath.Join(dir, name))` (no basename/CWD); solo se resuelve si `showSymlinkPath` está visible.
6. Grupo: `user.LookupGroupId` (GID); owner: `user.LookupId` (UID). Si el lookup falla, se usa el id numérico. Solo se resuelven si `showOwner`/`showGroup` están visibles, respectivamente.

## Ejemplos

**Solo nombres**

```json
{ "path": "/home/user", "all": false, "list": false }
```

**Detalle completo**

```json
{ "path": "/var/log", "all": true, "list": true }
```

**Detalle con columnas reducidas (ahorro de tokens)**

```json
{ "path": "/var/log", "list": true, "showOwner": false, "showGroup": false, "showModTime": false, "showIsSymlink": false, "showSymlinkPath": false }
```

Devuelve solo `Name|Size|Mode|IsDir|`.

## Notas / límites

- Orientada a Linux (`syscall.Stat_t`).
- Relacionada: `internal/policy`, `internal/toolmeta`.
