#!/bin/sh
# install.sh — instalador one-liner de linux-mcp (KaribuLab/linux-mcp).
#
# Uso recomendado:
#   curl -fsSL https://raw.githubusercontent.com/KaribuLab/linux-mcp/main/install.sh | sudo sh
#
# Override de versión (opcional):
#   LINUX_MCP_VERSION=v0.10.0 curl -fsSL ... | sudo sh
#
# Override de destino del binario (opcional):
#   LINUX_MCP_BINDIR=/opt sh -c 'curl ... | sudo env LINUX_MCP_BINDIR=/opt sh'
#
# POSIX sh (probado con dash, ash, bash --posix). Sin bashisms.
# shellcheck shell=sh

set -eu

readonly REPO="KaribuLab/linux-mcp"
readonly GITHUB_RELEASES="https://github.com/${REPO}/releases"
readonly GITHUB_RAW="https://raw.githubusercontent.com/${REPO}/main"
readonly UNIT_URL="${GITHUB_RAW}/deploy/systemd/linux-mcp.service"
readonly BINDIR_DEFAULT="/usr/local/bin"
readonly UNIT_PATH="/etc/systemd/system/linux-mcp.service"
readonly SOCKET_PATH="/run/linux-mcp/issue.sock"
readonly ENDPOINT="http://127.0.0.1:5000"

# Resueltas en tiempo de ejecución.
TMPDIR_INSTALL=""
ARCH=""
VERSION=""
BINDIR="${LINUX_MCP_BINDIR:-$BINDIR_DEFAULT}"

log() {
    printf '[install] %s\n' "$*" >&2
}

die() {
    log "ERROR: $*"
    exit 1
}

require_root() {
    if [ "$(id -u)" -ne 0 ]; then
        die "este instalador requiere root. Ejecutá: curl -fsSL <url> | sudo sh"
    fi
}

require_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        die "falta el comando '$1'. Instalalo antes de continuar."
    fi
}

require_prereqs() {
    for cmd in systemctl curl sha256sum getent mktemp install stat; do
        require_cmd "$cmd"
    done
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)
            ARCH=amd64
            ;;
        aarch64)
            ARCH=arm64
            ;;
        *)
            die "arquitectura no soportada: $(uname -m). Soportadas: amd64 (x86_64) y arm64 (aarch64, que cubre Raspberry Pi 3/4/5 con OS 64-bit). Para otras arquitecturas, compilá desde el repo."
            ;;
    esac
    log "arquitectura: ${ARCH}"
}

resolve_version() {
    if [ -n "${LINUX_MCP_VERSION:-}" ]; then
        case "$LINUX_MCP_VERSION" in
            v[0-9]*.[0-9]*.[0-9]*)
                VERSION="$LINUX_MCP_VERSION"
                log "versión (forzada por LINUX_MCP_VERSION): ${VERSION}"
                return 0
                ;;
            *)
                die "LINUX_MCP_VERSION debe matchear vX.Y.Z (recibido: '$LINUX_MCP_VERSION')"
                ;;
        esac
    fi

    # /releases/latest redirige a /releases/tag/<tag>; seguimos el redirect con
    # -w '%{url_effective}' y NO usamos la API REST (así no consumimos rate-limit).
    latest_url=$(
        curl -fsSL -o /dev/null -w '%{url_effective}' \
            "${GITHUB_RELEASES}/latest" 2>/dev/null || echo ""
    )
    if [ -z "$latest_url" ]; then
        die "no pude resolver la última release. Verificá la conectividad o fijá LINUX_MCP_VERSION=vX.Y.Z con un tag existente."
    fi

    VERSION=${latest_url##*/releases/tag/}
    if [ -z "$VERSION" ] || [ "$VERSION" = "$latest_url" ]; then
        die "no pude parsear la versión desde el redirect: ${latest_url}"
    fi
    log "versión (latest): ${VERSION}"
}

assert_version_exists() {
    # El endpoint /download/<tag>/SHA256SUMS devuelve 200 si el tag existe.
    http_code=$(
        curl -fsSL -o /dev/null -w '%{http_code}' \
            "${GITHUB_RELEASES}/download/${VERSION}/SHA256SUMS" 2>/dev/null \
            || echo "000"
    )
    if [ "$http_code" != "200" ]; then
        die "no existe el release ${VERSION} (HTTP ${http_code}). Fijá LINUX_MCP_VERSION con un tag válido de https://github.com/${REPO}/releases."
    fi
}

install_binary() {
    TMPDIR_INSTALL=$(mktemp -d) || die "no pude crear el tmpdir"
    trap_cleanup

    cd "$TMPDIR_INSTALL" || die "no pude entrar a ${TMPDIR_INSTALL}"

    log "descargando binario y SHA256SUMS para ${VERSION}/${ARCH}"
    curl -fsSLO "${GITHUB_RELEASES}/download/${VERSION}/linux-mcp-linux-${ARCH}" \
        || die "falló la descarga del binario"
    curl -fsSLO "${GITHUB_RELEASES}/download/${VERSION}/SHA256SUMS" \
        || die "falló la descarga de SHA256SUMS"

    log "validando SHA256"
    if ! sha256sum --ignore-missing -c SHA256SUMS; then
        die "SHA256 inválido. No se modificó ${BINDIR}/linux-mcp"
    fi

    log "instalando binario en ${BINDIR}/linux-mcp"
    install -m 0755 "linux-mcp-linux-${ARCH}" "${BINDIR}/linux-mcp" \
        || die "falló install del binario"

    expected="linux-mcp version ${VERSION}"
    reported=$("${BINDIR}/linux-mcp" --version 2>&1 || true)
    if [ "$reported" != "$expected" ]; then
        die "el binario instalado no reporta la versión esperada. Esperado: '${expected}'. Reportado: '${reported}'"
    fi
    log "binario OK: ${reported}"
}

provision_users() {
    if ! getent group mcp-admin >/dev/null 2>&1; then
        log "creando grupo mcp-admin"
        groupadd --system mcp-admin || die "no pude crear el grupo mcp-admin"
    fi
    if ! getent passwd mcp-agent >/dev/null 2>&1; then
        log "creando usuario mcp-agent"
        useradd --system --home /nonexistent --shell /usr/sbin/nologin mcp-agent \
            || die "no pude crear el usuario mcp-agent"
    fi
    getent group mcp-admin >/dev/null 2>&1 \
        || die "mcp-admin no existe y no pude crearlo; la unit no puede arrancar sin él"
}

install_unit() {
    log "descargando unit desde main"
    tmp_unit="${TMPDIR_INSTALL}/linux-mcp.service"
    curl -fsSL "${UNIT_URL}" -o "$tmp_unit" \
        || die "falló la descarga de la unit"

    log "instalando unit en ${UNIT_PATH}"
    install -m 0644 "$tmp_unit" "${UNIT_PATH}" \
        || die "falló install de la unit"

    log "systemctl daemon-reload"
    systemctl daemon-reload || die "daemon-reload falló"

    log "systemctl enable --now linux-mcp.service"
    systemctl enable --now linux-mcp.service || die "enable --now falló"

    if ! systemctl is-active --quiet linux-mcp.service; then
        die "el servicio no quedó activo. Revisá: journalctl -u linux-mcp.service"
    fi
    log "servicio linux-mcp.service activo"
}

verify_endpoint() {
    log "verificando endpoint (poll hasta 20s)"
    i=0
    while [ "$i" -lt 20 ]; do
        code=$(
            curl -sS -o /dev/null -w '%{http_code}' \
                --max-time 2 "$ENDPOINT" 2>/dev/null || echo "000"
        )
        if [ "$code" = "401" ]; then
            log "endpoint OK (401 — vivo y protegido)"
            return 0
        fi
        i=$((i + 1))
        sleep 1
    done
    die "el endpoint ${ENDPOINT} nunca devolvió 401 dentro de 20s. Revisá la sección Troubleshooting en docs/runbooks/install-systemd.md"
}

verify_socket() {
    log "verificando socket de emisión"
    if [ ! -S "$SOCKET_PATH" ]; then
        die "no existe el socket ${SOCKET_PATH}. Revisá la sección Troubleshooting en docs/runbooks/install-systemd.md"
    fi
    info=$(stat -c '%a %U %G' "$SOCKET_PATH")
    expected="660 mcp-agent mcp-admin"
    if [ "$info" != "$expected" ]; then
        die "socket con permisos/owner incorrectos. stat='${info}' esperado='${expected}'. Revisá la sección Troubleshooting en docs/runbooks/install-systemd.md"
    fi
    log "socket OK (${info})"
}

verify() {
    verify_endpoint
    verify_socket
    log "verificación final: OK"
}

cleanup() {
    if [ -n "${TMPDIR_INSTALL:-}" ] && [ -d "${TMPDIR_INSTALL:-}" ]; then
        rm -rf "$TMPDIR_INSTALL"
    fi
}

trap_cleanup() {
    # Idempotente: solo registra el trap la primera vez.
    trap 'cleanup' EXIT INT TERM
}

print_final_summary() {
    printf '%s\n' "${BINDIR}/linux-mcp"
    printf '%s\n' "${UNIT_PATH}"
}

main() {
    trap_cleanup
    log "instalador de linux-mcp"
    require_root
    require_prereqs
    detect_arch
    resolve_version
    assert_version_exists
    install_binary
    provision_users
    install_unit
    verify
    log "instalación completa. Para autorizar operadores: sudo usermod -aG mcp-admin <usuario> (debe cerrar sesión y volver a entrar)."
    print_final_summary
}

main "$@"