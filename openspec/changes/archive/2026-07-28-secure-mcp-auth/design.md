## Context

`linux-mcp` expone `cat` y `list` por Streamable HTTP en `localhost:5000`, sin autenticación. El acceso real es por túnel SSH (`ssh -L 5000:127.0.0.1:5000 host`) desde los equipos de varias personas, y `linux-mcp auth` se ejecuta **siempre en el host del servicio**, nunca desde el equipo cliente.

Estado actual relevante:

- `cmd/linux-mcp/main.go` llama directo a `http.ListenAndServe("localhost:5000", handler)`; no hay subcomandos.
- `internal/handler.NewHandler()` invoca `withCORS(h)` sin orígenes; como el parámetro es variádico, `len(allowedOrigins) == 0` y la validación entera se salta. El handler refleja cualquier `Origin`.
- `Access-Control-Allow-Headers` no incluye `Authorization`.
- La unit systemd corre como `mcp-agent` con `ProtectSystem=strict`, `ProtectHome=read-only` y `CAP_DAC_READ_SEARCH`.

El SDK `github.com/modelcontextprotocol/go-sdk v1.6.1` ya trae `auth.RequireBearerToken(verifier, opts)`: extrae el bearer, delega en un `TokenVerifier`, exige `TokenInfo.Expiration` no nula ni vencida, valida scopes y responde 401 con `WWW-Authenticate`. Se usa tal cual; no se reimplementa.

## Goals / Non-Goals

**Goals:**

- Que solo miembros de un grupo OS puedan obtener credenciales, con el control aplicado por el kernel y no por lógica de la aplicación.
- Identidad del solicitante infalsificable, para que la auditoría sirva.
- Cero material criptográfico en disco.
- Cumplir la parte del spec MCP que los clientes realmente consumen: `Authorization: Bearer`, 401 con `WWW-Authenticate`, validación de audiencia y expiración.
- Cerrar el CORS fail-open y cubrir rebinding DNS.
- Separar `serve` de `auth` en subcomandos.

**Non-Goals:**

- OAuth 2.1 completo, PRM, DCR.
- Revocación individual, listado de tokens, persistencia de la clave.
- Scopes por tool. TLS. Emisión remota.

## Decisions

### D1: Canal de emisión = unix socket; el control de acceso es DAC

- **Qué:** `serve` crea `/run/linux-mcp/issue.sock` con permisos `0660 mcp-agent:mcp-admin`. `linux-mcp auth` se conecta ahí.
- **Por qué:** conectarse a un unix socket requiere permiso de **escritura** sobre el archivo, así que la pertenencia a `mcp-admin` la verifica el kernel al abrir. No hay lógica de autorización que auditar ni que rodear.
- **Por qué no restringir el binario:** `auth` es solo un cliente; cualquiera podría escribir el suyo. El único control no rodeable es el permiso del socket.
- **Alternativa rechazada — endpoint TCP admin + uid vía `/proc/net/tcp`:** posible (la columna `uid` está ahí, y a través de `ssh -L` sshd conecta como el usuario autenticado), pero exige parsear `/proc`, leer la entrada del socket **del cliente** y no la del servidor, y deja una ventana TOCTOU al reusarse el puerto efímero. Innecesario porque `auth` corre en el host.
- **Prohibido:** sockets abstractos (`@nombre`). No tienen chequeo de permisos: cualquier proceso del namespace de red conectaría.
- **Creación sin ventana de permisos:** aplicar `syscall.Umask(0o177)` alrededor del `net.Listen` en vez de `chmod` posterior, que dejaría un instante con el socket más abierto de lo debido.

### D2: La identidad viene de `SO_PEERCRED`, no del cliente

- **Qué:** al aceptar la conexión, `serve` obtiene `ucred{Pid, Uid, Gid}` con `getsockopt(fd, SOL_SOCKET, SO_PEERCRED)` y deriva el `sub` del uid. `auth` **no** tiene flag `--sub`.
- **Por qué:** el kernel anota esas credenciales en la conexión al hacer `connect()`; el cliente no las envía y no puede falsificarlas.
- **Por qué no validar un `--sub` contra `/etc/passwd`:** no cierra nada. Si María pasa `--sub patricio`, el chequeo aprueba porque `patricio` es un usuario válido. La propiedad necesaria no es "existe" sino "es quien abrió esta conexión".
- **No usar `ucred.Gid` para el chequeo de grupo:** devuelve el grupo **primario**. Un usuario que sea miembro suplementario de `mcp-admin` fallaría la comparación. El chequeo de grupo ya lo hace D1 por permisos del socket; `SO_PEERCRED` se usa para identidad y auditoría.
- **Claims:** `sub` con el nombre de usuario (legible) y un claim privado con el uid numérico (estable ante reciclado de nombres por `userdel`/`useradd`). Ambos al log.

### D3: JWT HS256 con clave efímera en memoria

- **Qué:** `serve` genera 32 bytes con `crypto/rand` al arrancar y firma con HS256. La clave no se escribe a disco ni se deriva de nada persistente.
- **Por qué simétrico:** el mismo proceso firma y verifica. Ed25519 solo aporta cuando el verificador no debe poder firmar, que no es el caso.
- **Consecuencia aceptada:** reiniciar `serve` invalida todos los tokens. Es la rotación y la revocación de emergencia del sistema.
- **Camino de salida:** si algún día la emisión sale del proceso, se migra a EdDSA con clave privada solo para el emisor.
- **Librería:** `go-jose/v4`, ya presente en el módulo como indirect.

### D4: Claims y validación

Emisión:

| Claim | Valor |
|-------|-------|
| `iss` | `linux-mcp` |
| `sub` | nombre de usuario derivado del uid del peer |
| `aud` | URL del recurso, p. ej. `http://127.0.0.1:5000` |
| `exp` | ahora + TTL efectivo |
| `nbf`, `iat` | ahora |
| `jti` | 128 bits aleatorios |
| `scope` | `mcp:read` |

El `TokenVerifier` valida firma, `iss`, `aud`, `nbf` y `exp`, y devuelve `TokenInfo{UserID: sub, Expiration: exp, Scopes: ["mcp:read"]}`.

- **`aud` es obligatorio:** el spec MCP exige que el servidor rechace tokens que no fueron emitidos para él.
- **`UserID = sub` no es decorativo:** el transporte Streamable HTTP del SDK lo usa para impedir que una sesión MCP sea continuada por otro usuario.
- **Scope único `mcp:read`:** `RequireBearerTokenOptions.Scopes` lo exige, así que un token sin él da 403, y se anuncia en el `WWW-Authenticate` del 401.

### D5: El TTL lo pide el cliente y lo acota el servidor

- `auth --ttl 8h` es una petición; `serve --max-ttl 24h` es el techo. El efectivo es `min(ttl, max-ttl)`.
- **Por qué:** sin techo, alguien pide un año y la expiración deja de ser un control.
- `auth` informa por stderr el `exp` efectivo cuando hubo recorte.

### D6: Orden del middleware

```
withCORS  ∘  RequireBearerToken  ∘  mcpHandler
```

- **`withCORS` va por fuera y corta el `OPTIONS` con 204 antes de tocar auth.** El preflight del browser no lleva header `Authorization`; si el orden se invierte, el preflight recibe 401 y el MCP Inspector nunca llega a conectar.
- `Access-Control-Allow-Headers` **debe** incluir `Authorization`, o el preflight rechaza el header y el Inspector no puede autenticarse. Esto no es endurecimiento: sin ello la feature no funciona en browser.

### D7: CORS fail-closed + validación de `Host`

Son dos controles para dos ataques distintos.

```
Host ∉ {127.0.0.1:5000, localhost:5000}   → 403          (rebinding DNS)
Origin ausente                             → pasa         (Cursor, curl: no son browsers)
Origin ∈ allowlist                         → pasa + ACAO: <origin> + Vary: Origin
Origin ∉ allowlist                         → 403, sin ACAO
```

- **Por qué el chequeo de `Host` y no solo `Origin`:** en rebinding DNS el atacante hace que su dominio resuelva a `127.0.0.1`; el browser considera la petición same-origin y **CORS no se evalúa**. Lo que delata el ataque es el `Host: evil.com`.
- **Por qué `Origin` ausente pasa:** CORS es una política que aplica el browser. Un cliente que no es browser nunca manda `Origin` y no está sujeto a ella. A los clientes no-browser los protege el Bearer, no el CORS.
- **Flag:** `--cors` con orígenes separados por coma. `StringSliceVar` de cobra ya parte por comas y admite repetir el flag.
- **Son orígenes, no dominios:** `esquema://host:puerto`. `localhost:6274` no matchea nunca; `http://localhost:6274` y `http://127.0.0.1:6274` son orígenes distintos.
- **Default: vacío.** Ningún origen de browser permitido salvo que se configure explícitamente.
- **Por qué vacío y no los orígenes del Inspector:** el MCP Inspector no conecta al servidor desde el browser. Su UI (`6274`) habla con su propio proxy Node (`6277`), y es el proxy el que abre la conexión a `linux-mcp`. Esa petición sale de Node y no lleva `Origin`, así que pasa sin evaluar CORS. Preconfigurar orígenes de browser concedería acceso para un caso que no ocurre, contradiciendo el fail-closed. El comentario actual en `main.go` que atribuye el CORS a "MCP Inspector runs in the browser" describe un supuesto que no se sostiene.
- **`--cors` sigue existiendo** para clientes que sí conecten directamente desde un browser.
- **Comodín solo en el puerto y solo en loopback:** se acepta `http://localhost:*` y `http://127.0.0.1:*`, que casan cualquier puerto de ese host. Resuelve el caso real de desarrollo, donde lo que varía es el puerto del cliente (Inspector, extensiones) y no el host.
- **Sin comodín global:** no se soporta `--cors '*'`. Con `*`, cualquier página visitada puede hacer fetch a `127.0.0.1:5000` y leer la respuesta; el chequeo de `Host` no lo impide, porque esa petición envía un `Host` legítimo y solo detiene rebinding DNS. El comodín de puerto restringe el permiso a páginas servidas desde la propia máquina.
- **El `Origin` rechazado se registra en el log.** Al responder 403 por origen, el servidor MUST loggear el valor recibido, para que el operador sepa qué añadir al allowlist en vez de adivinar o abrirlo todo. Es la alternativa práctica a `*` cuando se integra un cliente nuevo cuyo origen se desconoce.
- **Firma no variádica:** la config de orígenes pasa a ser explícita, para que omitirla sea error de compilación y no un agujero silencioso como hoy.

### D8: Estructura CLI

```
cmd/linux-mcp/main.go        os.Exit(command.Execute())

internal/command/
  root.go    rootCmd "linux-mcp", flag persistente --socket, Execute()
  serve.go   serveCmd
  auth.go    authCmd
```

```
serve  --socket /run/linux-mcp/issue.sock  --socket-group mcp-admin
       --addr 127.0.0.1:5000  --cors <lista>  --max-ttl 24h
auth   --socket (heredado)  --ttl 8h
```

- `--socket` es persistente en `root` porque es el único punto de encuentro entre los dos procesos.
- `--addr` con `127.0.0.1` explícito en vez de `localhost`, que puede resolver también a `::1`.
- `linux-mcp` sin argumentos muestra la ayuda; **no** arranca el servidor. Arrancar un servidor de red por omisión es un comportamiento sorpresivo.
- **Disciplina de salida de `auth`:** el token va solo a stdout; avisos, `exp` efectivo y errores a stderr. Así `TOKEN=$(linux-mcp auth --ttl 8h)` funciona en scripts.

### D9: Auditoría a journald, no estado en el servidor

```
emisión:  jti=a3f9…  uid=1002  sub=maria  exp=…  pid=48122
uso:      jti=a3f9…  sub=maria  tool=cat  path=/etc/nginx/nginx.conf
```

- El `jti` correlaciona ambas líneas y reconstruye la cadena completa: quién pidió el token, cuándo y qué hizo con él.
- Son líneas de log, no estado consultado para autorizar. No rompe D3 ni obliga a mantener tabla en memoria.

### D10: Cambios en la unit systemd

```ini
ExecStart=/usr/local/bin/linux-mcp serve      # antes: sin subcomando
RuntimeDirectory=linux-mcp                    # crea /run/linux-mcp; ProtectSystem=strict lo impediría si no
RuntimeDirectoryMode=0755                     # traverse para miembros de mcp-admin
SupplementaryGroups=mcp-admin                 # para que mcp-agent pueda chgrp el socket
LimitCORE=0                                   # un core dump filtraría la clave HMAC a disco
ProtectProc=invisible
RestrictAddressFamilies=AF_UNIX AF_INET
```

- `LimitCORE=0` pasa a ser relevante justo ahora: con D3 hay material de clave en RAM.
- `RestrictAddressFamilies` es viable porque D1 descartó el endpoint TCP admin, que habría necesitado netlink o `/proc/net`.
- **Alternativa considerada — unit `.socket` con `SocketGroup=mcp-admin` y `SocketMode=0660`:** systemd crearía el socket ya con los permisos correctos y evitaría `SupplementaryGroups`. Se descarta en v1 porque obliga a manejar `LISTEN_FDS` en el código y a mantener dos units.

### D11: Desarrollo local sin modo inseguro

En local `serve` corre como el propio desarrollador, así que el socket es de su propiedad y puede conectarse sin pertenecer a ningún grupo. El grupo solo existe porque en producción `serve` corre como `mcp-agent`, un usuario distinto del operador.

- **`--socket-group` vacío ⇒ no se hace `chgrp`** y el socket queda `0600`. Es **más** restrictivo que producción, no menos: solo el dueño emite tokens.
- **No se añade un flag `--insecure` ni un modo dev.** Una ruta de código que relaje la autorización solo existiría para desarrollo y sería la primera candidata a filtrarse a producción por un default mal puesto.
- **Socket local en `tmp/`:** `/run/linux-mcp` no es escribible por un usuario normal. `tmp/` ya está en `exclude_dir` de `.air.toml`, así que crear el socket ahí no dispara rebuilds en bucle; en un directorio vigilado sí lo haría. `tmp/` debe entrar a `.gitignore`.
- **`.air.toml` necesita `args_bin`:** hoy está vacío y ejecuta el binario sin argumentos, lo que con cobra imprime la ayuda y termina. Debe invocar `serve` con el socket local.
- **Flujo objetivo:** `task dev` en una terminal, `task token` en otra, y el token pegado en el MCP Inspector.

### D12: Documentación de integración por cliente

Cada agente configura servidores MCP de forma distinta y no todos soportan Streamable HTTP con headers propios. Se documenta uno por archivo en `docs/agents/<cliente>.md` (Cursor, Claude, Codex, OpenCode) en vez de una sola página comparativa, para que cada operador copie solo su bloque de configuración.

- El soporte de header `Authorization` por cliente MUST verificarse contra la documentación vigente al escribir cada archivo; si algún cliente no lo soporta de forma nativa, el documento debe indicar el puente necesario en vez de asumir que funciona.
- El flujo de desarrollo local vive aparte, en `docs/runbooks/local-development.md`, porque su audiencia es quien modifica el repo y no quien lo instala.

### D13: Estrategia de verificación

La mayor parte de lo que hay que verificar es automatizable sin contenedores ni privilegios, y estaba mal clasificado como smoke manual.

- **`go test` cubre casi todo:** un unix socket en `t.TempDir()` funciona en un test normal, así que la emisión completa se prueba de verdad. `SO_PEERCRED` incluso se verifica de forma real, porque el test corre como el usuario que lo lanza y el kernel reporta su uid: se comprueba que el `sub` sale del kernel y no de un parámetro. El lado HTTP va con `httptest`. La cadena de auditoría se verifica capturando el `slog` en un buffer y comprobando que emisión y uso comparten `jti`, en vez de leer `journalctl` a mano.
- **Sin testcontainers.** Lo único que no se puede probar como un solo usuario es el borde multiusuario (`permission denied` para quien no está en `mcp-admin`, y el `chgrp`). Es un escenario, y traer testcontainers-go implicaría además exponer la API Docker de podman. Queda como verificación manual sobre un host con el servicio instalado.
- **La unit se valida estáticamente** con `systemd-analyze verify`, que atrapa directivas inválidas sin provisionar una máquina. Probar systemd de verdad (que `RuntimeDirectory` se cree, que `SupplementaryGroups` aplique) exigiría un contenedor con systemd o una VM, y no compensa.
- **Queda manual solo lo que es software de terceros:** el MCP Inspector y el túnel SSH. El túnel es forwarding TCP transparente y nuestro código no lo ve, así que automatizarlo verificaría `ssh`, no `linux-mcp`.
- **CI:** el repositorio no tenía pipeline. Se añade uno con build, `vet`, tests y `systemd-analyze verify`, tomando la versión de Go del `go.mod` en vez de fijarla, para que no quede desalineada al actualizar el módulo.

### D14: Distribución de binarios y recorte a Linux

- **Solo Linux.** `SO_PEERCRED` no existe en macOS (usa `LOCAL_PEERCRED`/`getpeereid`) ni en Windows, así que los targets `darwin/*` y `windows/*` dejarían de compilar al entrar la emisión. El proyecto ya era Linux-only de facto: lee `/proc`, la denylist son paths de Linux y el deploy depende de `CAP_DAC_READ_SEARCH` y systemd. Se eliminan en vez de sostenerlos con build tags y stubs que fallarían en runtime, porque un binario que compila pero no puede emitir tokens es peor que no tenerlo.
- **Release en tag, no artifacts, para distribuir.** Los artifacts de Actions exigen sesión de GitHub para descargarse y expiran; un Release da URL pública y estable. Los artifacts se mantienen igualmente, pero para probar el binario de un pull request antes de mergear, que es otro problema.
- **`SHA256SUMS` en el release.** Distribuir binarios sin checksum obliga al usuario a confiar en la descarga sin poder verificarla.
- **`--version` inyectado con `-ldflags -X`,** con fallback para builds locales. Sin esto, un binario descargado es indistinguible de otro y el troubleshooting se vuelve adivinanza.

## Risks / Trade-offs

| Riesgo | Mitigación |
|--------|------------|
| Reiniciar el servicio invalida todos los tokens | Aceptado y documentado; es también la revocación de emergencia |
| `usermod -aG mcp-admin` no toma efecto hasta reabrir sesión | Fila propia en troubleshooting del runbook; es el primer fallo que se va a ver |
| **BREAKING**: `ExecStart` sin subcomando deja el servicio mostrando el help y saliendo | Unit y runbook se actualizan en el mismo change; nota explícita en el paso de update |
| `mcp-agent` queda en `mcp-admin` y podría emitirse tokens | Sin impacto: es el propio servidor, ya tiene todo el acceso |
| Volcado de memoria del proceso expone la clave HMAC | `LimitCORE=0`, `ProtectProc=invisible`; contra un atacante con root en el host no hay defensa real y no se pretende |
| Token pegado en la config del cliente queda en texto plano | TTL corto; `jti` en logs permite rastrear el uso |
| Sin revocación individual | Decisión explícita; TTL + reinicio. El `jti` ya está en el token si se quiere añadir denylist después |
| Publicar PRM haría fallar clientes | Non-goal: 401 con `WWW-Authenticate: Bearer` desnudo, sin `resource_metadata` |

## Migration Plan

1. Añadir cobra y `internal/command`; `serve` reproduce el comportamiento actual. Verificar que el binario sigue sirviendo MCP.
2. Añadir emisión (socket + `SO_PEERCRED` + firma) y `auth`, todavía sin exigir el token en el endpoint MCP.
3. Activar `RequireBearerToken` en el handler. A partir de aquí el acceso sin token da 401.
4. Reemplazar `withCORS` por la versión fail-closed con `--cors` y chequeo de `Host`.
5. Actualizar unit, runbook y README. Documentar el paso de update: instalar unit nueva **antes** de reiniciar, o el servicio no arranca.
6. Verificar con MCP Inspector, que llega por su proxy y por tanto sin `Origin`, y con Cursor por túnel SSH. Dejar el borde multiusuario para un host con el servicio instalado.
7. Rollback: revertir el change y restaurar `ExecStart` sin subcomando. No hay datos que migrar.

## Open Questions

Ninguna abierta. Cerrado en exploración:

- **Canal de emisión:** unix socket, no endpoint TCP. `auth` corre siempre en el host.
- **Identidad:** `sub` derivado de `SO_PEERCRED`; sin flag `--sub`.
- **Formato del token:** JWT firmado con expiración, HS256, clave efímera en RAM.
- **Autorización de emisión:** grupo `mcp-admin` por permisos del socket.
- **Revocación:** fuera de alcance; TTL más reinicio.
- **Scopes:** uno fijo, `mcp:read`.
- **CORS:** flag `--cors` separado por comas, fail-closed, sin comodín.
