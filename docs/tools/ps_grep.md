# Tool: `ps_grep`

Lista procesos (`ps`) y filtra **filas** por patrón (semántica `ps | grep`). Sin binarios del host.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `ps_grep` |
| Código | `internal/tool/ps_grep.go` |
| Registro | `AddPsGrepTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.PsGrepToolDescription` |

## Parámetros

Los de `ps` más:

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `pattern` | `string` | sí | Filtro de filas |
| `extended` | `bool` | no | RE2 sobre la fila completa |
| `ignoreCase` | `bool` | no | Case-insensitive |

### Modos de `pattern` (`extended=false`)

| Patrón | Comportamiento |
|--------|----------------|
| Contiene `*`, `?` o `[` | Glob contra columna **Comm** |
| Sin metacaracteres | Subcadena literal sobre la fila completa |
| `extended=true` | RE2 sobre la fila completa |

## Prompt de ejemplo (agente)

```text
Usa el tool linux-mcp `ps_grep` con pattern nginx* para ver procesos cuyo Comm matchee ese glob, e incluye Cmdline y Exe para revisar binario real vs argv.
```

## Notas

- Meta: `[ps_grep …]`. Cap 1000. Relacionada: [`ps.md`](ps.md).
