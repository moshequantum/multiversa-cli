#!/usr/bin/env bash
# Encrypted bootable Linux USB — from macOS
# Guides creation of a LUKS-encrypted bootable Ubuntu 24.04 USB
# Usage: ./encrypted_usb_macos.sh

set -euo pipefail

UBUNTU_VERSION="24.04.4"
UBUNTU_ISO="ubuntu-${UBUNTU_VERSION}-desktop-amd64.iso"
UBUNTU_URL="https://releases.ubuntu.com/${UBUNTU_VERSION%%.*}/${UBUNTU_ISO}"

# ── UI helpers ────────────────────────────────────────────────────────────────

separator() { printf '\n%*s\n\n' 60 '' | tr ' ' '━'; }
prompt()     { read -r -p "  → $1: " REPLY; echo "$REPLY"; }
confirm()    { read -r -p "  → $1 [y/N]: " REPLY; [[ "${REPLY,,}" == "y" ]]; }

# ── Step 1: Identify USB ──────────────────────────────────────────────────────

separator
echo "  STEP 1 — Identify your USB drive"
separator

echo "  Connected disks:"
echo ""
diskutil list | grep -E "^/dev|USB|FDisk|GUID" | sed 's/^/    /'
echo ""
echo "  ⚠  CRITICAL: Identify the correct disk. Wrong disk = data loss."
echo "     USB drives typically show as /dev/disk2 or /dev/disk3."
echo "     Look for the disk matching your USB size."
echo ""

USB_DISK=$(prompt "Enter USB device (e.g. /dev/disk2)")

if [[ ! "$USB_DISK" =~ ^/dev/disk[0-9]+$ ]]; then
  echo "ERROR: Invalid device path. Must be /dev/diskN" >&2
  exit 1
fi

# Show disk info
echo ""
echo "  Disk info for $USB_DISK:"
diskutil info "$USB_DISK" | grep -E "Device|Size|Media|Partition" | sed 's/^/    /'
echo ""

if ! confirm "Is this the correct USB drive? (THIS WILL BE WIPED)"; then
  echo "Aborted. Nothing was changed."
  exit 0
fi

# ── Step 2: Download Ubuntu ISO ───────────────────────────────────────────────

separator
echo "  STEP 2 — Ubuntu 24.04 LTS ISO"
separator

ISO_PATH="$HOME/Downloads/$UBUNTU_ISO"

if [[ -f "$ISO_PATH" ]]; then
  echo "  ✓ ISO already downloaded: $ISO_PATH"
else
  echo "  Downloading Ubuntu $UBUNTU_VERSION..."
  echo "  URL: $UBUNTU_URL"
  echo ""
  curl -fL --progress-bar "$UBUNTU_URL" -o "$ISO_PATH"
  echo "  ✓ Downloaded: $ISO_PATH"
fi

# ── Step 3: Write ISO to USB ──────────────────────────────────────────────────

separator
echo "  STEP 3 — Write ISO to USB"
separator

echo "  This uses 'dd' to write the bootable installer to $USB_DISK."
echo "  ALL DATA ON $USB_DISK WILL BE PERMANENTLY ERASED."
echo ""

if ! confirm "Confirm: wipe $USB_DISK and write Ubuntu installer"; then
  echo "Aborted. Nothing was changed."
  exit 0
fi

echo "  → Unmounting $USB_DISK..."
diskutil unmountDisk "$USB_DISK"

RAW_DISK="${USB_DISK/disk/rdisk}"  # Use raw device for speed
echo "  → Writing ISO (this takes 5-10 minutes)..."
sudo dd if="$ISO_PATH" of="$RAW_DISK" bs=4m status=progress
sync
echo "  ✓ ISO written to USB."

# ── Step 4: Post-install script ───────────────────────────────────────────────

separator
echo "  STEP 4 — Generate post-install script"
separator

POSTINSTALL="$HOME/Downloads/multiversa_postinstall.sh"

cat > "$POSTINSTALL" << 'POSTINSTALL_EOF'
#!/usr/bin/env bash
# MultiversaGroup Lab — Post-install script
# Run this ONCE after booting your new Ubuntu install from the USB

set -euo pipefail

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  MultiversaGroup Lab — First Boot Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Update system
sudo apt-get update -qq && sudo apt-get upgrade -y

# Install Claude Code CLI
# pnpm approach (Multiversa rule: never npm)
curl -fsSL https://get.pnpm.io/install.sh | sh -
export PNPM_HOME="$HOME/.local/share/pnpm"
export PATH="$PNPM_HOME:$PATH"
pnpm add -g @anthropic-ai/claude-code

# Install the lab-setup skill
mkdir -p ~/.claude/skills
SKILLS_DIR="$HOME/.claude/skills/lab-setup"
if [ ! -d "$SKILLS_DIR" ]; then
  echo "→ Clone or copy the lab-setup skill to ~/.claude/skills/lab-setup/"
  echo "  (copy from your other machine or USB)"
fi

# Run the full stack setup
if [ -f "$SKILLS_DIR/scripts/setup_linux.sh" ]; then
  sudo bash "$SKILLS_DIR/scripts/setup_linux.sh"
  bash "$SKILLS_DIR/scripts/setup_stack.sh" --profile multiversa_group
fi

echo ""
echo "✓ Post-install complete."
echo "  Run: /lab-setup multiversa  to set up the private workspace"
POSTINSTALL_EOF

chmod +x "$POSTINSTALL"
echo "  ✓ Post-install script saved: $POSTINSTALL"
echo "     Copy it to the USB or keep it accessible for after the install."

# ── Step 5: Instructions ──────────────────────────────────────────────────────

separator
echo "  NEXT STEPS (manual)"
separator
cat << 'INSTRUCTIONS'
  1. Eject the USB safely:
       diskutil eject /dev/diskN

  2. Insert USB into target machine. Boot from USB:
       - Hold OPTION (⌥) on Mac at startup
       - Press F12 / DEL on PC at startup

  3. In the Ubuntu installer:
       - Choose "Install Ubuntu"
       - At "Installation type": select "Something else" for manual partitioning
       - Create partitions with LUKS encryption:

         /dev/sdX1  512MB   FAT32   /boot/efi   (no encryption)
         /dev/sdX2    1GB   ext4    /boot        (no encryption)
         /dev/sdX3  rest    LUKS    (encrypt this)
           └── LVM inside LUKS:
               swap  4GB
               /    14GB  ext4
               /home rest  ext4

       - Set a strong LUKS passphrase (memorize it — no recovery without it)

  4. After install completes, reboot and enter your LUKS passphrase.

  5. Copy multiversa_postinstall.sh to the new system and run it:
       bash multiversa_postinstall.sh

INSTRUCTIONS
separator
