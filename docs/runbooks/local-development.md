# Desarrollo local

Cómo levantar linux-mcp en tu máquina, emitir un token y probarlo con MCP Inspector, sin crear usuarios ni grupos del sistema.

## Flujo corto

```bash
# terminal 1: servidor con recarga
task dev

# terminal 2: token para probar
task token
```

`task dev` corre `air`, que arranca `linux-mcp serve --socket ./tmp/issue.sock --socket-group ""`. El socket queda dentro del repo, en `tmp/`, que está ignorado por git y excluido del watcher.

`task token` ejecuta `linux-mcp auth` contra ese mismo socket. Acepta un TTL más corto para probar la expiración:

```bash
task token TTL=2m
```

## Por qué en local no hace falta el grupo `mcp-admin`

Quien decide si puedes pedir un token es el permiso del socket, no el código. Con `--socket-group ""` el socket queda en `0600` y pertenece a tu usuario: eres el único que puede pedir tokens, que es exactamente lo que quieres en tu máquina. En el servicio con systemd el socket queda `0660 mcp-agent:mcp-admin`, y ahí sí hace falta el grupo.

No hay un modo "inseguro" ni una forma de saltarse la autenticación: el flujo local es el mismo que en producción, solo cambia quién es dueño del socket.

## MCP Inspector

```bash
npx @modelcontextprotocol/inspector
```

1. Transporte: **Streamable HTTP**
2. URL: `http://localhost:5000`
3. En la barra lateral, **Authentication**: pega el token de `task token` como *Bearer Token*
4. Connect

### Los dos tokens del Inspector no son el mismo

Es fácil confundirlos:

| Token | Para qué | De dónde sale |
|-------|----------|---------------|
| *Session token* / proxy token | Autentica el UI del Inspector contra su propio proceso proxy | Lo imprime el Inspector al arrancar |
| *Bearer token* | Autentica contra linux-mcp | `task token` |

### Por qué normalmente no necesitas `--cors`

El Inspector no habla directamente desde el browser con tu servidor: el UI habla con un proxy Node que corre en tu máquina, y ese proxy es quien llama a linux-mcp. Como no es un browser, no manda el header `Origin`, y la política de orígenes no llega a aplicarse. Por eso el Inspector conecta con la configuración por defecto, que no permite ningún origen de browser.

Si en cambio pruebas con un cliente que sí corre en el browser y llega directo al servidor, verás `403 origin not allowed` y el log del servidor te dirá el origen exacto:

```
WARN origin rejected origin=http://localhost:41235
```

Con eso puedes autorizarlo. Cuando el puerto del cliente cambia en cada arranque, usa el comodín de puerto, que solo se acepta para hosts loopback:

```bash
go run ./cmd/linux-mcp serve --socket ./tmp/issue.sock --socket-group "" \
  --cors 'http://localhost:*,http://127.0.0.1:*'
```

Para dejarlo fijo en el flujo de `air`, agrega esos argumentos a `args_bin` en `.air.toml`.

## Probar sin cliente MCP

El endpoint responde `401` sin token, que es la señal de que está arriba:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' http://localhost:5000
# 401
```

## Cosas que conviene recordar

- Cada vez que `air` recompila y reinicia el servidor, la clave de firma se regenera: **el token anterior deja de servir**. Si empiezas a ver `401` después de guardar un archivo, pide otro con `task token`.
- El `Host` se valida contra el puerto de `--addr`. Usa `localhost:5000` o `127.0.0.1:5000`; un proxy que reescriba `Host` recibirá `403`.
- Si `task dev` falla con `... exists and is not a socket`, borra el archivo que quedó en `tmp/issue.sock`.

## Ver también

- [`linux-mcp serve`](../commands/serve.md)
- [`linux-mcp auth`](../commands/auth.md)
- [Instalar con systemd](install-systemd.md)
