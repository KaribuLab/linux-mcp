# linux-mcp

Servidor [MCP](https://modelcontextprotocol.io/) en Go que expone herramientas del sistema de archivos Linux a clientes MCP (Cursor, Claude, MCP Inspector, etc.).

Usa transporte **Streamable HTTP** en `http://localhost:5000`.

## Requisitos

- Go 1.26+
- [Task](https://taskfile.dev/) opcional (`go tool task` ya está en el módulo)

## Instalación

```bash
git clone https://github.com/KaribuLab/linux-mcp.git
cd linux-mcp
go tool task build
```

El binario queda en `dist/linux-mcp-<os>-<arch>`.

Sin Task:

```bash
go build -o ./tmp/main ./cmd/linux-mcp
```

## Uso

### Ejecutar el servidor

```bash
go run ./cmd/linux-mcp
# o
./dist/linux-mcp-$(go env GOOS)-$(go env GOARCH)
```

Escucha en `localhost:5000`.

### Build

```bash
go tool task build          # host OS/ARCH
go tool task build:all      # linux/darwin/windows × amd64/arm64
go tool task linux_arm64:build
go tool task clean
```

Los Taskfiles por plataforma están en `taskfiles/`.

### Desarrollo con recarga

```bash
go tool air
```

### MCP Inspector

1. Arranca el servidor.
2. Transporte **Streamable HTTP**.
3. URL: `http://localhost:5000`
4. Connect.

El handler incluye CORS para que Inspector (browser) pueda conectar.

## Herramientas

| Tool | Descripción | Documentación |
|------|-------------|---------------|
| `cat` | Lee archivo de texto acotado (meta + body; caps 100 líneas ∩ 64 KiB) | [docs/tools/cat.md](docs/tools/cat.md) |
| `list` | Lista directorio (meta + tabla markdown; cap 1000) | [docs/tools/list.md](docs/tools/list.md) |

Índice y convención de documentación: [docs/README.md](docs/README.md).

Instalación como servicio: [docs/runbooks/install-systemd.md](docs/runbooks/install-systemd.md) (unit en `deploy/systemd/linux-mcp.service`).

Toda tool nueva o modificada debe documentarse en `docs/tools/` (regla OpenSpec en `openspec/config.yaml`).

## Estructura

```
cmd/linux-mcp/       # entrypoint
internal/handler/     # servidor MCP + CORS
internal/tool/        # tools (cat, list)
internal/policy/      # denylist, sniff, lectura acotada
internal/toolmeta/    # headers de respuesta
docs/tools/           # documentación por tool
docs/runbooks/        # runbooks (systemd, etc.)
deploy/systemd/       # unit de referencia
taskfiles/            # builds por OS/ARCH
Taskfile.yml          # build principal
openspec/             # configuración OpenSpec
```

## Licencia

Copyright 2026 KaribuLab

Licenciado bajo la [Apache License 2.0](LICENSE).
