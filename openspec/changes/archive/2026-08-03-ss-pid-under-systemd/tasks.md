## 1. Unit systemd

- [x] 1.1 Actualizar `deploy/systemd/linux-mcp.service`: añadir `CAP_SYS_PTRACE` a `AmbientCapabilities` y `CapabilityBoundingSet` (mantener `CAP_DAC_READ_SEARCH`); eliminar `ProtectProc=invisible`
- [x] 1.2 Revisar comentarios de la unit para explicar por qué están esas caps (resolución Pid de sockets) sin mencionar secretos

## 2. Runbook e índice

- [x] 2.1 Actualizar `docs/runbooks/install-systemd.md`: documentar `CAP_SYS_PTRACE`, ausencia de `ProtectProc=invisible`, riesgo Bearer → inventario de dueños de sockets, paso de verificación Pid en listener ajeno, nota de update de unit
- [x] 2.2 Comprobar que `docs/README.md` / `README.md` siguen enlazando el runbook (ajustar solo si el texto de hardening quedó obsoleto)

## 3. Docs de tools ss / ss_grep

- [x] 3.1 Actualizar `docs/tools/ss.md`: expectativa Pid/Process bajo unit de referencia; resolución inode/`/proc/*/fd`; sección `## Prompt de ejemplo (agente)` con prompt que pida dueño (Pid) de un puerto vía `ss`
- [x] 3.2 Actualizar `docs/tools/ss_grep.md`: misma expectativa de Pid; prompt de ejemplo que filtre por puerto/dirección y use identidad de proceso
- [x] 3.3 Si el texto de `SsToolDescription` / `SsGrepToolDescription` en `internal/tool` contradice el deploy de referencia, alinearlo (sin cambiar schema JSON)

## 4. Verificación

- [x] 4.1 Ejecutar tests existentes de `ss`/`ss_grep` (`go test` del paquete) y confirmar que siguen verdes
- [x] 4.2 Checklist manual (o nota en runbook ya cubierta en 2.1): con unit nueva instalada, `ss`/`ss_grep` muestran Pid no vacío en al menos un LISTEN de uid ≠ mcp-agent cuando exista en el host
