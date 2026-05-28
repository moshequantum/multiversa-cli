#!/usr/bin/env bash
# Multiversa Restore
# Backs up your Multiversa state, installs prerequisites, clones repos,
# and launches the lab wizard to configure your full stack.
#
# Works on: macOS (native) + Linux (including USB-bootable environments)
# Usage:
#   ./scripts/restore.sh              # interactive live run
#   ./scripts/restore.sh --dry-run    # preview without side effects
#   curl -sSL https://raw.githubusercontent.com/moshequantum/multiversa-cli/main/scripts/restore.sh | bash

set -euo pipefail

# ── Config ─────────────────────────────────────────────────────────────────────
MULTIVERSA_HOME="${MULTIVERSA_HOME:-$HOME/.multiversa}"
WORKSPACE_DIR="${MULTIVERSA_WORKSPACE:-$HOME/Documents/01_Multiversa}"
BACKUP_DIR="${MULTIVERSA_HOME}/backups/$(date +%Y%m%d_%H%M%S)"
CLI_REPO="https://github.com/moshequantum/multiversa-cli.git"
LAB_REPO="https://github.com/moshequantum/multiversalab.git"
GO_MIN_VERSION="1.22"
DRY_RUN=false

# ── UI ─────────────────────────────────────────────────────────────────────────
BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
DIM='\033[2m'
RESET='\033[0m'

log()    { echo -e "${CYAN}▸${RESET} $*"; }
ok()     { echo -e "${GREEN}✓${RESET} $*"; }
warn()   { echo -e "${YELLOW}⚠${RESET} $*"; }
err()    { echo -e "${RED}✗${RESET} $*" >&2; exit 1; }
section(){ echo; echo -e "${BOLD}── $* ${DIM}────────────────────────────────────────${RESET}"; echo; }
run()    { if $DRY_RUN; then echo -e "  ${DIM}[dry-run]${RESET} $*"; else eval "$*"; fi; }

# ── Args ───────────────────────────────────────────────────────────────────────
for arg in "$@"; do
  case "$arg" in
    --dry-run)   DRY_RUN=true ;;
    --help|-h)
      echo "Usage: restore.sh [--dry-run]"
      echo
      echo "  --dry-run   Preview all steps without making any changes."
      echo
      echo "Environment overrides:"
      echo "  MULTIVERSA_HOME       Default: ~/.multiversa"
      echo "  MULTIVERSA_WORKSPACE  Default: ~/Documents/01_Multiversa"
      exit 0
      ;;
    *) warn "Unknown argument: $arg" ;;
  esac
done

# ── OS / Arch ──────────────────────────────────────────────────────────────────
OS=""
case "$(uname -s)" in
  Darwin) OS="macos" ;;
  Linux)  OS="linux" ;;
  *)      err "Unsupported OS: $(uname -s)" ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  GO_ARCH="amd64" ;;
  aarch64|arm64) GO_ARCH="arm64" ;;
  *)        GO_ARCH="$ARCH" ;;
esac

# ── Banner ─────────────────────────────────────────────────────────────────────
echo
echo -e "${BOLD}  ╔═══════════════════════════════╗${RESET}"
echo -e "${BOLD}  ║   Multiversa Restore v0.4     ║${RESET}"
echo -e "${BOLD}  ╚═══════════════════════════════╝${RESET}"
echo
echo -e "  ${CYAN}Platform:${RESET}  ${OS} / ${ARCH}"
echo -e "  ${CYAN}Workspace:${RESET} ${WORKSPACE_DIR}"
echo -e "  ${CYAN}Mode:${RESET}      $($DRY_RUN && echo "${YELLOW}dry-run (no changes)${RESET}" || echo "live")"
echo

# ── Step 1: Backup ─────────────────────────────────────────────────────────────
section "Backup"

backup() {
  log "Creating backup at ${BACKUP_DIR}..."
  run "mkdir -p '${BACKUP_DIR}'"

  if [ -f "${MULTIVERSA_HOME}/profile.toml" ]; then
    run "cp '${MULTIVERSA_HOME}/profile.toml' '${BACKUP_DIR}/'"
    ok "profile.toml"
  else
    warn "No profile.toml found — fresh install assumed"
  fi

  if [ -d "${HOME}/.multiversa" ]; then
    run "cp -r '${HOME}/.multiversa/stacks' '${BACKUP_DIR}/stacks' 2>/dev/null || true"
    ok ".multiversa/stacks"
  fi

  # Claude Code memory — the brain behind the sessions
  if [ -d "${HOME}/.claude/projects" ]; then
    run "cp -rp '${HOME}/.claude/projects' '${BACKUP_DIR}/claude_projects'"
    ok "Claude project memory"
  fi
  if [ -f "${HOME}/.claude/settings.json" ]; then
    run "cp '${HOME}/.claude/settings.json' '${BACKUP_DIR}/claude_settings.json'"
    ok "Claude settings"
  fi
  if [ -f "${HOME}/.claude/CLAUDE.md" ]; then
    run "cp '${HOME}/.claude/CLAUDE.md' '${BACKUP_DIR}/CLAUDE.md'"
    ok "Global CLAUDE.md"
  fi

  # SSH keys — critical for GitHub access on fresh machine
  if [ -d "${HOME}/.ssh" ]; then
    run "cp -r '${HOME}/.ssh' '${BACKUP_DIR}/ssh'"
    ok "SSH keys (~/.ssh)"
  else
    warn "No SSH keys found — you will need to configure GitHub access manually"
  fi

  # Git config
  if [ -f "${HOME}/.gitconfig" ]; then
    run "cp '${HOME}/.gitconfig' '${BACKUP_DIR}/.gitconfig'"
    ok ".gitconfig"
  fi

  ok "Backup complete → ${BACKUP_DIR}"
}

backup

# ── Step 2: Prerequisites ──────────────────────────────────────────────────────
section "Prerequisites"

need_git() {
  if command -v git &>/dev/null; then
    ok "git $(git --version | awk '{print $3}')"
    return
  fi
  log "Installing git..."
  case "$OS" in
    macos)
      command -v xcode-select &>/dev/null && run "xcode-select --install" || run "brew install git"
      ;;
    linux)
      if command -v apt-get &>/dev/null; then
        run "sudo apt-get update -qq && sudo apt-get install -y git"
      elif command -v dnf &>/dev/null; then
        run "sudo dnf install -y git"
      elif command -v pacman &>/dev/null; then
        run "sudo pacman -S --noconfirm git"
      else
        err "Cannot install git — package manager not recognized"
      fi
      ;;
  esac
}

need_go() {
  if command -v go &>/dev/null; then
    local v
    v=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)
    ok "go ${v}"
    return
  fi

  log "Installing Go (${GO_MIN_VERSION}+)..."
  case "$OS" in
    macos)
      if command -v brew &>/dev/null; then
        run "brew install go"
      else
        err "Homebrew not found. Install Homebrew first: https://brew.sh"
      fi
      ;;
    linux)
      local GO_VER="1.23.4"
      local TARBALL="go${GO_VER}.linux-${GO_ARCH}.tar.gz"
      log "Downloading ${TARBALL}..."
      run "curl -sSL 'https://go.dev/dl/${TARBALL}' -o /tmp/${TARBALL}"
      run "sudo rm -rf /usr/local/go"
      run "sudo tar -C /usr/local -xzf /tmp/${TARBALL}"
      run "rm /tmp/${TARBALL}"
      # Add to PATH for current shell session
      export PATH="$PATH:/usr/local/go/bin"
      ok "Go ${GO_VER} installed at /usr/local/go"
      warn "Add to your shell profile: export PATH=\$PATH:/usr/local/go/bin"
      ;;
  esac
}

need_pnpm() {
  if command -v pnpm &>/dev/null; then
    ok "pnpm $(pnpm --version)"
    return
  fi
  log "Installing pnpm..."
  run "curl -fsSL https://get.pnpm.io/install.sh | sh -"
  # Source pnpm environment for the current session
  export PNPM_HOME="${HOME}/.local/share/pnpm"
  export PATH="$PNPM_HOME:$PATH"
  ok "pnpm installed"
}

need_git
need_go
need_pnpm

# ── Step 3: Multiversa CLI ─────────────────────────────────────────────────────
section "Multiversa CLI"

install_cli() {
  log "Installing latest Multiversa CLI..."
  local gobin
  gobin="$(go env GOPATH)/bin"
  run "go install github.com/moshequantum/multiversa-cli/cmd/multiversa@latest"

  # Ensure GOPATH/bin is in PATH
  if ! echo "$PATH" | grep -q "$gobin"; then
    export PATH="$PATH:$gobin"
    warn "Add to your shell profile: export PATH=\$PATH:${gobin}"
  fi

  if command -v multiversa &>/dev/null; then
    ok "multiversa $(multiversa version 2>/dev/null || echo 'installed')"
  else
    warn "multiversa installed to ${gobin} but not yet in \$PATH — restart your shell"
  fi
}

install_cli

# ── Step 4: Clone repositories ────────────────────────────────────────────────
section "Repositories"

clone_repo() {
  local name="$1"
  local url="$2"
  local dest="$3"

  if [ -d "${dest}/.git" ]; then
    log "${name}: already cloned — pulling latest..."
    run "git -C '${dest}' pull --ff-only"
    ok "${name} up to date"
  else
    log "Cloning ${name}..."
    run "mkdir -p '$(dirname "${dest}")'"
    run "git clone '${url}' '${dest}'"
    ok "${name} → ${dest}"
  fi
}

clone_repo "multiversa-cli" "$CLI_REPO" "${WORKSPACE_DIR}/Shared/multiversa-cli"
clone_repo "multiversalab"   "$LAB_REPO"  "${WORKSPACE_DIR}/Lab/repo"

# ── Step 5: Lab wizard ────────────────────────────────────────────────────────
section "Lab Setup"

run_lab() {
  if ! command -v multiversa &>/dev/null; then
    warn "multiversa not found in PATH — skipping lab wizard"
    warn "After adding GOPATH/bin to PATH, run: multiversa lab"
    return
  fi

  log "Launching multiversa lab..."
  if $DRY_RUN; then
    echo -e "  ${DIM}[dry-run] multiversa lab${RESET}"
  else
    multiversa lab
  fi
}

run_lab

# ── Done ──────────────────────────────────────────────────────────────────────
echo
echo -e "${GREEN}${BOLD}  Restore complete.${RESET}"
echo
echo -e "  ${CYAN}Backup:${RESET}    ${BACKUP_DIR}"
echo -e "  ${CYAN}Workspace:${RESET} ${WORKSPACE_DIR}"
echo
$DRY_RUN && echo -e "  ${YELLOW}This was a dry-run. No changes were made.${RESET}" && echo
