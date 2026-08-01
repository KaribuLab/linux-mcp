## 1. Workflow de versionado y release

- [x] 1.1 Extender `.github/workflows/ci.yml` (o añadir job en el mismo archivo): en push a `main`, tras `verify`, descargar kli `v0.4.2`, comparar `kli semver` con el último tag, y exponer output `should_release` + `version`
- [x] 1.2 Job `release` condicionado a `should_release`: `permissions: contents: write` e `id-token: write`; `kli semver -t`; push de tags con `GITHUB_TOKEN`; `VERSION=<tag> go tool task build:all`; verificar `--version`; generar `SHA256SUMS`; `gh release create` con `dist/*`
- [x] 1.3 Añadir `concurrency` en el flujo de release en `main` para evitar carreras
- [x] 1.4 Eliminar o vaciar `.github/workflows/release.yml` para no duplicar `gh release create` en push de tags (una sola fuente de verdad)

## 2. Documentación

- [x] 2.1 Actualizar `README.md`: Releases se generan al mergear a `main` vía kli + Conventional Commits; mantener ejemplo de descarga con `curl -fsSLO` y tag real
- [x] 2.2 Actualizar `docs/runbooks/install-systemd.md` con la misma nota de origen de Releases
- [x] 2.3 Tras el primer Release del pipeline, fijar `VERSION` en los ejemplos de descarga de `README.md` y `docs/runbooks/install-systemd.md` a la última versión publicada (`gh release view --json tagName`)

## 3. Validación

- [x] 3.1 Validar YAML de workflows (sintaxis / `actionlint` si está disponible, o revisión manual de jobs/condiciones)
- [x] 3.2 `openspec validate kli-semver-release --strict`
- [x] 3.3 Verificar con `gh` que el run de CI en `main` concluyó OK y que existe un Release con amd64, arm64 y `SHA256SUMS`
