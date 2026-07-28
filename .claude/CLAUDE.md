# Reglas de trabajo para Claude Code en linux-mcp

Este repositorio usa dos protocolos para mantener contexto entre sesiones y analizar el código sin gastar tokens de más:

- [`rules/engram.md`](../rules/engram.md) — Cómo usar Engram para guardar y recuperar información importante del proyecto.
- [`rules/graphify.md`](../rules/graphify.md) — Cómo usar Graphify para analizar el código y reducir el consumo de tokens.

## Flujo estándar al iniciar una sesión

1. Lee este archivo y las reglas enlazadas.
2. Recupera contexto de Engram:
   - `mem_context` y `mem_search` con palabras clave del objetivo actual.
   - Busca especialmente el `topic_key` `project:linux-mcp:current-state`.
3. Si la pregunta es sobre arquitectura, relaciones entre archivos o flujo de datos, consulta Graphify primero (`graphify-out/graph.json` o `graphify query`).
4. Después de orientarte, trabaja en el código.

## Durante el trabajo

- Guarda en Engram cada decisión, convención, bug corregido o descubrimiento no obvio.
- Si inicias un cambio estructural, documenta el estado actual bajo `topic_key: project:linux-mcp:current-state`.
- No almacenes secretos (tokens, claves, credenciales) en Engram.
- Sigue el flujo OpenSpec del proyecto cuando el humano pida implementar un cambio; los artefactos de planeación viven en `openspec/`.

## Antes de cerrar la sesión

- Llama `mem_session_summary` con el resumen de la sesión.
- Si el grafo de Graphify quedó desactualizado tras cambios grandes, ejecuta `graphify --update .`.

## Idioma

El proyecto usa español neutro en documentación, commits y memoria.
