# Tool: `list`

Lista entradas de un directorio y las formatea como tabla markdown para el cliente MCP.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `list` |
| Código | `internal/tool/list.go` |
| Registro | `AddListFilesTool` → `NewHandler` |
| Descripción MCP | List the files in a directory in markdown format |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `path` | `string` | sí | Directorio a listar |
| `all` | `bool` | no | Si `true`, incluye archivos ocultos (nombre con prefijo `.`) |
| `list` | `bool` | no | Si `true`, tabla detallada; si `false`, solo nombres |

## Respuesta

Texto markdown (`TextContent`).

### Modo simple (`list=false`)

```markdown
|File|
|---|
|nombre|
```

Cada fila se genera desde `fileInfo` con solo `Name` poblado (el resto en cero/vacío en el formato de fila).

### Modo detallado (`list=true`)

Cabecera:

```markdown
|Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath|
|---|---|---|---|---|---|---|---|---|
```

Columnas:

| Columna | Origen |
|---------|--------|
| Name | `DirEntry.Name()` |
| Size | `FileInfo.Size()` |
| Mode | `FileInfo.Mode()` |
| Owner | UID vía `syscall.Stat_t` + `user.LookupId` |
| Group | GID vía `syscall.Stat_t` + `user.LookupId` |
| ModTime | `FileInfo.ModTime()` |
| IsDir | `DirEntry.IsDir()` |
| IsSymlink | `DirEntry.Type() == os.ModeSymlink` |
| SymlinkPath | `os.Readlink` si es symlink |

## Comportamiento

1. `os.ReadDir(path)`.
2. Filtra ocultos salvo `all=true`.
3. Con `list=true`, resuelve dueño/grupo y opcionalmente destino del symlink.
4. Errores de `Info`, lookup de usuario o `Readlink` (salvo not-exist) abortan la tool con error.

## Ejemplos

**Solo nombres, sin ocultos**

```json
{
  "path": "/home/user",
  "all": false,
  "list": false
}
```

**Detalle estilo `ls -l`, con ocultos**

```json
{
  "path": "/var/log",
  "all": true,
  "list": true
}
```

## Notas / límites

- Orientada a Linux: usa `syscall.Stat_t` (no portable a todos los GOOS).
- Lookup de grupo usa `user.LookupId` con el GID (misma API que UID); en algunos sistemas el nombre de grupo puede no resolverse como se espera.
- `Readlink` recibe `file.Name()` (nombre relativo); si el CWD del proceso no es `path`, el symlink puede fallar o resolverse mal.
- Relacionada en el grafo: `ListFiles` → `ListFilesArgs`, `CallToolRequest`, `CallToolResult`; registro vía `AddListFilesTool` desde `NewHandler`.
