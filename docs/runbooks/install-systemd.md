# Instalar linux-mcp con systemd

Runbook para desplegar el servidor MCP como servicio systemd con usuario `mcp-agent` y `CAP_DAC_READ_SEARCH`.

La **policy de lectura en el proceso Go es obligatoria** aunque no uses systemd: la unit solo añade defensa en profundidad (`InaccessiblePaths`, hardening de escritura). El binario puede correr a mano sin esta unit.

## Requisitos

- Linux con systemd
- Go 1.26+ (para build) o un binario ya compilado
- Privilegios root para crear usuario, instalar unit y capability ambient

## 1. Crear usuario y grupo

```bash
sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin mcp-agent
```

## 2. Build e instalar el binario

Desde el repo:

```bash
go tool task build
sudo install -m 0755 dist/linux-mcp-$(go env GOOS)-$(go env GOARCH) /usr/local/bin/linux-mcp
```

Sin Task:

```bash
go build -o /tmp/linux-mcp ./cmd/linux-mcp
sudo install -m 0755 /tmp/linux-mcp /usr/local/bin/linux-mcp
```

## 3. Instalar y habilitar la unit

```bash
sudo install -m 0644 deploy/systemd/linux-mcp.service /etc/systemd/system/linux-mcp.service
sudo systemctl daemon-reload
sudo systemctl enable --now linux-mcp.service
```

## 4. Verificar

```bash
systemctl status linux-mcp.service
curl -sS -o /dev/null -w '%{http_code}\n' http://localhost:5000
journalctl -u linux-mcp.service -n 50 --no-pager
```

El servicio escucha en `localhost:5000` (Streamable HTTP).

## 5. Actualizar

```bash
go tool task build
sudo install -m 0755 dist/linux-mcp-$(go env GOOS)-$(go env GOARCH) /usr/local/bin/linux-mcp
sudo systemctl restart linux-mcp.service
```

Si cambió la unit:

```bash
sudo install -m 0644 deploy/systemd/linux-mcp.service /etc/systemd/system/linux-mcp.service
sudo systemctl daemon-reload
sudo systemctl restart linux-mcp.service
```

## 6. Desinstalar

```bash
sudo systemctl disable --now linux-mcp.service
sudo rm -f /etc/systemd/system/linux-mcp.service
sudo systemctl daemon-reload
sudo rm -f /usr/local/bin/linux-mcp
# opcional: sudo userdel mcp-agent
```

## Troubleshooting

| Síntoma | Qué revisar |
|---------|-------------|
| `permission denied` al leer configs de sistema | `CAP_DAC_READ_SEARCH` en la unit; `getpcaps` / `systemctl show linux-mcp` |
| Servicio no arranca | `journalctl -u linux-mcp`; binario en `/usr/local/bin/linux-mcp`; usuario `mcp-agent` existe |
| `cat`/`list` bloquean paths | Esperado: denylist en app (`/etc/shadow`, keys, etc.). Systemd `InaccessiblePaths` es complemento |
| Inspector no conecta | CORS ya va en el handler; URL `http://localhost:5000`; el servicio solo escucha localhost |

## Referencias

- Unit: [`deploy/systemd/linux-mcp.service`](../../deploy/systemd/linux-mcp.service)
- Tools: [`docs/tools/cat.md`](../tools/cat.md), [`docs/tools/list.md`](../tools/list.md)
