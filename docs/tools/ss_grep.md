# Tool: `ss_grep`

Lista sockets (`ss`) y filtra **filas** por patrón. Sin binario `ss`.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `ss_grep` |
| Código | `internal/tool/ss_grep.go` |
| Registro | `AddSsGrepTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.SsGrepToolDescription` |

## Parámetros

Los de `ss` más `pattern` / `extended` / `ignoreCase`.

Glob (`* ? [`) matchea columna **Local**. Literal/RE2 sobre la fila completa — ojo: `Peer` en LISTEN suele ser `0.0.0.0:0`; preferí patrones como `0.0.0.0:3306`.

## Prompt de ejemplo (agente)

```text
Usa el tool linux-mcp `ss_grep` con pattern 0.0.0.0: y state=LISTEN para listar binds en todas las interfaces, con showPid y showProcess en true.
```

## Notas

- Meta: `[ss_grep …]`. Relacionada: [`ss.md`](ss.md).
