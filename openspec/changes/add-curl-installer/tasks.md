## 1. Scaffold del script

- [x] 1.1 Crear `install.sh` en la raíz del repo con shebang `#!/bin/sh`, `set -eu` (sin `pipefail` — no es POSIX), trap de EXIT para cleanup de tmpdir, y banner inicial a stderr. Sin bashisms: usar `[ ]` en lugar de `[[ ]]`, no usar `local` (variables de función con prefijo único), no usar arrays, no usar `printf -v`. Validar con `shellcheck -s sh install.sh` localmente antes de commit
- [x] 1.2 Marcar el archivo ejecutable en git (`git update-index --chmod=+x install.sh`)
- [x] 1.3 Stubear las funciones principales vacías: `die()`, `require_root()`, `require_cmd()`, `detect_arch()`, `resolve_version()`, `install_binary()`, `provision_users()`, `install_unit()`, `verify()` — solo firmas con `return 0`, el cuerpo se llena en las tasks siguientes

## 2. Pruebas de prerrequisitos

- [x] 2.1 Implementar `require_root()`: comprobar `$(id -u)` y abortar con mensaje si no es `0`
- [x] 2.2 Implementar `require_cmd()` y lazo de chequeo: exigir `systemctl`, `curl`, `sha256sum`, `getent`, `mktemp`, `install`, `stat`. Cualquier faltante aborta con qué herramienta falta y por qué
- [x] 2.3 Probar el script en un contenedor sin `systemctl` (ej. `debian-slim`) y verificar que aborta limpio con el mensaje correcto — validado con `docker run --rm --user root --mount type=bind debian:12-slim sh /install.sh`: imprime `[install] ERROR: falta el comando 'systemctl'. Instalalo antes de continuar.` y sale con exit 1

## 3. Detección de arquitectura y versión

- [x] 3.1 Implementar `detect_arch()` con `case "$(uname -m)" in x86_64) ARCH=amd64 ;; aarch64) ARCH=arm64 ;; *) die "Arquitectura no soportada: ..." ;; esac`. Cualquier otra cosa aborta
- [x] 3.2 Implementar `resolve_version()`: si `LINUX_MCP_VERSION` está seteada y matchea `^v[0-9]+\.[0-9]+\.[0-9]+$`, usarla. Si no, seguir el redirect de `https://github.com/KaribuLab/linux-mcp/releases/latest` con `curl -fsSL -o /dev/null -w '%{url_effective}'` y parsear el último segmento del path
- [x] 3.3 Validar que la versión resuelta exista realmente: hacer un `curl -fsI` contra `https://github.com/KaribuLab/linux-mcp/releases/download/<version>/SHA256SUMS` y abortar si devuelve 404

## 4. Descarga y validación del binario

- [x] 4.1 Implementar `install_binary()`: `mktemp -d`, `curl -fsSLO` del binario y de `SHA256SUMS`, validar con `sha256sum --ignore-missing -c SHA256SUMS` dentro del tmpdir, `install -m 0755` a `${LINUX_MCP_BINDIR:-/usr/local/bin}/linux-mcp`. Si hash no valida: abortar, NO pisar el binario actual
- [x] 4.2 Validar `--version` post-instalación: el binario debe responder `linux-mcp version <tag>` (parseo con `grep -E`)

## 5. Creación idempotente de usuario y grupo

- [x] 5.1 Implementar `provision_users()`: `getent group mcp-admin >/dev/null || groupadd --system mcp-admin`; `getent passwd mcp-agent >/dev/null || useradd --system --home /nonexistent --shell /usr/sbin/nologin mcp-agent`
- [x] 5.2 Verificar que `mcp-admin` existe antes de invocar `systemctl enable --now` (si la unit declara `SupplementaryGroups=mcp-admin` y el grupo no existe, `serve` falla)

## 6. Instalación de unit y activación

- [x] 6.1 Implementar `install_unit()`: `curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/deploy/systemd/linux-mcp.service -o /tmp/linux-mcp.service`, `install -m 0644` en `/etc/systemd/system/linux-mcp.service`
- [x] 6.2 Encadenar `systemctl daemon-reload` y `systemctl enable --now linux-mcp.service`. Capturar stderr de `daemon-reload` y `enable` para el log final
- [x] 6.3 Si el servicio no quedó `active` después de enable, abortar con mensaje que pida revisar `journalctl -u linux-mcp.service`

## 7. Verificación final

- [x] 7.1 Implementar `verify_endpoint()`: poll de hasta 10 s contra `http://127.0.0.1:5000`, salir en cuanto `curl -sS -o /dev/null -w '%{http_code}'` devuelva `401`
- [x] 7.2 Implementar `verify_socket()`: `stat -c '%a %U %G' /run/linux-mcp/issue.sock` debe devolver `660 mcp-agent mcp-admin`. Cualquier otra cosa aborta
- [x] 7.3 Si `verify_endpoint` o `verify_socket` fallan, imprimir un mensaje apuntando a "Troubleshooting" del runbook

## 8. Documentación

- [x] 8.1 Crear `docs/runbooks/install-one-line.md` con la receta canónica, advertencias ("la unit viene de `main`", "pipe-to-bash: revisá antes"), y enlaces cruzados al runbook systemd existente
- [x] 8.2 Editar `docs/runbooks/install-systemd.md`: añadir una sección al inicio "One-liner" con el comando canónico, mantener el resto como referencia manual
- [x] 8.3 Editar `README.md`: en la sección de instalación (o crearla si no existe), añadir el bloque con el comando canónico
- [x] 8.4 Asegurar que `docs/README.md` siga apuntando al runbook systemd (sin cambios esperados, solo verificar) — además se añadió fila para `install-one-line.md` arriba de `install-systemd.md`

## 9. CI

- [x] 9.1 Añadir step `shellcheck -s sh install.sh` al job `verify` en `.github/workflows/ci.yml`, después del build y antes de publicar binarios. Fallar el build ante `error` o `warning`. El `-s sh` es obligatorio: sin él, shellcheck no enforce POSIX y un bashism pasaría el linter
- [x] 9.2 Si shellcheck no está disponible en `ubuntu-latest`, instalarlo vía `sudo apt-get install -y shellcheck` antes del step

## 10. Validación automatizable en CI

> Estas tareas reemplazan las validaciones E2E manuales originales (10.1–10.6 del primer borrador). No requieren hosts reales: corren en `ubuntu-latest` con paquetes estándar. La validación post-deployment en una Pi real queda documentada en `docs/runbooks/install-one-line.md` (Troubleshooting) pero ya no es una task de OpenSpec porque OpenSpec exige tasks verificables por quien implementa, no por el operador.

- [x] 10.1 Crear `.github/scripts/scan-bashisms.sh` (POSIX `sh`, lintereado con `shellcheck -s sh`): grepea patrones comunes de bash (`[[ ]]`, `pipefail`, `${var,,}`, `${var^^}`, `local`, `declare -a`, `printf -v`, `mapfile`, process substitution, `&>`). Es defensa-en-profundidad sobre shellcheck, no un reemplazo
- [x] 10.2 Añadir step al CI que instale `dash` y `busybox-static` y corra `dash -n install.sh` y `busybox sh -n install.sh` (chequeo de sintaxis contra las dos implementaciones de `/bin/sh` más comunes en servers)
- [x] 10.3 Añadir step al CI que corra `.github/scripts/scan-bashisms.sh install.sh`. Debe pasar limpio
- [x] 10.4 Validar localmente: `dash -n install.sh`, `busybox sh -n install.sh` y `.github/scripts/scan-bashisms.sh install.sh` deben pasar antes de mergear — verificado en este entorno con dash 0.5.12 (Debian 12) y busybox 1.35.0 (Debian 12), extraídos vía docker