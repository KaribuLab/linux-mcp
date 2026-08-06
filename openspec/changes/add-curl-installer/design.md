## Context

Hoy la única vía documentada para desplegar linux-mcp es `docs/runbooks/install-systemd.md`, que enumera ~10 comandos sueltos: crear `mcp-agent` y `mcp-admin`, descargar binario + `SHA256SUMS`, validar hash, descargar unit desde `main` (no desde el tag), `daemon-reload`, `enable --now`, y verificar. El runbook ya tiene todas las piezas y la CI publica binarios firmados en cada merge a `main`, así que el material está listo: lo que falta es unirlo en un único script versionado en el repo, escrito en **POSIX `sh`** para no asumir bash, que un operador pueda invocar con `curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/install.sh | sudo sh`.

Restricciones heredadas del sistema actual:

- El binario sale de **Releases** (`linux-mcp-linux-{amd64,arm64}` + `SHA256SUMS`), generado por `.github/workflows/ci.yml` con kli y semver desde Conventional Commits.
- La unit sale de **`main`**, no del tag. Es config de deploy, no un asset versionado. Esta asimetría se mantiene.
- El grupo `mcp-admin` debe existir **antes** de que el servicio arranque (la unit lo declara en `SupplementaryGroups`).
- El socket de emisión debe quedar `srw-rw---- mcp-agent mcp-admin` en `/run/linux-mcp/issue.sock`. Cualquier otra combinación rompe `linux-mcp auth`.
- El endpoint en loopback responde `401` sin token, que es la señal canónica de "vivo y protegido".
- La membresía de grupo solo aplica al abrir nueva sesión (limitación del propio `gpasswd`/`usermod`, no del script).

## Goals / Non-Goals

**Goals:**

- Un solo comando root-less-friendly-en-lectura (`curl -fsSL <url> | sudo sh`) que llega a un servicio funcional.
- Idempotencia total: re-ejecutar no rompe nada, solo aplica la última versión disponible y reemplaza la unit si cambió.
- Validación de SHA256 antes de tocar `/usr/local/bin/`, con cleanup obligatorio del tmpdir aunque la validación falle.
- Falla ruidosa: aborta si `systemctl` no existe, si la arquitectura no es `amd64`/`arm64`, si la verificación final no se cumple, o si el binario descargado no responde `--version`.
- Descubribilidad: el archivo vive en la raíz del repo (`install.sh`) para que su URL sea estable y un humano pueda leerlo antes de pipe-to-bash.

**Non-Goals:**

- No compila desde fuente (`go tool task build` sigue siendo el camino documentado).
- No reemplaza la sección de `update` ni `uninstall` del runbook: el script es **install** (incluye un camino de upgrade implícito porque siempre baja la última, pero no se posiciona como canal oficial de update).
- No añade operadores a `mcp-admin`. La membresía queda al operador (`usermod -aG mcp-admin maria`), igual que ahora.
- No auto-detecta CORS allowlist ni configuración de cliente MCP.
- No incluye `--dry-run`, `--uninstall` ni flags generales en esta iteración.

## Decisions

### D1. Script en la raíz del repo: `install.sh`

**Decisión:** `install.sh` ejecutable (`0755` via `git update-index --chmod=+x`), en la raíz, con shebang `#!/bin/sh` (POSIX).

**Por qué:** la URL canónica del one-liner es `https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/install.sh`. Si viviera bajo `scripts/install.sh` o `deploy/install.sh`, la URL pública seguiría funcionando pero la descubribilidad cae y el script deja de parecer "el" instalador. `#!/bin/sh` (en lugar de `#!/usr/bin/env bash` o `#!/bin/bash`) garantiza que el script corre en cualquier Linux de servidor sin requerir bash — `/bin/sh` es POSIX-mandated y está presente en Debian (suele ser `dash`), RHEL (bash en modo POSIX), Alpine (busybox `ash`) y cualquier otra distribución sensata.

**Alternativas consideradas:**

- `scripts/install.sh`: más ordenado, peor para copy-paste mental.
- Solo en `docs/`: choca con el hecho de que `docs/` no contiene código ejecutable.

### D2. Detección de versión: `LINUX_MCP_VERSION` con default `latest`

**Decisión:** si `LINUX_MCP_VERSION` está seteada y es válida (`vX.Y.Z`), se respeta. Si no, se resuelve `latest` siguiendo el redirect de `https://github.com/KaribuLab/linux-mcp/releases/latest` con `curl -fsSL -o /dev/null -w '%{url_effective}'`. La URL final es `/releases/tag/<tag>`; el script parsea el último segmento. No usa la API REST, así que no consume rate-limit.

**Por qué:** el patrón `curl | bash` implica "querés lo último" por defecto. Permitir override por env es la única puerta de escape para reproducibilidad y para air-gapped con proxy.

**Alternativas consideradas:**

- API REST `https://api.github.com/repos/.../releases/latest`: consume rate-limit (60/h sin token) y agrega una dependencia de auth para la próxima vez que alguien suba el límite.
- Tag fijo hardcoded: anti-patrón, requiere actualizar el script en cada release.
- Solo `latest`: pierde reproducibilidad.

### D3. Detección de arquitectura: `uname -m` mapeado a `amd64`/`arm64`

**Decisión:** `case "$(uname -m)" in x86_64) ARCH=amd64 ;; aarch64) ARCH=arm64 ;; *) die ;; esac`.

**Por qué:** coincide exactamente con el runbook actual y con los assets que publica la CI (`linux-mcp-linux-{amd64,arm64}` + `SHA256SUMS`). El mapeo `aarch64 → arm64` cubre Raspberry Pi 3, 4, 5 y Zero 2 W con Raspberry Pi OS 64-bit (Bookworm o superior) o cualquier otra distribución ARMv8 reportando `aarch64` (Ubuntu Server ARM64, Debian ARM64, etc.). Cualquier otra arquitectura — `armv7l` / `armv6l` (Pi 1/2/Zero con OS 32-bit), `i386`, `riscv64` — aborta con mensaje claro apuntando a "Compilar desde el repo", porque no hay binario publicado para ellas y compilar desde fuente está documentado en el runbook. Soportar ARM 32-bit implicaría que la CI publique un tercer asset (`linux-mcp-linux-arm`), lo cual queda fuera de este change.

### D4. Validación de hash: tmpdir dedicado + `sha256sum --ignore-missing -c`

**Decisión:** `mktemp -d`, descargar binario y `SHA256SUMS` adentro, validar, y solo entonces `install -m 0755` a `/usr/local/bin/linux-mcp`. `trap 'rm -rf "$TMPDIR"' EXIT` para cleanup obligatorio.

**Por qué:** exactamente el patrón que ya recomienda el runbook (`-f`, `sha256sum --ignore-missing`). El tmpdir dedicado evita pisar un binario válido en `/usr/local/bin/` si la validación falla.

### D5. Idempotencia de usuario/grupo con `getent`

**Decisión:** `getent group mcp-admin || groupadd --system mcp-admin`. Igual para `mcp-agent` con `getent passwd`. Sin flags de "ya existe" en `useradd`/`groupadd` porque esos flags varían entre distros (`--non-unique` en algunas, errores fatales en otras).

**Por qué:** `getent` es POSIX y portable. Si la membresía del grupo cambia (porque alguien añadió un operador), el script no la toca.

### D6. Unit siempre desde `main`, sobrescribiendo si existe

**Decisión:** `curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/deploy/systemd/linux-mcp.service -o /tmp/linux-mcp.service`, `install -m 0644` sobre `/etc/systemd/system/linux-mcp.service`, `systemctl daemon-reload`, `systemctl enable --now linux-mcp.service`.

**Por qué:** misma decisión que el runbook (la unit es config de deploy, no asset del release). Sobrescribir asegura que el binario nuevo vea capabilities/hardening nuevos — exactamente el escenario "actualizar" del runbook, pero aplicado automáticamente.

### D7. Verificación final con poll al endpoint

**Decisión:** bucle de hasta 10 s con `curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:5000`, sale en cuanto devuelve `401`. Después, `stat -c '%a %U %G' /run/linux-mcp/issue.sock` y compara contra `660 mcp-agent mcp-admin`. Si algo no coincide → falla con diagnóstico (sugerir Troubleshooting del runbook).

**Por qué:** `401` es la prueba de que el servidor arrancó y está aplicando la policy de auth. `ls -l` parseado con `awk` es frágil en i18n; `stat -c` es estable.

### D8. ShellCheck en CI con shell POSIX

**Decisión:** nuevo job `lint-shell` en `.github/workflows/ci.yml` que corre `shellcheck -s sh install.sh` y falla el build ante warnings de severidad `error` o `warning`.

**Por qué:** shell sin chequeo estático es frágil; un solo cambio de quoting puede introducir un `set -u` mal puesto y romper el script en producción. Pasar `-s sh` fuerza a shellcheck a evaluar como POSIX `sh`, lo que detecta extensiones bash inadvertidas (`[[ ]]`, `local`, arrays, `pipefail`) antes de que el script llegue a un servidor donde `/bin/sh` es `dash` y esas extensiones revientan.

**Alternativas consideradas:**

- `sh -n install.sh`: solo sintaxis, no semántica.
- `shellcheck install.sh` (sin `-s sh`): shellcheck intenta adivinar el dialecto del shebang; con `#!/bin/sh` debería atinar pero `-s sh` lo hace explícito y robusto.
- Pre-commit local: opt-in, no protege a terceros que manden PRs sin hook.

### D9. Sin flags, sin subcomandos

**Decisión:** el script no acepta argumentos posicionales ni flags. Las dos únicas perillas son las env vars `LINUX_MCP_VERSION` (override de versión) y `LINUX_MCP_BINDIR` (override del destino, default `/usr/local/bin`).

**Por qué:** la audiencia del one-liner quiere cero fricción. Flags añadirían la tentación de "modo silencioso" o "skip verify", que va contra el objetivo de "falla ruidosa". Las dos env vars cubren los dos casos legítimos (versión fija y destino no estándar, ej. `/opt`).

## Risks / Trade-offs

- **[Pipe-to-bash descarga de `main`]** → El runbook nuevo y el snippet del `README` deben incluir un párrafo corto: "Antes de ejecutar, revisá el script. La unit siempre sale de `main`; un cambio reciente de capabilities puede afectar tu despliegue."
- **[Cambio en la unit aplicado silenciosamente al re-ejecutar]** → Documentado en runbook; comportamiento consistente con el espíritu de "instalá la unit antes de reiniciar" del runbook actual.
- **`uname -m` devuelve algo no mapeado (ej. `armv7l` en Raspberry Pi con OS 32-bit)** → abort inmediato con mensaje que mencione que arquitecturas soportadas son `amd64` (x86_64) y `arm64` (aarch64, que cubre Raspberry Pi 3/4/5 con OS 64-bit), y que apunte al camino "Compilar desde el repo" del runbook.
- **`curl` rate-limit si un operador re-ejecuta el script 100 veces en una hora contra `releases/latest`** → las redirects a `/releases/tag/<tag>` no son la API REST y no consumen rate-limit. El binario y `SHA256SUMS` salen de `objects.githubusercontent.com` (CDN).
- **`mcp-admin` existe pero con un GID distinto al que la unit espera** → la unit no fija GID, solo nombre (`Group=mcp-agent`, `SupplementaryGroups=mcp-admin`), así que no hay drift. Pero si un operador ya tenía `mcp-admin` con miembros de un deploy anterior, el script no los pisa. Documentado: la membresía sigue siendo decisión humana.
- **Cambio en el contrato de la unit rompe un deploy existente** → el script siempre sobrescribe la unit y reinicia el servicio. Esto invalida todos los tokens emitidos (mismo comportamiento que `systemctl restart` hoy). El runbook ya advierte sobre este comportamiento; el snippet de salida del script debe recordarlo.
- **`shellcheck -s sh` agrega un job más al CI** → ~5 s, `linux/amd64`, sin red externa. Aceptable.
- **Perder `pipefail` en POSIX `sh`** → sin esta opción, una pipeline devuelve el exit del último comando. En este script las dos pipelines con consecuencias reales (`curl … | grep …` para parsear el redirect del tag, `curl … | sh` que no usamos) se manejan evitando la pipeline o capturando el exit antes de seguir; el resto son pipelines de logging sin impacto en el resultado. Documentado en tasks.

## Migration Plan

No hay rollout gradual: el script aparece, el runbook lo apunta como método recomendado, y los pasos manuales siguen como referencia.

1. Merge del change → `install.sh` aparece en `main`.
2. CI pasa (incluyendo nuevo job `shellcheck`).
3. Tag siguiente → binario disponible en Releases.
4. El comando `curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/install.sh | sudo sh` funciona para un operador nuevo.
5. Operadores existentes pueden migrar ejecutando el mismo comando: idempotente, descarga la última versión, reemplaza la unit, reinicia.

**Rollback:** borrar `install.sh` del repo. No hay estado de servidor a deshacer: el script no deja archivos propios más allá de lo que ya pone el runbook (binario, unit, grupos, socket efímero).

## Open Questions

- ¿Queremos también un `uninstall.sh` simétrico? Decisión: no en este change; queda como follow-up si aparece demanda.
- ¿Sumar `--no-enable` para entornos donde systemd está deshabilitado pero el operador quiere el binario listo? Decisión: no; el script aborta si `systemctl` no existe, que es el comportamiento correcto hoy. Si surge el caso, se añade como flag.
- ¿Publicar un SHA256SUMS firmado con cosign o sigstore además del SHA plano? Decisión: out of scope; Releases ya tiene checksums firmados por el hash de GitHub (atributo `verified` en la UI). El SHA256 plano es la práctica del runbook actual.