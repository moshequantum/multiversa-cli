#!/usr/bin/env bash
# Encrypted bootable Linux USB — from Linux
# Creates a LUKS-encrypted partition scheme for a bootable Ubuntu 24.04 USB
# Usage: sudo ./encrypted_usb_linux.sh

set -euo pipefail

UBUNTU_VERSION="24.04.4"
UBUNTU_ISO="ubuntu-${UBUNTU_VERSION}-desktop-amd64.iso"
UBUNTU_URL="https://releases.ubuntu.com/${UBUNTU_VERSION%%.*}/${UBUNTU_ISO}"

# ── UI helpers ────────────────────────────────────────────────────────────────

separator() { printf '\n%*s\n\n' 60 '' | tr ' ' '━'; }
confirm()   { read -r -p "  → $1 [y/N]: " REPLY; [[ "${REPLY,,}" == "y" ]]; }
prompt()    { read -r -p "  → $1: " REPLY; echo "$REPLY"; }

# ── Requires root ─────────────────────────────────────────────────────────────

if [[ $EUID -ne 0 ]]; then
  echo "ERROR: This script must be run as root (sudo)" >&2
  exit 1
fi

REAL_USER="${SUDO_USER:-$USER}"

# ── Step 1: Identify USB ──────────────────────────────────────────────────────

separator
echo "  STEP 1 — Identify your USB drive"
separator

echo "  Connected block devices:"
echo ""
lsblk -o NAME,SIZE,TYPE,MOUNTPOINT,LABEL -d | sed 's/^/    /'
echo ""
echo "  ⚠  CRITICAL: Choose the correct device. Wrong device = data loss."
echo "     USB drives are typically /dev/sdb, /dev/sdc, etc."
echo ""

USB_DEV=$(prompt "Enter USB device (e.g. /dev/sdb)")

if [[ ! "$USB_DEV" =~ ^/dev/[a-z]+$ ]]; then
  echo "ERROR: Invalid device. Must be /dev/sdX or /dev/nvmeXnY" >&2
  exit 1
fi

echo ""
echo "  Device info for $USB_DEV:"
lsblk -o NAME,SIZE,TYPE,FSTYPE,LABEL "$USB_DEV" 2>/dev/null | sed 's/^/    /'
echo ""

if ! confirm "Is this the correct USB drive? (THIS WILL BE WIPED)"; then
  echo "Aborted. Nothing was changed."
  exit 0
fi

# Unmount any mounted partitions
umount "${USB_DEV}"* 2>/dev/null || true

# ── Step 2: Download Ubuntu ISO ───────────────────────────────────────────────

separator
echo "  STEP 2 — Ubuntu 24.04 LTS ISO"
separator

ISO_PATH="/home/${REAL_USER}/Downloads/${UBUNTU_ISO}"

if [[ -f "$ISO_PATH" ]]; then
  echo "  ✓ ISO already downloaded: $ISO_PATH"
else
  echo "  Downloading Ubuntu $UBUNTU_VERSION..."
  sudo -u "$REAL_USER" curl -fL --progress-bar "$UBUNTU_URL" -o "$ISO_PATH"
fi

# ── Step 3: Partition the USB ─────────────────────────────────────────────────

separator
echo "  STEP 3 — Partition layout"
separator
echo ""
echo "  The following partitions will be created on $USB_DEV:"
echo ""
echo "    Part 1:   512MB   FAT32     /boot/efi   (UEFI — unencrypted)"
echo "    Part 2:     1GB   ext4      /boot        (GRUB — unencrypted)"
echo "    Part 3:   rest    LUKS      (encrypted)"
echo "                └── LVM group:"
echo "                    swap   4GB"
echo "                    /     14GB  ext4"
echo "                    /home rest  ext4"
echo ""

if ! confirm "Proceed with partitioning $USB_DEV"; then
  echo "Aborted."
  exit 0
fi

echo "  → Wiping partition table..."
wipefs -a "$USB_DEV"
sgdisk --zap-all "$USB_DEV"

echo "  → Creating GPT partition table..."
parted -s "$USB_DEV" mklabel gpt

echo "  → Creating /boot/efi (512MB FAT32)..."
parted -s "$USB_DEV" mkpart "EFI"  fat32 1MiB 513MiB
parted -s "$USB_DEV" set 1 esp on
mkfs.fat -F32 -n EFI "${USB_DEV}1"

echo "  → Creating /boot (1GB ext4)..."
parted -s "$USB_DEV" mkpart "BOOT" ext4 513MiB 1537MiB
mkfs.ext4 -L BOOT "${USB_DEV}2"

echo "  → Creating LUKS container (rest of disk)..."
parted -s "$USB_DEV" mkpart "LUKS" ext4 1537MiB 100%

# ── Step 4: LUKS encryption ───────────────────────────────────────────────────

separator
echo "  STEP 4 — LUKS encryption"
separator
echo ""
echo "  You will set a passphrase for the encrypted partition."
echo "  ⚠  This passphrase cannot be recovered. Write it down securely."
echo ""

cryptsetup luksFormat --type luks2 \
  --cipher aes-xts-plain64 \
  --key-size 512 \
  --hash sha512 \
  --iter-time 3000 \
  "${USB_DEV}3"

echo ""
echo "  → Opening LUKS container..."
cryptsetup open "${USB_DEV}3" mv_lab_crypt

# ── Step 5: LVM inside LUKS ───────────────────────────────────────────────────

separator
echo "  STEP 5 — LVM inside LUKS"
separator

pvcreate /dev/mapper/mv_lab_crypt
vgcreate mv_lab_vg /dev/mapper/mv_lab_crypt
lvcreate -L 4G   -n swap mv_lab_vg
lvcreate -L 14G  -n root mv_lab_vg
lvcreate -l 100%FREE -n home mv_lab_vg

mkswap -L SWAP  /dev/mv_lab_vg/swap
mkfs.ext4 -L ROOT /dev/mv_lab_vg/root
mkfs.ext4 -L HOME /dev/mv_lab_vg/home

echo "  ✓ LVM volumes created and formatted."

# ── Step 6: Mount and install ─────────────────────────────────────────────────

separator
echo "  STEP 6 — Installer"
separator
echo ""
echo "  The USB partition scheme is ready."
echo "  To install Ubuntu onto this structure:"
echo ""
echo "  1. Boot from the Ubuntu ISO (separate USB or existing install):"
echo "       sudo dd if=$ISO_PATH of=/dev/sdX bs=4M status=progress"
echo "       (use a DIFFERENT drive for the installer)"
echo ""
echo "  2. During Ubuntu installer 'Something else' partitioning:"
echo "       ${USB_DEV}1  → /boot/efi  (no format, FAT32)"
echo "       ${USB_DEV}2  → /boot      (no format, ext4)"
echo "       /dev/mv_lab_vg/root → /  (ext4, format)"
echo "       /dev/mv_lab_vg/home → /home (ext4, format)"
echo "       /dev/mv_lab_vg/swap → swap"
echo "       Boot loader: $USB_DEV"
echo ""

# Close LUKS (clean up)
vgchange -an mv_lab_vg
cryptsetup close mv_lab_crypt

echo "  ✓ LUKS container closed."
echo ""
echo "  Post-install: boot from the new USB and run:"
echo "  bash ~/.claude/skills/lab-setup/scripts/setup_linux.sh"
echo "  bash ~/.claude/skills/lab-setup/scripts/setup_stack.sh"
echo ""
separator
