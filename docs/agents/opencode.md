# Conectar OpenCode a linux-mcp

OpenCode soporta servidores MCP remotos con headers propios, así que no necesita ningún puente.

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

En `opencode.json`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "linux-mcp": {
      "type": "remote",
      "url": "http://localhost:5000",
      "enabled": true,
      "oauth": false,
      "headers": {
        "Authorization": "Bearer {env:LINUX_MCP_TOKEN}"
      }
    }
  }
}
```

OpenCode interpola `{env:VAR}`, así que el token no queda escrito en el archivo:

```bash
export LINUX_MCP_TOKEN=<token>
```

`oauth: false` importa aquí: linux-mcp no tiene servidor de autorización, y sin ese ajuste OpenCode puede intentar un flujo OAuth al recibir el `401`.

## 3. Verificar

Arranca OpenCode y revisa las tools disponibles: deben aparecer `cat` y `list` bajo `linux-mcp`.

## Renovar el token

Cuando el token expira, o después de que el servicio se reinicie, las llamadas fallan con `401`. Vuelve a ejecutar `linux-mcp auth`, exporta el valor nuevo en `LINUX_MCP_TOKEN` y reinicia OpenCode.

## Notas

- El `timeout` por defecto para descubrir las tools es de 5000 ms; súbelo si el túnel es lento.
- No hace falta configurar `--cors`: OpenCode no es un browser y no manda el header `Origin`.

## Ver también

- [`linux-mcp auth`](../commands/auth.md)
- [Instalar con systemd](../runbooks/install-systemd.md)
