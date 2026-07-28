# Documentación

Documentación del servidor MCP **linux-mcp** y de cada tool expuesta a clientes.

## Arquitectura (resumen)

```
cmd/linux-mcp/main.go
  └─ handler.NewHandler()
       ├─ mcp.NewServer (Implementation: linux-mcp)
       ├─ tool.AddCatFileTool
       ├─ tool.AddListFilesTool
       └─ Streamable HTTP + CORS → localhost:5000
```

Fuente del grafo de código: `graphify-out/` (`NewHandler` → tools).

## Tools

| Tool | Doc | Código |
|------|-----|--------|
| [`cat`](tools/cat.md) | Leer archivo texto acotado (meta + body) | `internal/tool/cat.go` |
| [`list`](tools/list.md) | Listar directorio (meta + markdown) | `internal/tool/list.go` |

## Runbooks

| Runbook | Descripción |
|---------|-------------|
| [Instalar con systemd](runbooks/install-systemd.md) | Usuario `mcp-agent`, unit, enable, update, uninstall |

## Convención

Toda tool MCP nueva o modificada **debe** tener documentación actualizada en `docs/tools/<nombre>.md` y aparecer en esta tabla. OpenSpec refuerza esta regla en `openspec/config.yaml`.
