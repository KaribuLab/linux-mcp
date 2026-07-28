## 1. CLI con cobra

- [x] 1.1 Añadir `github.com/spf13/cobra` al módulo y crear `internal/command/root.go` (rootCmd `linux-mcp`, flag persistente `--socket`, `Execute()` con código de salida); sin argumentos muestra ayuda y no abre listeners
- [x] 1.2 Crear `internal/command/serve.go` con flags `--addr` (default `127.0.0.1:5000`), `--socket-group`, `--max-ttl`, `--cors`; mover ahí el arranque actual del servidor
- [x] 1.3 Reducir `cmd/linux-mcp/main.go` a `os.Exit(command.Execute())` y verificar que `linux-mcp serve` sirve MCP igual que antes del change
- [x] 1.4 Actualizar `.air.toml` con `args_bin` invocando `serve` y el socket local (`./tmp/issue.sock`); añadir `tmp/` a `.gitignore`
- [x] 1.5 Añadir task `dev` (air) y la var `SOCKET: ./tmp/issue.sock` al `Taskfile.yml`
- [x] 1.6 Añadir `--version` al rootCmd, leyendo una variable de paquete inyectable con `-ldflags -X`, con valor por defecto para builds locales

## 2. Emisión de tokens

- [x] 2.1 Crear el listener del socket de emisión: `net.Listen("unix", path)` con `syscall.Umask(0o177)` alrededor, `chgrp` al grupo administrativo y limpieza del socket al parar; rechazar rutas de namespace abstracto (`@…`)
- [x] 2.2 Obtener `ucred` con `syscall.GetsockoptUcred` sobre la conexión aceptada y resolver uid → nombre de usuario; tests de la resolución y del rechazo de rutas abstractas
- [x] 2.3 Generar la clave HMAC de 32 bytes con `crypto/rand` al arrancar `serve`, mantenerla solo en memoria y no exponerla por ninguna API
- [x] 2.4 Emitir el JWT HS256 con claims `iss`, `sub`, uid numérico, `aud`, `exp`, `nbf`, `iat`, `jti` (128 bits) y `scope=mcp:read`; aplicar `min(ttl solicitado, --max-ttl)`
- [x] 2.5 Crear `internal/command/auth.go`: conecta al socket, pide el token con `--ttl`, imprime **solo** el token a stdout y el `exp` efectivo o los avisos a stderr; sin flag de subject
- [x] 2.6 Emitir la línea de auditoría de emisión (`jti`, uid, sub, pid, exp) al log del servicio
- [x] 2.7 Tests de emisión: TTL recortado por el tope, `exp` siempre presente, claims completos, subject ignorando cualquier valor enviado por el cliente
- [x] 2.8 Implementar `--socket-group` vacío como "sin `chgrp`", dejando el socket en `0600`; test de que el modo local no depende de ningún grupo
- [x] 2.9 Añadir task `token` al `Taskfile.yml` (`go run {{.PKG}} auth --socket {{.SOCKET}} --ttl {{.TTL}}`, con `TTL` por defecto `8h` y sobreescribible) y verificar el flujo `task dev` + `task token` + MCP Inspector

## 3. Validación del bearer

- [x] 3.1 Implementar el `auth.TokenVerifier`: verificar firma, `iss`, `aud`, `nbf` y `exp`; devolver `TokenInfo{UserID: sub, Expiration: exp, Scopes: ["mcp:read"]}`
- [x] 3.2 Cablear `auth.RequireBearerToken` en `NewHandler` con `Scopes: ["mcp:read"]` y sin `ResourceMetadataURL`, de modo que el `WWW-Authenticate` no anuncie metadata OAuth inexistente
- [x] 3.3 Añadir la línea de auditoría de uso (`jti`, sub, tool) por invocación autenticada
- [x] 3.4 Tests de validación: sin header → 401; token expirado → 401; `aud` incorrecta → 401; payload manipulado → 401; sin scope → 403; token válido → 200

## 4. Política de Origin y Host

- [x] 4.1 Reemplazar la firma variádica de `withCORS` por config explícita fail-closed; añadir `Authorization` a `Access-Control-Allow-Headers` y mantener `Vary: Origin`
- [x] 4.2 Parsear `--cors` como lista separada por comas y repetible, con default **vacío**; validar que los valores sean orígenes completos (`esquema://host:puerto`) y fallar al arrancar si no lo son; eliminar el comentario de `main.go` que atribuye el CORS al Inspector
- [x] 4.2b Soportar comodín de puerto en hosts loopback (`http://localhost:*`, `http://127.0.0.1:*`); rechazar al arrancar el comodín de host y el comodín de puerto en hosts no loopback
- [x] 4.2c Loggear el `Origin` recibido cuando se rechaza por allowlist, para poder integrar clientes nuevos sin adivinar
- [x] 4.3 Añadir la validación del header `Host` contra el host y puerto de escucha, independiente del allowlist de `Origin`
- [x] 4.4 Verificar el orden `withCORS ∘ RequireBearerToken ∘ mcpHandler` y que el preflight `OPTIONS` se responde sin token
- [x] 4.5 Tests: `Origin` desconocido → 403 sin `Access-Control-Allow-Origin` y con el origen en el log; `Origin` permitido → header eco; sin `Origin` → pasa; comodín de puerto loopback casa cualquier puerto y sigue rechazando hosts externos; comodín de host falla al arrancar; `Host` externo → 403; preflight sin token → éxito

## 5. Deploy y documentación

- [x] 5.1 Actualizar `deploy/systemd/linux-mcp.service`: `ExecStart` con `serve`, `RuntimeDirectory=linux-mcp`, `RuntimeDirectoryMode=0755`, `SupplementaryGroups=mcp-admin`, `LimitCORE=0`, `ProtectProc=invisible`, `RestrictAddressFamilies=AF_UNIX AF_INET`
- [x] 5.2 Actualizar `docs/runbooks/install-systemd.md`: creación del grupo `mcp-admin`, alta de operadores, aviso de re-login, verificación de permisos del socket, y aviso en el paso de update de instalar la unit nueva antes de reiniciar
- [x] 5.3 Añadir al runbook la sección de conexión: `linux-mcp auth`, túnel `ssh -L 5000:127.0.0.1:5000 host`, configuración del cliente MCP con `Authorization: Bearer`, y la advertencia de que reiniciar el servicio invalida los tokens
- [x] 5.4 Añadir al runbook la tabla de troubleshooting: `permission denied` en el socket por membresía sin refrescar, socket ausente por `RuntimeDirectory`, 401 por token vencido o servicio reiniciado, 403 por `Origin` o `Host`
- [x] 5.5 Crear `docs/commands/serve.md`: flags (`--addr`, `--socket`, `--socket-group`, `--cors`, `--max-ttl`) con sus defaults, qué expone cada listener, y los códigos de error de arranque (socket ocupado, grupo inexistente, origen mal formado)
- [x] 5.6 Crear `docs/commands/auth.md`: flags (`--socket`, `--ttl`), disciplina de salida (token en stdout, avisos y `exp` efectivo en stderr), y códigos de error (`permission denied` por grupo, socket ausente, servicio caído)
- [x] 5.7 Verificar contra la documentación vigente de cada cliente cómo se configura un servidor MCP por Streamable HTTP con header `Authorization`, y anotar los que necesiten un puente en vez de soporte nativo
- [x] 5.8 Crear `docs/agents/cursor.md` y `docs/agents/claude.md`: bloque de configuración copiable, URL del túnel, header con el token y qué hacer cuando el token vence
- [x] 5.9 Crear `docs/agents/codex.md` y `docs/agents/opencode.md` con el mismo formato que los anteriores
- [x] 5.10 Crear `docs/runbooks/local-development.md`: `task dev` + `task token` + MCP Inspector, socket en `tmp/`, por qué en local no hace falta el grupo `mcp-admin`, por qué normalmente no hace falta `--cors` y cómo usar `--cors 'http://localhost:*,http://127.0.0.1:*'` cuando un cliente de browser arranca en un puerto impredecible, y la distinción entre el token de sesión del proxy del Inspector y el bearer de `linux-mcp` que se pega en la barra lateral
- [x] 5.11 Actualizar `README.md` y `docs/README.md`: tabla de comandos enlazando `docs/commands/`, índice de agentes enlazando `docs/agents/`, enlace al runbook de desarrollo local, uso de `serve` y `auth` y requisito de token

## 6. Verificación automatizada

- [x] 6.1 Test end-to-end en Go: socket real en `t.TempDir()` + handler con `httptest`; emitir un token por el socket, usarlo contra el endpoint MCP y comprobar que la tool responde
- [x] 6.2 Test de la cadena de auditoría: capturar el `slog` en un buffer y verificar que la línea de emisión y la de uso comparten el mismo `jti`
- [x] 6.3 Test de identidad real: verificar que el `sub` del token emitido corresponde al uid del proceso de test según `SO_PEERCRED`, sin que el cliente lo declare
- [x] 6.4 Verificar la unit con `systemd-analyze verify deploy/systemd/linux-mcp.service`

## 7. CI y distribución de binarios

- [x] 7.1 Reducir el build a Linux: eliminar los includes y los archivos `taskfiles/{darwin,windows}_*.yml`, dejar `build:all` con `linux_amd64` y `linux_arm64`, e inyectar la versión con `-ldflags -X` en ambos targets
- [x] 7.2 Añadir `.github/workflows/ci.yml`: checkout, setup-go tomando la versión del `go.mod`, cache de módulos, y `go build ./...`, `go vet ./...` y `go test ./...`
- [x] 7.3 Añadir al workflow la validación estática de la unit con `systemd-analyze verify` (disponible en los runners Ubuntu)
- [x] 7.4 Publicar los binarios `linux/amd64` y `linux/arm64` como artifacts en cada run, para poder probar el binario de un pull request
- [x] 7.5 Añadir el workflow de release disparado por tags `v*`: compilar ambas arquitecturas con la versión del tag, generar `SHA256SUMS` y adjuntar todo a un GitHub Release
- [x] 7.6 Verificar que un binario descargado del release reporta la versión del tag con `linux-mcp --version` y que su checksum coincide con `SHA256SUMS`
- [x] 7.7 Añadir al `README.md` el badge de estado y una sección de descarga que apunte a los releases y explique cómo verificar el checksum

## 8. Verificación manual

Solo lo que no aporta valor automatizar: software de terceros y el borde multiusuario, que necesita más de una cuenta del SO.

- [ ] 8.1 Smoke con MCP Inspector: token válido, token vencido, y comprobar que conecta sin configurar `--cors`
- [ ] 8.2 Smoke con cliente MCP por túnel SSH, con token válido y sin token
- [ ] 8.3 En un host con el servicio instalado, verificar que un usuario fuera de `mcp-admin` recibe `permission denied` al ejecutar `linux-mcp auth`, y que uno dentro del grupo obtiene token
- [ ] 8.4 En ese mismo host, verificar que el socket queda `0660 mcp-agent:mcp-admin` y que reiniciar el servicio invalida los tokens previos
