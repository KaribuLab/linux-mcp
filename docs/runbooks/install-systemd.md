# Instalar linux-mcp con systemd

Runbook para desplegar el servidor MCP como servicio systemd con usuario `mcp-agent`, capacidades `CAP_DAC_READ_SEARCH` + `CAP_SYS_PTRACE`, y emisión de tokens restringida al grupo `mcp-admin`.

> **Forma recomendada: one-liner.** Si preferís un único comando en vez de los pasos manuales de abajo, usá [`docs/runbooks/install-one-line.md`](install-one-line.md). El script `install.sh` (raíz del repo) automatiza todo este runbook de forma idempotente y es POSIX `sh`. Los pasos manuales de aquí siguen siendo la fuente de verdad para entender qué hace cada comando y para hosts donde pipe-to-sh no es opción.

La **policy de lectura en el proceso Go es obligatoria** aunque no uses systemd: la unit solo añade defensa en profundidad (`InaccessiblePaths`, hardening de escritura). El binario puede correr a mano sin esta unit.

### Capacidades de la unit y riesgo Bearer

| Cap / directiva | Para qué |
|-----------------|----------|
| `CAP_DAC_READ_SEARCH` | Lectura ops de archivos más allá del DAC del uid `mcp-agent` |
| `CAP_SYS_PTRACE` | Resolver inode de socket → Pid/Process vía `/proc/*/fd` de procesos de otros usuarios (como `ss -p`) |
| Sin `ProtectProc=invisible` | El servicio debe ver entradas `/proc` de otros uids para `ss`/`ps` |

Un Bearer válido puede obtener ese inventario de dueños de sockets a través de las tools MCP (solo lectura; no hay tool de volcado de memoria). Asumí ese riesgo si instalás la unit de referencia: acotá `mcp-admin`, TTL cortos y bind en loopback.

Dos identidades participan:

| Identidad | Para qué |
|-----------|----------|
| `mcp-agent` (usuario) | Corre el servicio y es dueño del socket de emisión |
| `mcp-admin` (grupo) | Lista de operadores autorizados a ejecutar `linux-mcp auth` |

## Requisitos

- Linux con systemd (`amd64` o `arm64`)
- Go 1.26+ solo si vas a compilar; para instalar desde Releases no hace falta
- Privilegios root para crear usuario y grupo, instalar unit y capability ambient

## 1. Crear usuario y grupo

```bash
sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin mcp-agent
sudo groupadd --system mcp-admin
```

El grupo `mcp-admin` debe existir **antes** de arrancar el servicio: la unit lo declara en `SupplementaryGroups` y `serve` falla al arrancar si no puede resolverlo.

## 2. Obtener e instalar el binario

### 2.1 Descargar un binario publicado (recomendado)

Los Releases los genera el CI al mergear a `main`: [kli](https://github.com/KaribuLab/kli) calcula el semver desde Conventional Commits y publica binarios `linux/amd64` y `linux/arm64` con `SHA256SUMS` en [Releases](https://github.com/KaribuLab/linux-mcp/releases). Solo Linux. Si aún no hay release, usá [§2.2](#22-compilar-desde-el-repo).

Elegí un tag que exista en Releases (no inventes el número) y el asset según tu arquitectura (`uname -m`: `x86_64` → `amd64`, `aarch64` → `arm64`). `-f` hace que `curl` falle si el asset no existe (sin `-f`, un 404 deja HTML en el archivo y `sha256sum` se queja del formato):

```bash
VERSION=v0.10.0   # última en https://github.com/KaribuLab/linux-mcp/releases
ARCH=amd64        # o arm64
BASE=https://github.com/KaribuLab/linux-mcp/releases/download/$VERSION
curl -fsSLO $BASE/linux-mcp-linux-$ARCH
curl -fsSLO $BASE/SHA256SUMS

sha256sum --ignore-missing -c SHA256SUMS
sudo install -m 0755 linux-mcp-linux-$ARCH /usr/local/bin/linux-mcp
linux-mcp --version
```

### 2.2 Compilar desde el repo

Desde el repo:

```bash
go tool task build
sudo install -m 0755 dist/linux-mcp-$(go env GOOS)-$(go env GOARCH) /usr/local/bin/linux-mcp
```

Sin Task:

```bash
go build -o /tmp/linux-mcp ./cmd/linux-mcp
sudo install -m 0755 /tmp/linux-mcp /usr/local/bin/linux-mcp
```

## 3. Instalar y habilitar la unit

Si tenés el repo clonado:

```bash
sudo install -m 0644 deploy/systemd/linux-mcp.service /etc/systemd/system/linux-mcp.service
```

Sin clonar (`-f` falla ante 404). La unit siempre sale de `main` (no del tag del binario): es config de deploy, no un asset versionado del Release.

```bash
curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/deploy/systemd/linux-mcp.service \
  -o /tmp/linux-mcp.service
sudo install -m 0644 /tmp/linux-mcp.service /etc/systemd/system/linux-mcp.service
```

Luego:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now linux-mcp.service
```

## 4. Verificar

```bash
systemctl status linux-mcp.service
journalctl -u linux-mcp.service -n 50 --no-pager
```

El endpoint MCP responde `401` sin token, que es justamente la señal de que está vivo y protegido:

```bash
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:5000
# 401
```

El socket de emisión debe quedar así:

```bash
sudo ls -l /run/linux-mcp/issue.sock
# srw-rw---- 1 mcp-agent mcp-admin 0 ... /run/linux-mcp/issue.sock
```

Si el modo no es `0660` o el grupo no es `mcp-admin`, ningún operador podrá pedir tokens (o los podrá pedir de más).

Comprobá que la unit cargó las caps y no oculta `/proc` ajeno:

```bash
systemctl show linux-mcp.service -p AmbientCapabilities -p CapabilityBoundingSet -p ProtectProc
# AmbientCapabilities=cap_dac_read_search cap_sys_ptrace  (orden/nombre pueden variar)
# ProtectProc=default   # o vacío / no "invisible"
```

Con un token y cliente MCP (o Inspector), listá listeners y confirmá Pid de un proceso que **no** sea `mcp-agent` (p. ej. `sshd`, MTA, web):

```text
Usa el tool linux-mcp `ss` con state=LISTEN y showPid/showProcess en true.
Para un puerto conocido de otro servicio, el Pid no debe quedar vacío.
```

Si Pid sigue vacío en listeners ajenos: unit vieja en `/etc` (falta [§7](#7-actualizar)), falta `CAP_SYS_PTRACE`, o `ProtectProc=invisible` en un drop-in.

## 5. Autorizar operadores

```bash
sudo usermod -aG mcp-admin maria
```

La membresía de grupo se resuelve al iniciar sesión: **maria debe cerrar sesión y volver a entrar** (o abrir una sesión SSH nueva) para que el cambio tenga efecto. `id -nG` en la sesión vieja seguirá mostrando la lista anterior.

Para revocar el acceso a emitir tokens:

```bash
sudo gpasswd -d maria mcp-admin
```

Los tokens que maria ya tenía siguen siendo válidos hasta que expiren o hasta que se reinicie el servicio.

## 6. Conectarse desde un cliente

### 6.1 Obtener el token

En el host donde corre el servicio, como usuario del grupo `mcp-admin`:

```bash
TOKEN=$(linux-mcp auth --ttl 8h)
```

El token sale por stdout y todo lo demás (subject y expiración) por stderr, así que la captura anterior deja solo el token en la variable. El token identifica a quien ejecutó el comando: no hay forma de pedir uno a nombre de otra persona.

### 6.2 Abrir el túnel SSH

El servidor solo escucha en `127.0.0.1`, así que desde tu máquina:

```bash
ssh -N -L 5000:127.0.0.1:5000 usuario@host-del-servicio
```

Mientras el túnel esté arriba, `http://localhost:5000` en tu máquina llega al MCP.

### 6.3 Configurar el cliente MCP

Transporte **Streamable HTTP**, URL `http://localhost:5000`, y el token en el header:

```json
{
  "url": "http://localhost:5000",
  "headers": { "Authorization": "Bearer <token>" }
}
```

La configuración exacta por cliente está en [`docs/agents/`](../agents/): [Claude Code](../agents/claude.md), [Codex CLI](../agents/codex.md), [OpenCode](../agents/opencode.md).

### 6.4 Qué pasa al reiniciar el servicio

La clave que firma los tokens se genera en memoria al arrancar y nunca toca disco. **Reiniciar el servicio invalida todos los tokens emitidos**: cada operador debe volver a ejecutar `linux-mcp auth`. Ese es también el único mecanismo de revocación masiva disponible.

## 7. Actualizar

Si cambió la unit, instálala **antes** de reiniciar; de lo contrario el servicio se reinicia con la definición vieja. En particular, una unit antigua con solo `CAP_DAC_READ_SEARCH` y `ProtectProc=invisible` deja `ss` sin Pid/Process en procesos ajenos aunque el binario sea nuevo. Desde el repo:

```bash
sudo install -m 0644 deploy/systemd/linux-mcp.service /etc/systemd/system/linux-mcp.service
sudo systemctl daemon-reload
```

Sin clonar (siempre `main`):

```bash
curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/deploy/systemd/linux-mcp.service \
  -o /tmp/linux-mcp.service
sudo install -m 0644 /tmp/linux-mcp.service /etc/systemd/system/linux-mcp.service
sudo systemctl daemon-reload
```

Luego el binario. Desde Releases (mismo patrón que en [§2.1](#21-descargar-un-binario-publicado-recomendado)):

```bash
VERSION=v0.10.0   # última en Releases
ARCH=amd64        # o arm64
BASE=https://github.com/KaribuLab/linux-mcp/releases/download/$VERSION
curl -fsSLO $BASE/linux-mcp-linux-$ARCH
curl -fsSLO $BASE/SHA256SUMS
sha256sum --ignore-missing -c SHA256SUMS
sudo install -m 0755 linux-mcp-linux-$ARCH /usr/local/bin/linux-mcp
sudo systemctl restart linux-mcp.service
```

O recompilando:

```bash
go tool task build
sudo install -m 0755 dist/linux-mcp-$(go env GOOS)-$(go env GOARCH) /usr/local/bin/linux-mcp
sudo systemctl restart linux-mcp.service
```

Avisa a los operadores: después del reinicio necesitan un token nuevo.

## 8. Desinstalar

```bash
sudo systemctl disable --now linux-mcp.service
sudo rm -f /etc/systemd/system/linux-mcp.service
sudo systemctl daemon-reload
sudo rm -f /usr/local/bin/linux-mcp
# opcional: sudo userdel mcp-agent && sudo groupdel mcp-admin
```

## Troubleshooting

| Síntoma | Causa probable | Qué hacer |
|---------|----------------|-----------|
| `permission denied` al ejecutar `linux-mcp auth` | La membresía en `mcp-admin` no está refrescada en la sesión actual | `id -nG` para confirmar; cerrar sesión y volver a entrar |
| `permission denied` incluso tras re-login | El socket no quedó en `0660 mcp-agent:mcp-admin` | `ls -l /run/linux-mcp/issue.sock`; revisar `SupplementaryGroups` en la unit y reiniciar |
| `no socket at /run/linux-mcp/issue.sock` | El servicio no está corriendo, o la unit no declara `RuntimeDirectory=linux-mcp` | `systemctl status linux-mcp`; `journalctl -u linux-mcp` |
| El servicio no arranca y el log menciona el grupo | `mcp-admin` no existe | `sudo groupadd --system mcp-admin` y reiniciar |
| `401` desde el cliente con un token que antes servía | El token expiró o el servicio se reinició | Volver a ejecutar `linux-mcp auth` |
| `403 origin not allowed` | Un cliente de browser manda un `Origin` fuera del allowlist | El log del servicio trae el origen exacto; añadirlo con `--cors` |
| `403 host not allowed` | El header `Host` no coincide con el `--addr` del servicio | Usar `localhost:5000` o `127.0.0.1:5000` a través del túnel, sin proxies que reescriban `Host` |
| `permission denied` al leer configs de sistema | Falta `CAP_DAC_READ_SEARCH` | `systemctl show linux-mcp`; revisar la unit |
| `ss`/`ss_grep`: Pid/Process vacíos en listeners de otros uids | Unit vieja, sin `CAP_SYS_PTRACE`, o `ProtectProc=invisible` | Reinstalá unit del repo ([§3](#3-instalar-y-habilitar-la-unit) / [§7](#7-actualizar)); `systemctl show` caps y ProtectProc; re-verificá [§4](#4-verificar) |
| `cat`/`list` bloquean paths | Esperado: denylist en app (`/etc/shadow`, keys, etc.) | Systemd `InaccessiblePaths` es complemento, no sustituto |
| `sha256sum: SHA256SUMS: no properly formatted...` | `SHA256SUMS` no es el archivo del release (suele ser HTML 404) | Confirmá que el tag existe en Releases; usá `curl -fsSL` y revisá `head SHA256SUMS` |
| `curl: (22) The requested URL returned error: 404` | El tag o el asset no existen | Listá tags reales en Releases o compilá con [§2.2](#22-compilar-desde-el-repo) |
| `status=11/SEGV` / `Result: core-dump` al arrancar | Unit vieja con `SystemCallFilter=@system-service` (seccomp + Go) | Reinstalá la unit del repo ([§3](#3-instalar-y-habilitar-la-unit) / [§7](#7-actualizar)); borrá drop-ins de debug en `/etc/systemd/system/linux-mcp.service.d/` si quedaron |

## Referencias

- Unit: [`deploy/systemd/linux-mcp.service`](../../deploy/systemd/linux-mcp.service)
- Comandos: [`docs/commands/serve.md`](../commands/serve.md), [`docs/commands/auth.md`](../commands/auth.md)
- Tools: [`docs/tools/cat.md`](../tools/cat.md), [`docs/tools/list.md`](../tools/list.md)
