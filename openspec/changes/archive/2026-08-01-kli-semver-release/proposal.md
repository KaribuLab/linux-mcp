## Why

Hoy el pipeline solo publica un GitHub Release cuando alguien pushea a mano un tag `v*`. Sin tags no hay binarios descargables y la instalación desde Releases falla con 404. Hace falta versionar y publicar automáticamente a partir de Conventional Commits, sin PAT externo ni Release Please.

## What Changes

- En cada push a `main`, calcular la siguiente versión semántica con [KaribuLab/kli](https://github.com/KaribuLab/kli) (`kli semver`) según mensajes `feat:` / `fix:` / breaking.
- Si la versión calculada no coincide con el último tag, crear el tag con `kli semver -t` usando `GITHUB_TOKEN` (`contents: write`, y `id-token: write` si el job lo declara).
- En el mismo flujo (sin depender de un segundo workflow disparado por el tag, para evitar el anti-loop de `GITHUB_TOKEN`), compilar `linux/amd64` y `linux/arm64`, generar `SHA256SUMS` y publicar un GitHub Release con esos artefactos.
- Adaptar o fusionar el workflow `release.yml` actual para que no choque con este flujo.
- Documentar en README/runbook que los releases salen de commits en `main` vía kli (y que el tag de ejemplo debe existir en Releases).
- Tras el primer Release publicado por el pipeline, actualizar los ejemplos de descarga en `README.md` y el runbook para que `VERSION` apunte a la **última versión** real (no un placeholder).

## Non-goals

- No adoptar Release Please ni semantic-release.
- No exigir PAT ni deploy key SSH.
- No cambiar el comportamiento del servidor MCP ni de las tools.
- No publicar binarios para macOS/Windows.

## Capabilities

### New Capabilities

_Ninguna._ El comportamiento nuevo es una evolución del pipeline de CI/release ya especificado.

### Modified Capabilities

- `ci-pipeline`: además del release disparado por tag manual, el pipeline MUST versionar con kli desde Conventional Commits en `main` y MUST publicar el Release (binarios + `SHA256SUMS`) en el mismo flujo autenticado con `GITHUB_TOKEN`.

## Impact

- `.github/workflows/ci.yml` y/o `.github/workflows/release.yml` (jobs de versionado, tag y publicación).
- Documentación: `README.md`, `docs/runbooks/install-systemd.md` (cómo aparecen los releases).
- Spec: `openspec/specs/ci-pipeline/spec.md` (delta en este change).
- Dependencia externa de runtime en CI: binario `kli` desde Releases de `KaribuLab/kli` (p. ej. `v0.4.2`).
- Sin impacto en tools MCP ni en el binario en runtime del usuario (solo en cómo se versiona y publica).
