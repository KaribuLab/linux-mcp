# Conectar Claude Code a linux-mcp

Claude Code soporta servidores MCP de tipo `http` con headers propios, así que no necesita ningún puente.

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

En `.mcp.json` del proyecto (o el archivo de configuración de usuario):

```json
{
  "mcpServers": {
    "linux-mcp": {
      "type": "http",
      "url": "http://localhost:5000",
      "headers": {
        "Authorization": "Bearer ${LINUX_MCP_TOKEN}"
      }
    }
  }
}
```

Claude Code expande `${VAR}` desde el entorno, así que el token no queda escrito en el archivo:

```bash
export LINUX_MCP_TOKEN=<token>
```

También puedes registrarlo desde la terminal con la misma definición:

```bash
claude mcp add-json linux-mcp '{"type":"http","url":"http://localhost:5000","headers":{"Authorization":"Bearer '"$LINUX_MCP_TOKEN"'"}}'
```

Ojo con esta última forma: escribe el token literal en la configuración, y queda en el historial del shell.

## 3. Verificar

```
/mcp
```

`linux-mcp` debe aparecer conectado, con las tools `cat` y `list`.

## Renovar el token

Cuando el token expira, o después de que el servicio se reinicie, las llamadas fallan con `401`. Vuelve a ejecutar `linux-mcp auth`, actualiza `LINUX_MCP_TOKEN` y reinicia Claude Code para que tome el valor nuevo.

## Notas

- El servidor responde `401` con un desafío `Bearer` que **no** anuncia metadata OAuth, porque no hay servidor de autorización que descubrir. El token siempre sale de `linux-mcp auth`.
- No hace falta configurar `--cors`: Claude Code no es un browser y no manda el header `Origin`.

## Ver también

- [`linux-mcp auth`](../commands/auth.md)
- [Instalar con systemd](../runbooks/install-systemd.md)
