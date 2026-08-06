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
       │              ├─ tool.AddListFilesTool
       │              ├─ tool.AddFindFilesTool
       │              ├─ tool.AddGrepTool
       │              ├─ tool.AddListGrepTool
       │              ├─ tool.AddFindGrepTool
       │              ├─ tool.AddPsTool
       │              ├─ tool.AddPsGrepTool
       │              ├─ tool.AddSsTool
       │              └─ tool.AddSsGrepTool
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
| [`find`](tools/find.md) | Buscar entradas por metadata en un árbol (meta + markdown, columnas configurables) | `internal/tool/find.go` |
| [`grep`](tools/grep.md) | Buscar un patrón en un archivo o árbol (meta + filas de texto) | `internal/tool/grep.go` |
| [`list_grep`](tools/list_grep.md) | Listar y filtrar filas (`ls \| grep`; meta + markdown) | `internal/tool/list_grep.go` |
| [`find_grep`](tools/find_grep.md) | Find + contenido (`find \| xargs grep`; meta + filas) | `internal/tool/find_grep.go` |
| [`ps`](tools/ps.md) | Listar procesos vía `/proc` (meta + markdown, `show*`) | `internal/tool/ps.go` |
| [`ps_grep`](tools/ps_grep.md) | Procesos filtrados (`ps \| grep`; meta + markdown) | `internal/tool/ps_grep.go` |
| [`ss`](tools/ss.md) | Listar sockets vía netlink (meta + markdown, `show*`) | `internal/tool/ss.go` |
| [`ss_grep`](tools/ss_grep.md) | Sockets filtrados (meta + markdown) | `internal/tool/ss_grep.go` |

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
| [Instalar con one-liner](runbooks/install-one-line.md) | Forma recomendada: `curl … | sudo sh`. POSIX sh, idempotente, valida SHA256, automatiza todo el runbook de systemd |
| [Instalar con systemd](runbooks/install-systemd.md) | Pasos manuales: usuario `mcp-agent`, grupo `mcp-admin`, caps DAC+ptrace, unit, tokens, update, uninstall |
| [Desarrollo local](runbooks/local-development.md) | `task dev`, `task token`, MCP Inspector, cuándo hace falta `--cors` |

## Convención

Toda tool MCP y todo comando de CLI, nuevo o modificado, **debe** tener documentación actualizada en `docs/tools/<nombre>.md` o `docs/commands/<nombre>.md` y aparecer en la tabla correspondiente. OpenSpec refuerza esta regla en `openspec/config.yaml`.

Cada `docs/tools/<nombre>.md` **debe** incluir la sección `## Prompt de ejemplo (agente)` con al menos un prompt en español neutro que un humano pueda pegar a un agente para ejercitar esa tool vía linux-mcp (forma sugerida: `Usa el tool linux-mcp \`<nombre>\` para …`).
