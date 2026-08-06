## Why

El runbook `docs/runbooks/install-systemd.md` exige ~10 comandos separados para llegar a un servicio systemd funcional (crear `mcp-agent`/`mcp-admin`, descargar binario, validar SHA256, descargar unit desde `main`, `daemon-reload`, `enable --now`, verificar con `curl`/`ls`). Cada paso tiene matices — `-f` en `curl`, grupo antes que unit, `mcp-admin` debe existir antes de arrancar — que se pierden cuando un operador nuevo copy-pastea a medias. Un instalador reproducible de una línea (`curl -fsSL … | sudo sh`) reduce ese conocimiento implícito a un único artefacto versionado en el repo y elimina drift entre "lo que dice el runbook" y "lo que ejecuta la gente".

## What Changes

- Nuevo script `install.sh` en la raíz del repo (raíz para que sea descubrible y para que `raw.githubusercontent.com/.../main/install.sh` sea la URL del one-liner).
- El script implementa, de forma idempotente, todo el flujo del runbook: detección de versión (`latest` desde la API de GitHub Releases, o fija por env `LINUX_MCP_VERSION`), detección de `arch` (`uname -m`), descarga del binario + `SHA256SUMS`, validación de hash, `install -m 0755` en `/usr/local/bin/linux-mcp`, `useradd --system mcp-agent` / `groupadd --system mcp-admin` (idempotente: detecta si ya existen), descarga de la unit desde `main` (sin tag, igual que hoy), `daemon-reload`, `enable --now`, verificación final (`curl 127.0.0.1:5000` → 401 esperado, `ls -l /run/linux-mcp/issue.sock` con modo `0660 mcp-agent:mcp-admin`).
- El script está escrito en **POSIX `sh`** (shebang `#!/bin/sh`, sin dependencia de bash) para correr en cualquier Linux de servidor sin requerir un shell particular. Aborta con `set -eu` (no se usa `pipefail` porque no es POSIX; las pipelines críticas se manejan chequeando el exit del último paso), exige root, exige `systemctl` y `curl`, y falla ruidosamente si `SHA256SUMS` no valida o si la verificación final no se cumple. Logs a stderr, paths a stdout, igual que el patrón de `linux-mcp auth`.
- Nueva sección en `docs/runbooks/install-systemd.md` que apunta al one-liner como método recomendado y mantiene los pasos manuales como referencia / flujo de "sin clonar el repo".
- Entrada nueva en `README.md` (sección de instalación) con el comando canónico.
- No cambia el binario, ni la unit, ni las tools. No es breaking.

## Capabilities

### New Capabilities
- `one-line-installer`: instalador shell versionado en el repo, escrito en **POSIX `sh`** (no bash), que descarga binario + unit, valida hash, crea usuario/grupo, habilita el servicio systemd y verifica el resultado, todo desde un único comando `curl | sudo sh`. Cubre idempotencia, detección de versión/arquitectura, manejo de errores y equivalencia funcional con los pasos manuales del runbook.

### Modified Capabilities
- (Vacío) El spec `systemd-install` existente cubre la unit, el runbook y el descubrimiento vía `docs/README.md`/`README.md`. La adición del one-liner es un nuevo método de despliegue paralelo; los requisitos de la unit, el binario y la verificación no cambian. El runbook se extiende (no se modifica semánticamente), así que el spec `systemd-install` no necesita delta.

## Impact

- Código:
  - `install.sh` (nuevo, ejecutable, ~150-200 líneas de POSIX `sh`, sin dependencia de bash).
  - `docs/runbooks/install-systemd.md` (sección nueva "One-liner" + ajustes de wording para reflejar que es el método recomendado).
  - `README.md` (snippet de instalación con el comando canónico).
  - `.github/workflows/ci.yml` (job nuevo `shellcheck` sobre `install.sh`; opcional pero barato).
- API/binarios: sin cambios. El binario, la unit y los flags `serve` siguen idénticos.
- Dependencias externas: `curl` (ya asumido por el runbook) y acceso saliente a `github.com` + `raw.githubusercontent.com`.
- Riesgos:
  - **Pipe-to-bash**: es el patrón que la audiencia eligió explícitamente, pero merece nota explícita en el runbook de que el script se sirve desde `main` (no desde un tag) y que conviene leerlo antes de ejecutarlo.
  - **`set -eu` (sin `pipefail`) + unit vieja**: el script omite `pipefail` por portabilidad POSIX; las pipelines donde importa el código de salida se manejan con `cmd1 && cmd2 || die` o capturando `$?` antes de la siguiente instrucción. Si la unit de `main` cambia (capabilities, hardening), un upgrade debe descargar la nueva unit antes de `restart`, exactamente como ya advierte el runbook.
  - **Idempotencia**: `usermod -aG mcp-admin` aplicado automáticamente podría sorprender; por eso el script NO añade usuarios al grupo, solo crea `mcp-admin` y deja la membresía al operador (igual que el runbook actual).

## Non-goals

- No compila desde el repo (sigue siendo el flujo de `go tool task build` documentado).
- No reemplaza `update` / `uninstall`: el script es instalación inicial; el runbook conserva esas secciones como referencia.
- No incluye modo `--dry-run` en esta iteración (se puede añadir si aparece demanda).
- No añade auto-update: la versión sigue siendo fija en el momento de la invocación (controlable vía `LINUX_MCP_VERSION`).