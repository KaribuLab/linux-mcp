## Context

Hoy existen dos workflows:

- `ci.yml` — en push/PR a `main`: tests, vet, govulncheck, build de ambas arch, artifacts 14 días.
- `release.yml` — solo en push de tag `v*`: build con `VERSION` del tag, `SHA256SUMS`, `gh release create`.

No hay tags en el repo → no hay Releases → la descarga documentada da 404.

KaribuLab ya versiona con [kli](https://github.com/KaribuLab/kli) (`kli semver` / `kli semver -t`) en otros repos. El humano rechazó Release Please (PAT) y prefiere kli + `GITHUB_TOKEN` del pipeline con `permissions` (`contents: write`, `id-token: write`).

Restricción GitHub: eventos generados con `GITHUB_TOKEN` no disparan otro workflow. Por eso el tag y la publicación del Release MUST ocurrir en el **mismo** workflow (o el tag se pushea con un token que sí dispare workflows; aquí evitamos eso).

## Goals / Non-Goals

**Goals:**

- Versionado semántico automático desde Conventional Commits en `main` vía kli.
- Publicar Release público (amd64, arm64, `SHA256SUMS`) sin PAT ni SSH deploy key.
- Mantener CI de verificación (tests/vet/unit) en PRs y en `main`.
- Inyectar la versión del tag en `--version` del binario publicado.

**Non-Goals:**

- Release Please / semantic-release.
- Binarios no-Linux.
- Cambios al servidor MCP o tools.
- Mirror a otros registros.

## Decisions

### D1: kli como calculadora de versión

- **Elección:** Descargar binario publicado de `KaribuLab/kli` (fijar tag, p. ej. `v0.4.2`) en el job; `kli semver` para la versión; `kli semver -t` para crear tags locales/remotos.
- **Alternativas:** Release Please (descartado: PAT), semantic-release (Node + más piezas), tag manual (estado actual, frágil).
- **Rationale:** Herramienta propia de la org; mismo convenio de commits que ya usa el repo.

### D2: Un solo flujo en `main` para tag + Release (anti-loop)

- **Elección:** En push a `main`, tras CI verde (o en un job que dependa de verify):
  1. Checkout `fetch-depth: 0`
  2. Bajar kli
  3. Comparar `kli semver` con `git describe --tags --abbrev=0` (vacío si no hay tags)
  4. Si difieren: `git config` bot → `kli semver -t` → `git push --tags` con `GITHUB_TOKEN`
  5. En el **mismo** job (o job siguiente del mismo run, sin esperar un workflow nuevo): `VERSION=<tag>` `go tool task build:all`, `sha256sum`, `gh release create <tag> dist/* --generate-notes`
- **Alternativas:** Workflow separado en `tags: v*` (falla con solo `GITHUB_TOKEN` por anti-loop); SSH deploy key (secreto extra, rechazado implícitamente al preferir permissions del token del pipeline).
- **Rationale:** Cumple “sin PAT” y publica el Release en el mismo run.

### D3: Destino del workflow de release actual

- **Elección:** Fusionar la lógica de publicación en el flujo de `main` (job `release` condicionado a “hay nueva versión”). Dejar de depender de `release.yml` solo-por-tag, o reducir `release.yml` a no-op / eliminarlo para no duplicar `gh release create`.
- **Alternativa:** Mantener `release.yml` para tags creados a mano fuera de Actions (poco frecuente). Preferible: un solo camino.
- **Rationale:** Una fuente de verdad evita releases duplicados o fallidos.

### D4: Permissions del job de release

```yaml
permissions:
  contents: write
  id-token: write
```

- `contents: write` — push de tags + crear Release + subir assets.
- `id-token: write` — alineado al patrón que el humano ya validó en otros pipelines KaribuLab; no bloquea si no se usa OIDC.
- Token: `secrets.GITHUB_TOKEN` (default del job). Sin secretos adicionales.

### D5: PRs y `main` sin bump

- PRs: solo verify + artifacts (como hoy); **no** crear tags ni Releases.
- Push a `main` donde `kli semver` == último tag: no crear Release (idempotente).
- Primer release (repo sin tags): `kli semver` calcula desde el historial; `kli semver -t` crea la cadena de tags que haga falta según la implementación de kli.

### D6: Documentación

- README y runbook: los Releases los genera el pipeline tras merge a `main` con Conventional Commits; listar tag real en [Releases](https://github.com/KaribuLab/linux-mcp/releases) antes de `curl`. Mantener `curl -fsSLO`.

Las rules de tools MCP no aplican (este change no toca tools).

## Risks / Trade-offs

- **[Risk]** Anti-loop mal entendido → tag sin Release. → **Mitigation:** publicar en el mismo workflow, nunca esperar un segundo run por el tag.
- **[Risk]** kli cambia formato de salida. → **Mitigation:** fijar versión de kli en la URL de descarga; fallar el job si el tag no matchea `^v[0-9]+\.[0-9]+\.[0-9]+$`.
- **[Risk]** `chore:` / `refactor:` pueden no bumpear según kli (solo feat/fix/breaking). → **Mitigation:** documentar; si no hay bump, no hay Release (correcto).
- **[Risk]** Carrera si dos pushes a `main` casi a la vez. → **Mitigation:** concurrency group `release-main` cancel-in-progress false / o `queue`.
- **[Trade-off]** Sin Release PR de revisión (a diferencia de Release Please). → Aceptable: el review ya ocurrió en el PR de código.

## Migration Plan

1. Aterrizar workflows + docs en `main`.
2. Primer push post-merge (o re-run) crea el primer tag/Release vía kli.
3. Verificar assets en Releases y `linux-mcp --version`.
4. Rollback: revertir el commit del workflow; borrar tag/Release malos a mano si hiciera falta.

## Open Questions

_Ninguna bloqueante._ Versión fija de kli en CI: usar `v0.4.2` (latest estable al diseñar) salvo que el humano pida otra.
