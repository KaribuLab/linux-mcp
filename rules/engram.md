# Uso estándar de Engram en linux-mcp

> Regla vinculada a [`AGENTS.md`](../AGENTS.md).

Engram es la memoria persistente del proyecto. Cualquier agente que trabaje en este repo puede recuperar contexto de sesiones anteriores y debe guardar lo relevante para las siguientes.

## Herramientas disponibles

- `mem_save` — guardar una observación.
- `mem_search` — buscar en memoria.
- `mem_context` — ver contexto reciente.
- `mem_get_observation` — leer una observación completa por ID.
- `mem_session_summary` — resumen de cierre de sesión.
- `mem_session_start` / `mem_session_end` — marcar inicio y fin.
- `mem_suggest_topic_key` / `mem_update` — actualizar un tema existente.

## Cuándo guardar (siempre de forma proactiva)

- Decisiones de arquitectura, diseño o trade-offs.
- Convenciones acordadas (nombres, estructura, patrones).
- Bugs corregidos (con causa raíz).
- Funcionalidades implementadas con enfoque no obvio.
- Descubrimientos, edge cases o comportamientos inesperados.
- Preferencias o restricciones del humano.
- Cambios de configuración o entorno.

## Formato de `mem_save`

```markdown
title: "Verbo + qué"          # ej. "Decidimos usar omitzero para time.Time"
type: decision                 # bugfix | decision | architecture | discovery | pattern | config | preference
scope: project                 # project (default) o personal
topic_key: project:linux-mcp:<tema>  # clave estable para temas que evolucionan
content: |
  **What**: Una oración de lo que se hizo.
  **Why**: Motivación (petición del humano, bug, rendimiento, etc.).
  **Where**: Archivos o rutas afectadas.
  **Learned**: Gotchas, edge cases o sorpresas (omitir si no hay).
```

## Claves de tema recomendadas

Usa estas claves para que todos los agentes encuentren el mismo contexto:

- `project:linux-mcp:current-state` — estado actual del proyecto: cambio activo, commit reciente, próximos pasos.
- `project:linux-mcp:architecture` — decisiones de arquitectura.
- `project:linux-mcp:conventions` — convenciones del proyecto.
- `project:linux-mcp:bugs` — bugs y correcciones.
- `project:linux-mcp:openspec:<change-id>` — cambios OpenSpec concretos.
- `project:linux-mcp:graphify` — estado/actualización del grafo.
- `project:linux-mcp:security` — decisiones de seguridad.

## Flujo de sesión

1. **Inicio**: `mem_session_start(intent="...")` y luego `mem_search`/`mem_context` sobre el objetivo.
2. **Trabajo**: guarda decisiones y descubrimientos con `topic_key` adecuado.
3. **Cierre**: `mem_session_summary` con el formato exacto del campo `content`.

## Qué no guardar

- Secretos, tokens, claves API, credenciales ni datos sensibles.
- Información efímera o que ya está en el diff.
- Duplicados: si un tema evoluciona, reusa el mismo `topic_key` para actualizarlo.
