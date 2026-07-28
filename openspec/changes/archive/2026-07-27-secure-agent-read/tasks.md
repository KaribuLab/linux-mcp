## 1. Policy + toolmeta

- [x] 1.1 Crear `internal/policy` con denylist de paths, chequeo de tipos peligrosos y helpers de resolución/absoluto
- [x] 1.2 Implementar sniff de private key sobre primera línea útil (PEM/OpenSSH/PGP/PPK) + tests unitarios de allow/deny/sniff
- [x] 1.3 Exponer helpers de lectura acotada (stream, max lines ∩ max bytes, detección NUL) reutilizables por tools
- [x] 1.4 Crear `internal/toolmeta`: headers `Stringer` (`CatHeader`, `ListHeader`, `Blocked`) + ensamblado con `strings.Builder` (tests de formato)

## 2. Tool `cat`

- [x] 2.1 Reescribir `CatFile` para usar `internal/policy` + `toolmeta` (body en `strings.Builder`, meta/`blocked` tipados; caps 100 líneas ∩ 64 KiB; sin cache full-file)
- [x] 2.2 Añadir arg `offset` (byte) + meta `next` (byte); resume con `Seek` en fd seekable (tests de 2ª página sin skip-líneas desde 0)
- [x] 2.3 Actualizar `mcp.Tool.Description` de `cat` con contrato completo: meta `[cat …]` + body texto crudo, caps, `next`/`offset` byte, `[blocked class=… path=…]`
- [x] 2.4 Añadir tests de `cat` (archivo chico, truncado por líneas/bytes, resume Seek, PEM bloqueado, path deny, binario)
- [x] 2.5 Actualizar `docs/tools/cat.md` (y `docs/README.md` si el resumen cambia) alineado al comportamiento real y a la Description MCP

## 3. Tool `list`

- [x] 3.1 Aplicar path policy + cap 1000 entradas; tabla en `strings.Builder` + `ListHeader`/`Blocked` vía `toolmeta`; corregir `Readlink` con `filepath.Join` y lookup de grupo por GID
- [x] 3.2 Actualizar `mcp.Tool.Description` de `list` con contrato completo: meta `[list …]` + **tabla markdown** (modos simple/detallado), truncado, `[blocked …]`
- [x] 3.3 Añadir tests de `list` (deny path, meta + truncado, symlink con CWD distinto, grupo)
- [x] 3.4 Actualizar `docs/tools/list.md` (y `docs/README.md` si aplica) alineado al comportamiento real y a la Description MCP

## 4. Systemd deploy

- [x] 4.1 Añadir `deploy/systemd/linux-mcp.service` (user `mcp-agent`, `CAP_DAC_READ_SEARCH`, hardening escritura)
- [x] 4.2 Añadir `docs/runbooks/install-systemd.md` (usuario OS, binario, enable, update, uninstall, fallos)
- [x] 4.3 Enlazar runbook desde `README.md` y `docs/README.md`

## 5. Verificación

- [x] 5.1 Compilar y pasar tests (`go test ./...`)
- [x] 5.2 Smoke manual breve (archivo normal, shadow/PEM bloqueados, dir grande) con el servidor local si está disponible
