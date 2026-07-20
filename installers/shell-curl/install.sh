#!/usr/bin/env sh
# Multiversa CLI installer — "la IA propone, tú decides".
# Uso: curl -sSL https://raw.githubusercontent.com/moshequantum/multiversa-cli/main/installers/shell-curl/install.sh | sh
#
# El instalador se encarga de todo; tú solo respondes y/n a cada paso.
# Nada se instala sin tu confirmación. Honra $MULTIVERSA_VERSION,
# $MULTIVERSA_INSTALL_DIR y $MULTIVERSA_YES (=1 acepta todo, para CI).

set -eu

REPO="moshequantum/multiversa-cli"
INSTALL_DIR="${MULTIVERSA_INSTALL_DIR:-/usr/local/bin}"
VERSION="${MULTIVERSA_VERSION:-latest}"
ASSUME_YES="${MULTIVERSA_YES:-0}"

CHARTREUSE="\033[38;5;191m"
IVORY="\033[38;5;230m"
DIM="\033[2m"
WARN="\033[33m"
RESET="\033[0m"

say()    { printf "%b%s%b\n" "$IVORY" "$1" "$RESET"; }
accent() { printf "%b%s%b\n" "$CHARTREUSE" "$1" "$RESET"; }
dim()    { printf "%b%s%b\n" "$DIM" "$1" "$RESET"; }
warn()   { printf "%b%s%b\n" "$WARN" "$1" "$RESET"; }

# ask <pregunta> — y/n leído desde /dev/tty para funcionar bajo `curl | sh`.
# Sin TTY (CI, pipes): responde sí solo si MULTIVERSA_YES=1, si no, no.
HAS_TTY=0
if (exec < /dev/tty) 2>/dev/null; then HAS_TTY=1; fi

ask() {
  if [ "$ASSUME_YES" = "1" ]; then return 0; fi
  if [ "$HAS_TTY" != "1" ]; then
    dim "  (sin terminal interactiva — omito: $1)"
    return 1
  fi
  printf "%b%s [y/n] %b" "$CHARTREUSE" "$1" "$RESET"
  read -r answer < /dev/tty || return 1
  case "$answer" in y|Y|s|S|si|SI|yes) return 0 ;; *) return 1 ;; esac
}

detect_os() {
  case "$(uname -s)" in
    Darwin) echo darwin ;;
    Linux)  echo linux ;;
    *) echo "SO no soportado por este script: $(uname -s) — usa go install o los .zip de releases" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    arm64|aarch64) echo arm64 ;;
    *) echo "Arquitectura no soportada: $(uname -m)" >&2; exit 1 ;;
  esac
}

accent "· · ·  M U L T I V E R S A  · · ·"
say "Instalador del orquestador del stack agentic curado."
dim "  Atribución upstream: https://github.com/$REPO/blob/main/CREDITS.md"
dim "  Nada se instala sin tu confirmación."
echo

OS=$(detect_os)
ARCH=$(detect_arch)
say "Detectado: $OS/$ARCH"

# ── 1. Resolver versión y descargar ─────────────────────────────
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | head -n1 | cut -d'"' -f4)
fi
[ -n "$VERSION" ] || { echo "No pude resolver la última versión." >&2; exit 1; }
say "Versión: $VERSION"

ASSET="multiversa_${VERSION#v}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$VERSION"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

say "Descargando $ASSET…"
curl -fsSL "$BASE/$ASSET" -o "$TMP/$ASSET"

# Verificación de integridad contra checksums.txt del release.
say "Verificando checksum…"
curl -fsSL "$BASE/checksums.txt" -o "$TMP/checksums.txt"
(
  cd "$TMP"
  if command -v sha256sum >/dev/null 2>&1; then
    grep " $ASSET\$" checksums.txt | sha256sum -c - >/dev/null
  elif command -v shasum >/dev/null 2>&1; then
    grep " $ASSET\$" checksums.txt | shasum -a 256 -c - >/dev/null
  else
    echo "sin sha256sum/shasum — omito verificación" >&2
  fi
)

tar -xzf "$TMP/$ASSET" -C "$TMP"

if [ ! -w "$INSTALL_DIR" ]; then
  say "Instalando en $INSTALL_DIR (requiere sudo)…"
  sudo install -m 0755 "$TMP/multiversa" "$INSTALL_DIR/multiversa"
else
  say "Instalando en $INSTALL_DIR…"
  install -m 0755 "$TMP/multiversa" "$INSTALL_DIR/multiversa"
fi
accent "✓ multiversa $VERSION instalado en $INSTALL_DIR/multiversa"
echo

# ── 2. Prerequisitos de los motores (cada uno se pregunta) ──────
say "Los motores curados necesitan algunas herramientas base:"
dim "  brew → Engram, gentle-ai · pipx → Graphify · pnpm → codegraph, gentle-pi"
echo

if ! command -v brew >/dev/null 2>&1; then
  if ask "brew (Homebrew) no está — ¿lo instalo con su script oficial?"; then
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    # Activar brew en esta sesión (rutas estándar Linux y macOS).
    for BREW in /home/linuxbrew/.linuxbrew/bin/brew /opt/homebrew/bin/brew /usr/local/bin/brew; do
      [ -x "$BREW" ] && eval "$("$BREW" shellenv)" && break
    done
  else
    dim "  · brew omitido — Engram y gentle-ai quedarán pendientes."
  fi
fi

if ! command -v pipx >/dev/null 2>&1; then
  if ask "pipx no está — ¿lo instalo?"; then
    if command -v apt-get >/dev/null 2>&1; then
      sudo apt-get install -y pipx
    elif command -v brew >/dev/null 2>&1; then
      brew install pipx
    else
      python3 -m pip install --user pipx
    fi
  else
    dim "  · pipx omitido — Graphify quedará pendiente."
  fi
fi

if ! command -v pnpm >/dev/null 2>&1; then
  if ask "pnpm no está — ¿lo instalo con su script oficial standalone?"; then
    curl -fsSL https://get.pnpm.io/install.sh | sh -
  else
    dim "  · pnpm omitido — codegraph y gentle-pi quedarán pendientes."
  fi
fi
echo

# ── 3. Configuración guiada ─────────────────────────────────────
if ask "¿Corro el asistente de configuración ahora? (multiversa init)"; then
  "$INSTALL_DIR/multiversa" init < /dev/tty || true
fi

if ask "¿Activo el chequeo diario de actualizaciones? (cron 09:00, solo avisa — nunca actualiza solo)"; then
  "$INSTALL_DIR/multiversa" updates cron --apply || true
fi

echo
accent "Listo. Tu universo, orquestado."
dim "  multiversa tenant new <slug>   → crea tu primer OS (perfil aislado)"
dim "  multiversa detect              → lee tu entorno sin tocar nada"
dim "  multiversa credits             → los autores upstream que hacen esto posible"
