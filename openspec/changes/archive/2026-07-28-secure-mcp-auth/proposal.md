## Why

Hoy el servidor MCP no tiene autenticación: cualquier proceso que alcance `localhost:5000` puede invocar `cat` y `list`, que exponen buena parte del filesystem (`/proc`, `/sys`, configs). El acceso previsto es por túnel SSH desde equipos de varias personas, así que hace falta saber **quién** se conecta y poder cortar el acceso. Además `withCORS` es fail-open: `NewHandler()` no pasa orígenes, la validación se salta y el handler refleja cualquier `Origin`, de modo que cualquier página web visitada puede leer respuestas del MCP local.

El binario tampoco tiene subcomandos: `serve` y la emisión de credenciales necesitan ejecutarse por separado, con permisos distintos.

## What Changes

- Añadir CLI con cobra en `internal/command` (`root.go`, `serve.go`, `auth.go`); `cmd/linux-mcp/main.go` queda reducido a `command.Execute()`.
- Añadir canal de emisión de tokens: `serve` escucha un unix socket (`/run/linux-mcp/issue.sock`, `0660 mcp-agent:mcp-admin`); `linux-mcp auth` se conecta, recibe un JWT y lo imprime a stdout.
- Derivar la identidad del solicitante con `SO_PEERCRED` (uid del kernel), no de un flag del cliente; el `sub` del token no es configurable.
- Firmar JWT HS256 con clave aleatoria de 32 bytes generada en memoria al arrancar `serve`; la clave nunca toca disco y rota en cada reinicio.
- Exigir `Authorization: Bearer` en el endpoint MCP vía `auth.RequireBearerToken` del SDK, validando firma, `iss`, `aud`, `exp`/`nbf` y scope `mcp:read`.
- Registrar auditoría en journald: línea de emisión (`jti`, uid, sub, exp) y línea de uso por request (`jti`, sub, tool, path), correlacionables por `jti`.
- Reemplazar `withCORS` variádico fail-open por config explícita fail-closed con flag `--cors` (orígenes separados por coma, default vacío), admitiendo comodín de puerto solo en hosts loopback y registrando en el log los orígenes rechazados; añadir validación del header `Host` contra rebinding DNS.
- Actualizar la unit systemd: `ExecStart` con subcomando `serve`, `RuntimeDirectory`, `SupplementaryGroups=mcp-admin`, `LimitCORE=0`, `ProtectProc=invisible`, `RestrictAddressFamilies`.
- Actualizar el runbook con el grupo `mcp-admin`, la emisión del token, el túnel SSH y la configuración del cliente MCP.
- Soportar desarrollo local sin grupo administrativo: `--socket-group` vacío deja el socket en `0600`, sin `chgrp`. No se añade ningún modo inseguro.
- Añadir tasks `dev` y `token` al `Taskfile.yml`, `args_bin` a `.air.toml` y `tmp/` a `.gitignore`, para el flujo air + token + MCP Inspector.
- Añadir documentación de integración por cliente en `docs/agents/` (Cursor, Claude, Codex, OpenCode) y un runbook de desarrollo local, ambos indexados desde el README.
- Añadir workflow de GitHub Actions con build, `go vet`, tests y validación estática de la unit con `systemd-analyze verify`, más el badge en el README.
- Publicar binarios `linux/amd64` y `linux/arm64` como artifacts en cada run, y como GitHub Release con `SHA256SUMS` al empujar un tag `v*`, de modo que se puedan descargar sin cuenta de GitHub.
- Añadir `--version` al binario, inyectado con `-ldflags -X`, para poder identificar un binario descargado.
- **BREAKING** (build): se eliminan los targets `darwin/*` y `windows/*`. `SO_PEERCRED` es específico de Linux, así que dejarían de compilar al añadir la emisión; el proyecto ya dependía de `/proc`, de paths de Linux y de systemd.
- **BREAKING** (operación): `ExecStart=/usr/local/bin/linux-mcp` deja de funcionar; debe ser `linux-mcp serve`.
- **BREAKING** (clientes): las requests MCP sin `Bearer` válido devuelven 401; los orígenes de browser fuera del allowlist devuelven 403.

## Non-goals

- OAuth 2.1 completo (authorization server, Dynamic Client Registration, Protected Resource Metadata). Publicar PRM a medias haría que los clientes MCP intenten un flujo que no existe y fallen; el token es pre-emitido out-of-band.
- Revocación de tokens individuales y listado de tokens activos. El control es TTL corto más reinicio del servicio, que rota la clave e invalida todo.
- Persistencia de la clave de firma entre reinicios.
- Scopes por tool. Un único scope fijo `mcp:read`.
- TLS. El transporte sigue siendo HTTP sobre loopback; la confidencialidad la da el túnel SSH.
- Emisión de tokens en remoto. `linux-mcp auth` se ejecuta siempre en el host del servicio.

## Capabilities

### New Capabilities

- `token-issuance`: Canal de emisión por unix socket con control de acceso DAC, identidad desde `SO_PEERCRED`, firma JWT con clave efímera en memoria, tope de TTL y auditoría de emisión.
- `bearer-auth`: Validación de `Authorization: Bearer` en cada request MCP (firma, claims, audiencia, expiración, scope) y respuestas 401/403 conformes al spec MCP.
- `cli-commands`: CLI cobra con `serve` y `auth`, sus flags y la disciplina de salida de `auth` (token en stdout, resto en stderr).
- `http-origin-policy`: Política fail-closed de `Origin` configurable por `--cors` y validación de `Host` contra rebinding DNS.
- `agent-integration-docs`: Documento por cliente MCP (Cursor, Claude, Codex, OpenCode) con configuración copiable y token, más el runbook de desarrollo local, indexados desde el README.
- `ci-pipeline`: Workflow de GitHub Actions con build, vet, tests y validación estática de la unit systemd, con badge en el README.

### Modified Capabilities

- `systemd-install`: La unit pasa a invocar el subcomando `serve`, crea el runtime directory del socket, gana el grupo suplementario `mcp-admin` y endurecimiento adicional; el runbook incorpora la creación del grupo y el flujo de obtención del token.

## Impact

- Código: nuevo `internal/command/` (root, serve, auth), nuevo paquete de auth (emisión + verificación + store de config), cambios en `internal/handler/server.go` (CORS, Host, middleware) y en `cmd/linux-mcp/main.go`.
- Dependencias: añadir `github.com/spf13/cobra` (directa). `github.com/go-jose/go-jose/v4` ya está en el módulo como indirect y pasa a directa. `syscall.GetsockoptUcred` es stdlib, no suma dependencia.
- Deploy: `deploy/systemd/linux-mcp.service`.
- Docs: `docs/commands/serve.md`, `docs/commands/auth.md`, `docs/agents/{cursor,claude,codex,opencode}.md` y `docs/runbooks/local-development.md` (nuevos), `docs/runbooks/install-systemd.md`, `docs/README.md`, `README.md`.
- Tooling local: `Taskfile.yml` (tasks `dev` y `token`), `.air.toml` (`args_bin`), `.gitignore` (`tmp/`).
- CI: `.github/workflows/ci.yml` y el workflow de release (nuevos). El repositorio no tenía pipeline.
- Build: `Taskfile.yml` pierde cuatro includes y se eliminan `taskfiles/{darwin,windows}_*.yml`; los targets de Linux inyectan la versión.
- Verificación: la mayor parte de la validación es automática en `go test`; queda manual solo el borde multiusuario, que necesita más de una cuenta del SO, y el smoke con clientes de terceros.
- Runtime: mismo transporte Streamable HTTP y mismas tools; cambia el contrato de acceso (401/403) y el arranque del binario.
- Operación: nuevo grupo OS `mcp-admin`; los tokens mueren en cada reinicio del servicio.
