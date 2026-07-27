# Tool: `cat`

Lee el contenido completo de un archivo del sistema de archivos local y lo devuelve como texto al cliente MCP.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `cat` |
| Código | `internal/tool/cat.go` |
| Registro | `AddCatFileTool` → `NewHandler` |
| Descripción MCP | Read the contents of a file |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `path` | `string` | sí | Ruta absoluta o relativa del archivo a leer |

## Respuesta

- Contenido: un bloque de texto (`TextContent`) con el cuerpo del archivo (UTF-8 interpretado como string Go).
- Error: si `os.ReadFile` falla (no existe, sin permiso, etc.), la tool retorna error al cliente.

## Comportamiento

1. Lee todo el archivo en memoria con `os.ReadFile(path)`.
2. No trunca ni pagina; archivos grandes pueden ser costosos en tokens/memoria.
3. No valida tipo MIME; binarios se interpretan como string (pueden verse corruptos).

## Ejemplos

**Leer un archivo**

```json
{
  "path": "/etc/hostname"
}
```

## Notas / límites

- Pensada para archivos de texto.
- Sin sandbox de rutas: el proceso ve el FS con los permisos del usuario que corre el servidor.
- Relacionada en el grafo: `CatFile` → `CatFileArgs`, `CallToolRequest`, `CallToolResult`.
