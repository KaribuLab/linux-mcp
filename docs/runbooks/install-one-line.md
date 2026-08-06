# Instalar linux-mcp con un one-liner

Forma recomendada de desplegar linux-mcp en un host Linux con systemd. El script vive en `install.sh` en la raíz del repo y se sirve desde `main`. Automatiza todos los pasos del [runbook de systemd](install-systemd.md): descarga binario, valida SHA256, crea `mcp-agent` y `mcp-admin`, instala la unit, habilita y arranca el servicio, y verifica el estado final.

## TL;DR

```bash
curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/install.sh | sudo sh
```

Después:

```bash
sudo usermod -aG mcp-admin <usuario>     # autorizar un operador
# (el usuario debe cerrar sesión y volver a entrar)

ssh -N -L 5000:127.0.0.1:5000 usuario@host-del-servicio
TOKEN=$(linux-mcp auth --ttl 8h)         # ejecutar en el host como miembro de mcp-admin
```

Configurar el cliente MCP con transporte **Streamable HTTP**, URL `http://localhost:5000`, y header `Authorization: Bearer <token>`.

## Qué hace el script

1. Detecta la arquitectura (`uname -m`): `x86_64` → `amd64`, `aarch64` → `arm64` (cubre Raspberry Pi 3/4/5 con OS 64-bit). Cualquier otra aborta.
2. Resuelve la versión: si `LINUX_MCP_VERSION=vX.Y.Z` está seteada en el entorno, la respeta; si no, sigue el redirect de `/releases/latest` (sin consumir rate-limit de la API REST) y usa la última publicada.
3. Descarga el binario + `SHA256SUMS` desde GitHub Releases a un `mktemp -d`, valida el hash con `sha256sum --ignore-missing -c`, y solo entonces lo instala con `install -m 0755` en `${LINUX_MCP_BINDIR:-/usr/local/bin}/linux-mcp`.
4. Verifica que el binario responde `--version` con `linux-mcp version <tag>`.
5. Crea el grupo `mcp-admin` y el usuario `mcp-agent` (con `home=/nonexistent`, `shell=/usr/sbin/nologin`) si no existen. No toca la membresía del grupo.
6. Descarga `deploy/systemd/linux-mcp.service` desde `main` (no desde el tag del binario), la instala en `/etc/systemd/system/linux-mcp.service`, ejecuta `daemon-reload` y `enable --now`.
7. Verifica que `systemctl is-active linux-mcp.service` es `active`.
8. Hace poll de hasta 20 s contra `http://127.0.0.1:5000` hasta ver `401` (señal canónica de "vivo y protegido").
9. Verifica que `/run/linux-mcp/issue.sock` existe con modo `0660`, owner `mcp-agent`, grupo `mcp-admin`.

Si cualquier paso falla, aborta con código distinto de cero y un mensaje claro. No deja el sistema a medio instalar: el trap de EXIT limpia el tmpdir, y si la validación de SHA256 falla el binario en `/usr/local/bin/linux-mcp` queda intacto.

## Variables de entorno

| Variable | Default | Para qué |
|----------|---------|----------|
| `LINUX_MCP_VERSION` | latest | Fija la versión a instalar (formato `vX.Y.Z`). Útil para reproducibilidad o entornos air-gapped con proxy. |
| `LINUX_MCP_BINDIR` | `/usr/local/bin` | Destino del binario. Útil si querés instalarlo en `/opt` u otro path no estándar. |

Ejemplos:

```bash
# Instalar una versión específica
LINUX_MCP_VERSION=v0.10.0 curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/install.sh | sudo sh

# Instalar el binario en /opt (necesitás que /opt/bin exista y esté en PATH)
curl -fsSL ... | sudo env LINUX_MCP_BINDIR=/opt sh
```

## Requisitos

- Linux con systemd (`amd64` o `arm64`).
- Privilegios root (el script aborta con mensaje claro si lo invocan sin `sudo`).
- `curl`, `sha256sum`, `getent`, `mktemp`, `install`, `stat`, `systemctl` en `PATH`. Si falta alguno, el script aborta indicando cuál.
- Acceso saliente a `github.com` y `raw.githubusercontent.com`.

## Re-ejecución / upgrade

El script es idempotente. Re-ejecutarlo:

- Reemplaza el binario por la última versión (o la fijada por `LINUX_MCP_VERSION`).
- Reemplaza la unit por la de `main` actual (si cambió).
- Recarga systemd y reinicia el servicio.

**Consecuencia importante:** reiniciar el servicio invalida todos los tokens emitidos (la clave de firma vive solo en memoria). Cada operador debe volver a ejecutar `linux-mcp auth`. Este comportamiento es idéntico a `systemctl restart linux-mcp.service` manual.

## Advertencias

- **Antes de ejecutar**, revisá el script en `https://github.com/KaribuLab/linux-mcp/blob/main/install.sh`. Es el contrato que aceptás al hacer pipe-to-sh.
- **La unit sale de `main`**, no del tag. Un cambio reciente de capabilities / hardening puede afectar un deploy existente. Si eso pasa, re-ejecutá el script después de mergear el cambio.
- **Membresía de grupo**: el script crea `mcp-admin` pero NO añade operadores automáticamente. Eso lo hacés vos con `sudo usermod -aG mcp-admin <usuario>`.
- **Caso no-sudo**: si invocás el script sin `sudo`, aborta en el primer chequeo con un mensaje apuntando al comando correcto. No toca nada.
- **Caso sin systemd** (ej. contenedor `debian-slim`): aborta en el chequeo de `systemctl` sin descargar binario ni unit.
- **Caso arquitecturas no soportadas** (`armv7l`, `armv6l`, `i386`, `riscv64`): aborta inmediatamente. Para esas hay que [compilar desde el repo](install-systemd.md#22-compilar-desde-el-repo).
- **Pipe-to-sh no es pipe-to-bash**: el script es POSIX `sh` y funciona con `/bin/sh = dash` (Debian/Ubuntu), `bash` en modo POSIX (RHEL), o `ash` (Alpine). Probado con `shellcheck -s sh`.

## Qué sigue

- [Runbook de systemd (pasos manuales)](install-systemd.md) — referencia, no necesitás seguirlo si usás el one-liner.
- [Unidad de referencia](https://github.com/KaribuLab/linux-mcp/blob/main/deploy/systemd/linux-mcp.service) — config de deploy versionada en el repo.
- [Comandos](https://github.com/KaribuLab/linux-mcp/blob/main/docs/commands/) — `serve`, `auth`.
- [Agentes MCP](https://github.com/KaribuLab/linux-mcp/blob/main/docs/agents/) — configuración por cliente (Claude Code, Codex CLI, OpenCode, Inspector).

## Troubleshooting

Cualquier fallo del script imprime un mensaje a stderr que apunta a esta sección. Los síntomas típicos después de una instalación exitosa están cubiertos en el [Troubleshooting del runbook de systemd](install-systemd.md#troubleshooting); los que son específicos del one-liner:

| Síntoma | Causa probable | Qué hacer |
|---------|----------------|-----------|
| `ERROR: este instalador requiere root` | Lo invocaste sin `sudo` | Volvé a invocarlo como `curl ... | sudo sh` |
| `ERROR: falta el comando 'X'` | El host no tiene `X` (ej. `systemctl` en un contenedor sin systemd, o `install` en Alpine busybox puro) | Instalá el paquete que provee `X` o usá un host soportado |
| `ERROR: arquitectura no soportada: <X>` | `uname -m` devolvió algo distinto de `x86_64` y `aarch64` | Compilar desde el repo (`docs/runbooks/install-systemd.md` §2.2) |
| `ERROR: no existe el release <X>` | Tag inválido en `LINUX_MCP_VERSION` o no hay red para resolver `latest` | Verificá el tag en Releases; exportá `LINUX_MCP_VERSION` con uno válido |
| `ERROR: SHA256 inválido` | El asset descargado no coincide con `SHA256SUMS` (casi siempre: red MITM o release corrupto) | Reintentá. Si persiste, descargá manualmente desde Releases y compará `sha256sum` |
| `ERROR: el endpoint http://127.0.0.1:5000 nunca devolvió 401 dentro de 20s` | El servicio arrancó pero algo bloquea el puerto / la policy de auth | `systemctl status linux-mcp.service` y `journalctl -u linux-mcp.service -n 50 --no-pager` |
| `ERROR: socket con permisos/owner incorrectos` | El socket no quedó en `0660 mcp-agent:mcp-admin` (unit vieja, o `mcp-admin` no existía al arrancar) | `systemctl show linux-mcp.service -p SupplementaryGroups`; reinstalá con el one-liner que reescribe la unit |
| `ERROR: el servicio no quedó activo` | `enable --now` falló | `journalctl -u linux-mcp.service` para el motivo exacto |