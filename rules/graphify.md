# Uso de Graphify para analizar el código

> Regla vinculada a [`AGENTS.md`](../AGENTS.md).

Graphify convierte el repo en un grafo de conocimiento consultable. La idea es evitar leer archivos de más cuando la pregunta es sobre estructura, relaciones o flujo.

## Cuándo usar Graphify

- Si existe `graphify-out/graph.json`, prefírelo para preguntas sobre:
  - Arquitectura general.
  - "¿Quién llama a X?", "¿Cómo funciona Y?", "¿Cuál es el flujo de Z?".
  - Relaciones entre archivos, funciones, comunidades.
- Cuando el repo tenga ~100+ archivos o contenido mixto (código, docs, PDFs), Graphify reduce el consumo de tokens.
- Para repositorios pequeños (<~50 archivos) o consultas puntuales a un solo archivo, una lectura directa o `grep` es más barato.

## Cómo consultar

```bash
# Pregunta en lenguaje natural
graphify query "¿cómo se valida el token en linux-mcp?"

# Traza un camino entre dos conceptos
graphify path "AuthHandler" "TokenVerifier"

# Explica un nodo
graphify explain "SO_PEERCRED"
```

Si el CLI no está disponible, puedes recorrer `graphify-out/graph.json` con NetworkX.

## Antes de leer archivos

1. Revisa `graphify-out/GRAPH_REPORT.md`: busca God Nodes, Surprising Connections y Suggested Questions.
2. Identifica la comunidad o los nodos relevantes.
3. Lee solo los archivos que el grafo señale, citando `source_location` cuando cites un hecho.

## Mantener el grafo actualizado

- Después de cambios grandes, ejecuta `graphify --update .`.
- Si el repo crece mucho (>5.000 archivos), evalúa indexar por subpaquete en vez de todo el repo.
- El reporte `GRAPH_REPORT.md` y el HTML en `graphify-out/graph.html` son la referencia visual.

## Honestidad

- No inventes aristas. Si no estás seguro, usa `AMBIGUOUS`.
- Muestra el costo en tokens y advierte si el grafo supera 5.000 nodos.
- No ejecutes el pipeline completo sin necesidad; usa `--update` para cambios incrementales.
