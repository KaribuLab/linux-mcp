# Conectar Codex CLI a linux-mcp

Codex CLI soporta transporte Streamable HTTP y puede tomar el token desde una variable de entorno, sin escribirlo en la configuración.

## 1. Preparar el acceso

En el host del servicio, como miembro del grupo `mcp-admin`:

```bash
linux-mcp auth --ttl 8h
```

Desde tu máquina, abre el túnel (el servidor solo escucha en loopback):

```bash
ssh -N -L 5000:127.0.0.1:5000 usuario@host-del-servicio
```

## 2. Configurar el servidor MCP

En `~/.codex/config.toml`:

```toml
[mcp_servers.linux-mcp]
url = "http://localhost:5000"
bearer_token_env_var = "LINUX_MCP_TOKEN"
```

Codex lee el token de esa variable en cada arranque y arma el header `Authorization: Bearer <token>`:

```bash
export LINUX_MCP_TOKEN=<token>
```

Si necesitas mandar headers adicionales, `http_headers` acepta valores literales y `env_http_headers` valores tomados del entorno.

## 3. Verificar

Arranca Codex y pídele que liste las tools disponibles: deben aparecer `cat` y `list` bajo `linux-mcp`.

## Renovar el token

Cuando el token expira, o después de que el servicio se reinicie, las llamadas fallan con `401`. Vuelve a ejecutar `linux-mcp auth`, exporta el valor nuevo en `LINUX_MCP_TOKEN` y arranca Codex otra vez.

## Notas

- `url` y `command` son excluyentes: la entrada con `url` es la de transporte HTTP y no debe llevar `command`.
- No hace falta configurar `--cors`: Codex no es un browser y no manda el header `Origin`.

## Ver también

- [`linux-mcp auth`](../commands/auth.md)
- [Instalar con systemd](../runbooks/install-systemd.md)
