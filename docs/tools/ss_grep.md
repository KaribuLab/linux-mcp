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

`Pid`/`Process` siguen la misma resolución que [`ss`](ss.md) (inode vía `/proc/*/fd`; bajo unit de referencia, también dueños de otros uids).

## Prompt de ejemplo (agente)

```text
Usa el tool linux-mcp `ss_grep` con pattern :25 y state=LISTEN, showPid y showProcess en true. Decí qué Pid/Process escucha ese puerto y si parece un servicio de correo u otro demonio.
```

## Notas

- Meta: `[ss_grep …]`. Relacionada: [`ss.md`](ss.md). Deploy: [`install-systemd.md`](../runbooks/install-systemd.md).
