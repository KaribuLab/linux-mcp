# Documentación

Documentación del servidor MCP **linux-mcp**: comandos, tools expuestas a clientes, integración por agente y runbooks de operación.

## Arquitectura (resumen)

```
cmd/linux-mcp/main.go
  └─ command.Execute()                       # cobra: serve, auth
       ├─ serve
       │    ├─ token.NewSigner                # clave HMAC efímera, solo en memoria
       │    ├─ issuer.Listen                  # socket unix + SO_PEERCRED → emite JWT
       │    └─ handler.NewHandler
       │         ├─ withHostCheck             # anti DNS rebinding
       │         ├─ withCORS                  # allowlist de Origin, fail-closed
       │         ├─ RequireBearerToken        # scope mcp:read
       │         └─ mcp.NewStreamableHTTPHandler → 127.0.0.1:5000
       │              ├─ tool.AddCatFileTool
       │              └─ tool.AddListFilesTool
       └─ auth                                # pide el token por el socket
```

## Comandos

| Comando | Doc | Código |
|---------|-----|--------|
| [`serve`](commands/serve.md) | Levanta el endpoint MCP y el socket de emisión | `internal/command/serve.go` |
| [`auth`](commands/auth.md) | Obtiene un token para conectarse | `internal/command/auth.go` |

## Tools

| Tool | Doc | Código |
|------|-----|--------|
| [`cat`](tools/cat.md) | Leer archivo texto acotado (meta + body) | `internal/tool/cat.go` |
| [`list`](tools/list.md) | Listar directorio (meta + markdown) | `internal/tool/list.go` |

## Agentes

Cómo configurar cada cliente MCP contra el servidor, con túnel SSH y token bearer.

| Agente | Doc |
|--------|-----|
| Claude Code | [agents/claude.md](agents/claude.md) |
| Codex CLI | [agents/codex.md](agents/codex.md) |
| OpenCode | [agents/opencode.md](agents/opencode.md) |

## Runbooks

| Runbook | Descripción |
|---------|-------------|
| [Instalar con systemd](runbooks/install-systemd.md) | Usuario `mcp-agent`, grupo `mcp-admin`, unit, tokens, update, uninstall |
| [Desarrollo local](runbooks/local-development.md) | `task dev`, `task token`, MCP Inspector, cuándo hace falta `--cors` |

## Convención

Toda tool MCP y todo comando de CLI, nuevo o modificado, **debe** tener documentación actualizada en `docs/tools/<nombre>.md` o `docs/commands/<nombre>.md` y aparecer en la tabla correspondiente. OpenSpec refuerza esta regla en `openspec/config.yaml`.
