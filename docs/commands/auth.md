# `linux-mcp auth`

Pide un token al servidor que ya está corriendo. El token identifica a **quien ejecuta el comando**, según lo que reporta el kernel: no hay flag para pedir uno a nombre de otra persona.

```bash
linux-mcp auth [flags]
```

## Flags

| Flag | Default | Descripción |
|------|---------|-------------|
| `--socket` | `/run/linux-mcp/issue.sock` | Socket de emisión del servidor. Debe ser el mismo `--socket` con el que arrancó `serve` |
| `--ttl` | `8h` | Vida solicitada para el token. El servidor la recorta a su `--max-ttl` |

## Salida

La salida está separada para que el token se pueda capturar sin ruido:

| Flujo | Contenido |
|-------|-----------|
| stdout | El token, y nada más |
| stderr | `subject`, expiración efectiva y avisos |

```bash
TOKEN=$(linux-mcp auth --ttl 8h)
# stderr:
# subject: maria
# expires: 2026-07-28T07:37:02-04:00
```

Si pides más de lo que el servidor concede, el token se emite recortado y el aviso va a stderr:

```
requested lifetime exceeds the server maximum; token capped
```

## Quién puede ejecutarlo

Conectarse al socket requiere permiso de escritura sobre él. En una instalación con systemd el socket queda `0660 mcp-agent:mcp-admin`, así que pueden pedir tokens los miembros del grupo `mcp-admin`. La autorización la aplica el kernel, no el proceso.

## Errores

Todos salen con código `1` y escriben `error: <detalle>` en stderr.

| Mensaje | Causa | Solución |
|---------|-------|----------|
| `permission denied on ...: you must belong to the group that owns the socket...` | No estás en `mcp-admin`, o la membresía todavía no aplica en esta sesión | `id -nG` para confirmar; si ya te agregaron, cerrar sesión y volver a entrar |
| `no socket at ...: the server may not be running, or it uses a different --socket` | El servicio está caído, o `serve` usa otra ruta | `systemctl status linux-mcp`; comparar el `--socket` de ambos comandos |
| `invalid ttl "..."` | Formato no reconocido | Usar duraciones de Go: `30m`, `8h`, `2h30m` |
| `ttl must be positive, got "..."` | `--ttl 0s` o negativo | Pedir una duración mayor que cero |

## Vigencia del token

El token deja de servir cuando expira **o cuando se reinicia el servicio**, porque la clave de firma se regenera en cada arranque y no se persiste. En ambos casos la respuesta del endpoint MCP es `401` y la solución es volver a ejecutar `auth`.

## Ver también

- [`linux-mcp serve`](serve.md)
- [Instalar con systemd](../runbooks/install-systemd.md)
- [Desarrollo local](../runbooks/local-development.md)
