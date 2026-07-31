# Tool: `ps`

Lista procesos vía `/proc` (nunca ejecuta el binario `ps` ni shells). Meta + tabla markdown acotada (máx. 1000 filas).

| Campo | Valor |
|-------|-------|
| Nombre MCP | `ps` |
| Código | `internal/tool/ps.go` |
| Registro | `AddPsTool` → `NewHandler` |
| Descripción MCP | Contrato completo en `tool.PsToolDescription` |

## Parámetros

| Nombre | Tipo | Requerido | Descripción |
|--------|------|-----------|-------------|
| `includeKernel` | `bool` | no | Si `true`, incluye kernel threads (cmdline vacío). Default `false` |
| `showPpid` | `bool` (default `true`) | no | Columna `Ppid` |
| `showUser` | `bool` (default `true`) | no | Columna `User` |
| `showStat` | `bool` (default `true`) | no | Columna `Stat` |
| `showCpu` | `bool` (default `true`) | no | Columna `Cpu` (placeholder `-` en v1) |
| `showMem` | `bool` (default `true`) | no | Columna `Mem` = RSS en KiB |
| `showCmdline` | `bool` (default `true`) | no | Columna `Cmdline` (máx. 256 chars + `…`) |
| `showExe` | `bool` (default `true`) | no | Columna `Exe` (realpath; máx. 256 chars + `…`) |

`Pid` y `Comm` siempre visibles. Orden fijo. Meta incluye `columns=`.

## Respuesta

```text
[ps entries=<returned>/<total> truncated=<bool> columns=...]
|Pid|Comm|...
```

## Prompt de ejemplo (agente)

```text
Usa el tool linux-mcp `ps` para listar procesos de usuario (sin kernel threads) ocultando Cmdline y Exe, y dime qué procesos consumen más memoria según la columna Mem.
```

## Notas

- Fuente: solo lectura de `/proc`. Sin `os/exec`.
- Relacionada: [`ps_grep.md`](ps_grep.md).
