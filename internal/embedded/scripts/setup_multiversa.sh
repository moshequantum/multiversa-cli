#!/usr/bin/env bash
# MultiversaGroup private workspace setup
# SSH · GPG · git identity · private repos · ~/.multiversa/ · vault
# Usage: ./setup_multiversa.sh

set -euo pipefail

GIT_NAME="Moshe"
GIT_EMAIL="moshequantum@gmail.com"
GIT_HANDLE="moshequantum"
MULTIVERSA_DIR="$HOME/.multiversa"
VAULT_PATH="$MULTIVERSA_DIR/vault/secrets.enc"
PRIVATE_REPO="git@github.com:${GIT_HANDLE}/multiversagroup.git"
LOCAL_GROUP_PATH="$HOME/Documents/01_Multiversa/Group/repo"

# ── UI helpers ────────────────────────────────────────────────────────────────

separator() { printf '\n%*s\n\n' 60 '' | tr ' ' '━'; }
info()       { echo "  → $*"; }
success()    { echo "  ✓ $*"; }
warn()       { echo "  ⚠  $*"; }

separator
echo "  MultiversaGroup — Private Workspace Setup"
separator

# ── 1. SSH key ────────────────────────────────────────────────────────────────

SSH_KEY="$HOME/.ssh/id_ed25519_multiversa"

if [[ -f "$SSH_KEY" ]]; then
  success "SSH key already exists: $SSH_KEY"
else
  info "Generating Ed25519 SSH key for GitHub..."
  ssh-keygen -t ed25519 -C "$GIT_EMAIL" -f "$SSH_KEY" -N ""
  success "SSH key generated: $SSH_KEY"
fi

SSH_PUB="$(cat "${SSH_KEY}.pub")"
echo ""
echo "  ┌─ PUBLIC KEY (add to GitHub → Settings → SSH Keys) ─────────"
echo "  │"
echo "$SSH_PUB" | sed 's/^/  │  /'
echo "  │"
echo "  └─────────────────────────────────────────────────────────────"
echo ""

# Add to SSH agent
eval "$(ssh-agent -s)" &>/dev/null || true
ssh-add "$SSH_KEY" 2>/dev/null || true

# SSH config entry
SSH_CONFIG="$HOME/.ssh/config"
touch "$SSH_CONFIG"
chmod 600 "$SSH_CONFIG"

if ! grep -q "Host github.com" "$SSH_CONFIG" 2>/dev/null; then
  cat >> "$SSH_CONFIG" << EOF

Host github.com
  HostName github.com
  User git
  IdentityFile $SSH_KEY
  AddKeysToAgent yes
EOF
  success "SSH config updated for github.com"
fi

# ── 2. GPG key ────────────────────────────────────────────────────────────────

separator
echo "  2. GPG key for signed commits"
separator

EXISTING_GPG=$(gpg --list-secret-keys --keyid-format LONG "$GIT_EMAIL" 2>/dev/null | grep sec | head -1 | awk '{print $2}' | cut -d'/' -f2 || echo "")

if [[ -n "$EXISTING_GPG" ]]; then
  success "GPG key exists: $EXISTING_GPG"
  GPG_KEY_ID="$EXISTING_GPG"
else
  info "Generating GPG key for $GIT_EMAIL..."
  gpg --batch --gen-key << EOF
Key-Type: EdDSA
Key-Curve: Ed25519
Name-Real: $GIT_NAME
Name-Email: $GIT_EMAIL
Expire-Date: 2y
%no-passphrase
%commit
EOF
  GPG_KEY_ID=$(gpg --list-secret-keys --keyid-format LONG "$GIT_EMAIL" | grep sec | head -1 | awk '{print $2}' | cut -d'/' -f2)
  success "GPG key generated: $GPG_KEY_ID"
fi

echo ""
echo "  ┌─ PUBLIC GPG KEY (add to GitHub → Settings → GPG Keys) ──────"
gpg --armor --export "$GPG_KEY_ID" | sed 's/^/  │  /'
echo "  └──────────────────────────────────────────────────────────────"
echo ""

# ── 3. Git identity ───────────────────────────────────────────────────────────

separator
echo "  3. Git global identity"
separator

git config --global user.name  "$GIT_NAME"
git config --global user.email "$GIT_EMAIL"
git config --global user.signingkey "$GPG_KEY_ID"
git config --global commit.gpgsign true
git config --global init.defaultBranch main
git config --global pull.rebase true
git config --global core.editor "nvim"

success "Git configured: $GIT_NAME <$GIT_EMAIL>"
success "GPG signing: $GPG_KEY_ID"

# ── 4. ~/.multiversa/ structure ──────────────────────────────────────────────

separator
echo "  4. ~/.multiversa/ workspace structure"
separator

mkdir -p \
  "$MULTIVERSA_DIR/engram_db" \
  "$MULTIVERSA_DIR/config" \
  "$MULTIVERSA_DIR/vault" \
  "$MULTIVERSA_DIR/logs"

# Config file
CONF="$MULTIVERSA_DIR/config/multiversa.toml"
if [[ ! -f "$CONF" ]]; then
  cat > "$CONF" << TOML
# MultiversaGroup local config
[identity]
name    = "$GIT_NAME"
email   = "$GIT_EMAIL"
github  = "$GIT_HANDLE"

[paths]
group_repo   = "$LOCAL_GROUP_PATH"
engram_db    = "$MULTIVERSA_DIR/engram_db/context.db"
vault        = "$VAULT_PATH"

[stack]
pnpm = true
pkg_manager = "pnpm"
TOML
  success "Config created: $CONF"
else
  success "Config already exists: $CONF"
fi

# ── 5. Clone private repo ─────────────────────────────────────────────────────

separator
echo "  5. Clone moshequantum/multiversagroup"
separator

if [[ -d "$LOCAL_GROUP_PATH/.git" ]]; then
  success "Repo already cloned: $LOCAL_GROUP_PATH"
  info "Pulling latest..."
  git -C "$LOCAL_GROUP_PATH" pull --rebase || warn "Pull failed — may need manual resolution"
else
  info "Testing GitHub SSH connection..."
  if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
    info "Cloning $PRIVATE_REPO..."
    mkdir -p "$(dirname "$LOCAL_GROUP_PATH")"
    git clone "$PRIVATE_REPO" "$LOCAL_GROUP_PATH"
    success "Repo cloned: $LOCAL_GROUP_PATH"
  else
    warn "GitHub SSH auth not yet configured."
    warn "Add the public key above to GitHub, then run:"
    warn "  git clone $PRIVATE_REPO $LOCAL_GROUP_PATH"
  fi
fi

# ── 6. pnpm global packages ───────────────────────────────────────────────────

separator
echo "  6. Global pnpm packages"
separator

if command -v pnpm &>/dev/null; then
  pnpm add -g typescript tsx @sveltejs/kit svelte-check
  success "typescript · tsx · svelte-kit · svelte-check installed globally"
else
  warn "pnpm not found — run setup_stack.sh first, then re-run this script"
fi

# ── 7. .env.local template ────────────────────────────────────────────────────

separator
echo "  7. .env.local template"
separator

ENV_TEMPLATE="$MULTIVERSA_DIR/config/.env.local.template"
if [[ ! -f "$ENV_TEMPLATE" ]]; then
  cat > "$ENV_TEMPLATE" << 'ENV'
# MultiversaGroup — Local environment template
# Copy to project root as .env.local and fill in values
# NEVER commit this file — it is gitignored by default

# ── InsForge (Supabase-compatible) ─────────────────────────────────────────
INSFORGE_URL=
INSFORGE_ANON_KEY=
INSFORGE_SERVICE_ROLE_KEY=

# ── Manychat ───────────────────────────────────────────────────────────────
MANYCHAT_API_KEY=
MANYCHAT_PAGE_ID=

# ── Hotmart ────────────────────────────────────────────────────────────────
HOTMART_CLIENT_ID=
HOTMART_CLIENT_SECRET=
HOTMART_BASIC=

# ── OpenRouter (AI routing) ────────────────────────────────────────────────
OPENROUTER_API_KEY=

# ── Anthropic ─────────────────────────────────────────────────────────────
ANTHROPIC_API_KEY=
ENV
  success ".env.local template: $ENV_TEMPLATE"
fi

# ── 8. Encrypted vault ────────────────────────────────────────────────────────

separator
echo "  8. Secrets vault (OpenSSL AES-256)"
separator

if [[ -f "$VAULT_PATH" ]]; then
  success "Vault already exists: $VAULT_PATH"
else
  info "Creating encrypted secrets vault..."
  echo ""
  echo "  Enter a strong vault passphrase (different from LUKS):"
  echo ""
  # Create empty vault placeholder
  echo '# MultiversaGroup Secrets Vault — created '"$(date '+%Y-%m-%d')" \
    | openssl enc -aes-256-cbc -pbkdf2 -iter 100000 -out "$VAULT_PATH"
  success "Vault created: $VAULT_PATH"
  echo ""
  echo "  To decrypt: openssl enc -d -aes-256-cbc -pbkdf2 -iter 100000 -in $VAULT_PATH"
fi

# ── Summary ───────────────────────────────────────────────────────────────────

separator
echo "  MultiversaGroup workspace setup complete"
separator
echo ""
echo "  SSH key:    $SSH_KEY"
echo "  GPG key:    $GPG_KEY_ID"
echo "  Config:     $CONF"
echo "  Group repo: $LOCAL_GROUP_PATH"
echo "  Vault:      $VAULT_PATH"
echo ""
echo "  Pending manual steps:"
echo "  1. Add SSH public key to GitHub (shown above)"
echo "  2. Add GPG public key to GitHub (shown above)"
echo "  3. Copy .env.local.template → project .env.local and fill values"
echo ""
separator
