# Tool: `ss`

Lista sockets vía **netlink sock_diag** in-process (`github.com/florianl/go-diag`). Nunca ejecuta el binario `ss`.

| Campo | Valor |
|-------|-------|
| Nombre MCP | `ss` |
| Código | `internal/tool/ss.go` |
| Registro | `AddSsTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.SsToolDescription` |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `state` | `string` | no | `LISTEN` (default), `ESTAB`, `all` |
| `family` | `string` | no | `inet` (default IPv4+IPv6), `inet4`, `inet6`, `unix`, `all` |
| `showState` … `showFamily` | `bool` (default `true`) | no | Ocultan columnas opcionales |

Identidad siempre: `Proto`, `Local`. Orden fijo. Meta `columns=`.

`Pid`/`Process` se resuelven por inode vía `/proc/*/fd` cuando el proceso es visible.

## Prompt de ejemplo (agente)

```text
Usa el tool linux-mcp `ss` con state=LISTEN y family=inet (defaults) ocultando Peer, y resume qué servicios escuchan en 0.0.0.0 o ::.
```

## Notas

- Modelo amplio: para conexiones establecidas usa `state=ESTAB`.
- Sin `os/exec`. Relacionada: [`ss_grep.md`](ss_grep.md).
