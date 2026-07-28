# `linux-mcp serve`

Levanta el servidor MCP. Abre **dos listeners**:

| Listener | Qué expone | Quién decide el acceso |
|----------|------------|------------------------|
| HTTP en `--addr` | El endpoint MCP (Streamable HTTP) | Token bearer emitido por `auth` |
| Socket unix en `--socket` | La emisión de tokens | Permisos del socket (grupo `--socket-group`) |

```bash
linux-mcp serve [flags]
```

## Flags

| Flag | Default | Descripción |
|------|---------|-------------|
| `--addr` | `127.0.0.1:5000` | Dirección del endpoint MCP. Mantenerla en loopback y llegar por túnel SSH |
| `--socket` | `/run/linux-mcp/issue.sock` | Ruta del socket de emisión. Flag persistente: `auth` usa el mismo |
| `--socket-group` | `mcp-admin` | Grupo dueño del socket. Con este valor el socket queda `0660`; vacío lo deja `0600` (solo el usuario del servicio) |
| `--max-ttl` | `24h` | Vida máxima que se concede a un token. Un `auth --ttl` mayor se recorta a este valor |
| `--cors` | *(vacío)* | Orígenes de browser permitidos, separados por coma. Vacío significa que ningún origen de browser puede conectar |

## Cómo funciona la autenticación

- La clave que firma los tokens se genera con `crypto/rand` al arrancar y vive **solo en memoria**. Reiniciar el servicio invalida todos los tokens emitidos.
- Los tokens son JWT HS256 con `iss`, `sub`, `uid`, `aud`, `exp`, `nbf`, `iat`, `jti` y `scope=mcp:read`. La `aud` es `http://<addr>`, así que un token emitido por otra instancia no sirve aquí.
- La identidad (`sub` y `uid`) la aporta el kernel vía `SO_PEERCRED` sobre el socket: el cliente no puede declararla.

## Origin y Host

Son dos controles distintos y ambos se aplican antes de la autenticación:

- **`Origin`**: solo aplica a clientes de browser. Una petición sin `Origin` (cualquier cliente que no sea un browser, incluido el proxy del MCP Inspector) no pasa por el allowlist. Una petición con `Origin` fuera del allowlist recibe `403` y el origen rechazado queda en el log del servicio.
- **`Host`**: siempre se valida contra el puerto de `--addr`, aceptando nombres loopback (`localhost`, `127.0.0.1`, `[::1]`) o exactamente el host de `--addr`. Esto corta el *DNS rebinding*, que el browser trata como same-origin y por lo tanto esquiva CORS.

Valores aceptados en `--cors`:

```bash
# Orígenes exactos, tal como los manda el browser (sin path ni barra final)
linux-mcp serve --cors 'http://localhost:6274,https://app.example.com'

# Comodín de puerto, solo para hosts loopback
linux-mcp serve --cors 'http://localhost:*,http://127.0.0.1:*'
```

No existe un valor que permita todos los orígenes. El comodín de host (`http://*`) y el comodín de puerto en un host público (`https://app.example.com:*`) se rechazan al arrancar.

## Errores de arranque

`serve` valida todo antes de abrir listeners y sale con código `1` escribiendo `error: <detalle>` en stderr.

| Mensaje | Causa | Solución |
|---------|-------|----------|
| `origin "..." must start with http:// or https://` | Valor de `--cors` sin esquema | Usar el origen completo tal como lo manda el browser |
| `origin "...": the host cannot be wildcarded...` | `--cors 'http://*'` | Listar los orígenes explícitamente |
| `origin "...": a wildcard port is only allowed for loopback hosts` | `--cors 'https://app.example.com:*'` | Indicar el puerto exacto |
| `listen address "..." must be host:port` | `--addr` sin puerto | Por ejemplo `127.0.0.1:5000` |
| `lookup group "mcp-admin": ...` | El grupo de `--socket-group` no existe | `sudo groupadd --system mcp-admin` o pasar `--socket-group ""` en local |
| `abstract socket "@..." is not allowed` | Ruta de `--socket` en el namespace abstracto | Usar una ruta de filesystem: los sockets abstractos no tienen permisos |
| `... exists and is not a socket` | La ruta de `--socket` está ocupada por un archivo normal | Elegir otra ruta o borrar el archivo a mano |
| `listen tcp ...: address already in use` | Otro proceso ocupa `--addr` | Liberar el puerto o cambiar `--addr` |

## Logs

El servicio emite auditoría en el log estándar:

```
INFO token issued jti=1c0Azh... uid=1002 sub=maria pid=48120 exp=2026-07-28T07:37:02-04:00
INFO mcp call method=tools/call tool=cat sub=maria jti=1c0Azh...
```

El `jti` es el mismo en la emisión y en cada uso, así que permite reconstruir qué hizo cada token.

## Ver también

- [`linux-mcp auth`](auth.md)
- [Instalar con systemd](../runbooks/install-systemd.md)
- [Desarrollo local](../runbooks/local-development.md)
