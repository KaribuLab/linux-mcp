#!/bin/sh
# scan-bashisms.sh — defensa-en-profundidad sobre shellcheck -s sh.
#
# Grepea patrones comunes que rompen en dash (Debian/Ubuntu) y
# busybox ash (Alpine). NO pretende ser exhaustivo: el linter
# principal sigue siendo `shellcheck -s sh install.sh`.
#
# Uso:
#   ./scan-bashisms.sh install.sh
#
# Sale con 0 si no encuentra nada, 1 si encuentra algún patrón.
# Para uso en CI; el operador puede leer el output y decidir si
# un match es un falso positivo (ej. el patrón dentro de un
# comentario explicativo).

set -eu

FILE="${1:-install.sh}"

if [ ! -f "$FILE" ]; then
    printf 'uso: %s <archivo>\n' "$0" >&2
    exit 2
fi

failed=0

check() {
    pattern=$1
    desc=$2
    # grep -nE: -n muestra número de línea, -E regex extendida
    matches=$(grep -nE "$pattern" "$FILE" || true)
    if [ -n "$matches" ]; then
        printf 'BASHISM (%s):\n' "$desc"
        printf '%s\n' "$matches" | sed 's/^/  /'
        failed=1
    fi
}

check '\[\[' '[[ ]] (usar [ ])'
check 'pipefail' 'set -o pipefail (no POSIX)'
check '[[:space:]]local[[:space:]]|^local[[:space:]]' 'local (no POSIX, evita)'
# shellcheck disable=SC2016  # Queremos literal ${var,,} en el mensaje
check '\$\{[A-Za-z_][A-Za-z0-9_]*,,' '${var,,} lowercase expansion (bash 4+)'
# shellcheck disable=SC2016  # Queremos literal ${var^^} en el mensaje
check '\$\{[A-Za-z_][A-Za-z0-9_]*\^\^' '${var^^} uppercase expansion (bash 4+)'
check 'declare -[aAiA]' 'declare -a/-A/-i'
check 'printf -v' 'printf -v (asignación a variable, bash)'
check '<\(' 'process substitution <( )'
check '>\(' 'process substitution >( )'
check '&>' 'redirection &> (bash)'
check '\bmapfile\b|\breadarray\b' 'mapfile/readarray (bash 4+)'

if [ "$failed" -eq 1 ]; then
    printf '%s contiene bashisms. Verificá si son falsos positivos (comentarios) o arreglálos.\n' "$FILE" >&2
    exit 1
fi

printf 'OK: %s sin bashisms comunes\n' "$FILE"
exit 0