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
[list path=<abs> entries=<returned>/<total> truncated=<bool>]
|Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath|
|---|---|---|---|---|---|---|---|---|
|...|
```

- `truncated=false` → se puede omitir `next`.
- v1 no expone arg de paginación de entradas; `truncated=true` señala que hay más.

### Bloqueo (IsError)

```text
[blocked class=path_denied path=<abs>]
```

## Comportamiento

1. `policy.CheckPath` antes de `ReadDir`.
2. Filtra ocultos salvo `all=true`.
3. Corta a 1000 entradas; meta `ListHeader` + filas en `strings.Builder`.
4. Symlink: `os.Readlink(filepath.Join(dir, name))` (no basename/CWD).
5. Grupo: `user.LookupGroupId` (GID); owner: `user.LookupId` (UID). Si el lookup falla, se usa el id numérico.

## Ejemplos

**Solo nombres**

```json
{ "path": "/home/user", "all": false, "list": false }
```

**Detalle**

```json
{ "path": "/var/log", "all": true, "list": true }
```

## Notas / límites

- Orientada a Linux (`syscall.Stat_t`).
- Relacionada: `internal/policy`, `internal/toolmeta`.
