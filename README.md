# linux-mcp

[![CI](https://github.com/KaribuLab/linux-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/KaribuLab/linux-mcp/actions/workflows/ci.yml)

Servidor [MCP](https://modelcontextprotocol.io/) en Go que expone herramientas del sistema de archivos Linux a clientes MCP (Claude, Codex, OpenCode, MCP Inspector, etc.).

Usa transporte **Streamable HTTP** en `http://localhost:5000` y exige un token bearer en cada llamada. El token lo emite el propio binario con `linux-mcp auth`, firmado con una clave que se genera en memoria al arrancar y nunca toca el disco.

## Requisitos

- Go 1.26+
- [Task](https://taskfile.dev/) opcional (`go tool task` ya está en el módulo)

## Instalación

### Descargar un binario publicado

Cada tag `v*` publica binarios para `linux/amd64` y `linux/arm64` junto con un archivo `SHA256SUMS` en [Releases](https://github.com/KaribuLab/linux-mcp/releases).

```bash
VERSION=v1.0.0
BASE=https://github.com/KaribuLab/linux-mcp/releases/download/$VERSION
curl -sSLO $BASE/linux-mcp-linux-amd64
curl -sSLO $BASE/SHA256SUMS

sha256sum --ignore-missing -c SHA256SUMS
sudo install -m 0755 linux-mcp-linux-amd64 /usr/local/bin/linux-mcp
linux-mcp --version
```

Solo se publican binarios de Linux: la emisión de tokens usa `SO_PEERCRED` y `/proc`, que no existen en otros sistemas operativos.

### Compilar desde el repo

```bash
git clone https://github.com/KaribuLab/linux-mcp.git
cd linux-mcp
go tool task build
```

El binario queda en `dist/linux-mcp-linux-<arch>`.

Sin Task:

```bash
go build -o ./tmp/main ./cmd/linux-mcp
```

## Uso

### Ejecutar el servidor

```bash
linux-mcp serve
```

Escucha en `127.0.0.1:5000` y abre el socket de emisión de tokens en `/run/linux-mcp/issue.sock`. Flags y errores de arranque: [docs/commands/serve.md](docs/commands/serve.md).

### Obtener un token

```bash
TOKEN=$(linux-mcp auth --ttl 8h)
```

El token sale por stdout y sale a nombre de quien ejecuta el comando, según lo que reporta el kernel. Quién puede pedirlo lo decide el permiso del socket, no la aplicación. Detalles: [docs/commands/auth.md](docs/commands/auth.md).

| Comando | Para qué |
|---------|----------|
| [`serve`](docs/commands/serve.md) | Levanta el endpoint MCP y el socket de emisión |
| [`auth`](docs/commands/auth.md) | Obtiene un token para conectarse |

### Build

```bash
go tool task build          # arquitectura del host
go tool task build:all      # linux/amd64 y linux/arm64
go tool task linux_arm64:build
go tool task clean
```

Los Taskfiles por arquitectura están en `taskfiles/`. Para fijar la versión que reporta `--version`:

```bash
VERSION=v1.0.0 go tool task build:all
```

### Desarrollo con recarga

```bash
task dev      # servidor con air y socket local en tmp/
task token    # token contra ese servidor, desde otra terminal
```

Guía completa: [docs/runbooks/local-development.md](docs/runbooks/local-development.md).

### MCP Inspector

1. Arranca el servidor y genera un token.
2. Transporte **Streamable HTTP**.
3. URL: `http://localhost:5000`
4. Pega el token en *Authentication → Bearer Token*.
5. Connect.

No hace falta configurar `--cors`: el Inspector llega al servidor a través de su propio proxy, que no manda header `Origin`.

### Conectar un agente

| Agente | Guía |
|--------|------|
| Claude Code | [docs/agents/claude.md](docs/agents/claude.md) |
| Codex CLI | [docs/agents/codex.md](docs/agents/codex.md) |
| OpenCode | [docs/agents/opencode.md](docs/agents/opencode.md) |

Como el servidor solo escucha en loopback, el acceso remoto es por túnel SSH:

```bash
ssh -N -L 5000:127.0.0.1:5000 usuario@host-del-servicio
```

## Herramientas

| Tool | Descripción | Documentación |
|------|-------------|---------------|
| `cat` | Lee archivo de texto acotado (meta + body; caps 100 líneas ∩ 64 KiB) | [docs/tools/cat.md](docs/tools/cat.md) |
| `list` | Lista directorio (meta + tabla markdown; cap 1000) | [docs/tools/list.md](docs/tools/list.md) |
| `find` | Busca entradas por metadata en un árbol (meta + tabla markdown, columnas configurables; cap 1000/50.000 nodos) | [docs/tools/find.md](docs/tools/find.md) |
| `grep` | Busca un patrón en un archivo o árbol, literal o RE2 (meta + filas de texto; private-key redactado, no bloqueado) | [docs/tools/grep.md](docs/tools/grep.md) |

Índice y convención de documentación: [docs/README.md](docs/README.md).

Instalación como servicio: [docs/runbooks/install-systemd.md](docs/runbooks/install-systemd.md) (unit en `deploy/systemd/linux-mcp.service`).

Toda tool y todo comando nuevo o modificado debe documentarse en `docs/tools/` o `docs/commands/` (regla OpenSpec en `openspec/config.yaml`).

## Estructura

```
cmd/linux-mcp/        # entrypoint
internal/command/     # CLI cobra (serve, auth)
internal/handler/     # servidor MCP + Host/CORS + bearer
internal/token/       # emisión y verificación de JWT
internal/issuer/      # socket unix de emisión (SO_PEERCRED)
internal/tool/        # tools (cat, list, find, grep)
internal/policy/      # denylist, sniff, lectura acotada, recorrido de árbol acotado (walk)
internal/toolmeta/    # headers de respuesta
docs/commands/        # documentación por comando
docs/tools/           # documentación por tool
docs/agents/          # integración por cliente MCP
docs/runbooks/        # runbooks (systemd, desarrollo local)
deploy/systemd/       # unit de referencia
taskfiles/            # builds por OS/ARCH
Taskfile.yml          # build principal
openspec/             # configuración OpenSpec
```

## Licencia

Copyright 2026 KaribuLab

Licenciado bajo la [Apache License 2.0](LICENSE).
