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

`Pid`/`Process` se resuelven por inode vía `/proc/*/fd` (`socket:[inode]`). Con la [unit systemd de referencia](../runbooks/install-systemd.md) (`CAP_SYS_PTRACE`, sin `ProtectProc=invisible`) esa resolución alcanza procesos de otros uids. Sin esas privilegios las columnas pueden quedar vacías; no se inventan Pids.

## Prompt de ejemplo (agente)

```text
Usa el tool linux-mcp `ss` con state=LISTEN y showPid/showProcess en true. Para el puerto que te indique (o un LISTEN sospechoso), reportá Proto, Local, Pid, Process y User; si hace falta cruzá el Pid con ps o ps_grep.
```

## Notas

- Modelo amplio: para conexiones establecidas usa `state=ESTAB`.
- Sin `os/exec`. Relacionada: [`ss_grep.md`](ss_grep.md).
