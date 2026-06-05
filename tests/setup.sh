#!/bin/bash
# Create and snapshot clean test VMs on Proxmox.
# Run from your dev machine: bash tests/setup.sh
# Requires: root@n1 SSH access via id_ed25519_nftest
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/matrix.sh"

SSH_PUBKEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICf+rO3jD40jAp2nEZN4d/TmOX9zD9C4o+o5GLypMFtr nf-k8s-test"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BOLD='\033[1m'; NC='\033[0m'
ok()   { printf "${GREEN}[✓]${NC} %s\n" "$*"; }
info() { printf "${BOLD}[→]${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}[!]${NC} %s\n" "$*"; }
die()  { printf "${RED}[✗]${NC} %s\n" "$*" >&2; exit 1; }
pve()  { $PROXMOX_SSH "$@" < /dev/null; }

# ── Delete old fedora test VM ────────────────────────────────────────────────
if pve qm status 118 &>/dev/null; then
  warn "Deleting old fedora-podman-test (VM 118)..."
  pve qm stop 118 --skiplock 1 2>/dev/null || true
  sleep 3
  pve qm destroy 118 --purge 1
  ok "VM 118 deleted"
fi

# ── Ensure snippets storage is enabled ──────────────────────────────────────
info "Enabling snippets on local storage..."
pve pvesm set local --content 'iso,snippets,vztmpl,backup' 2>/dev/null || true

# ── Cloud-init vendor snippet (installs qemu-guest-agent on first boot) ──────
info "Writing cloud-init vendor snippet..."
printf '%s\n' \
  '#cloud-config' \
  'runcmd:' \
  '  - which apt-get && apt-get install -y -q qemu-guest-agent || true' \
  '  - which dnf  && dnf  install -y -q qemu-guest-agent || true' \
  '  - systemctl enable --now qemu-guest-agent || true' \
  | $PROXMOX_SSH "cat > /var/lib/vz/snippets/nf-test-vendor.yml"
ok "Vendor snippet written"

# ── Per-VM setup ─────────────────────────────────────────────────────────────
setup_vm() {
  local vmid=$1 name=$2 ip=$3 user=$4 snapshot=$5 image_url=$6
  local image_dir="/var/lib/vz/template/images"
  pve mkdir -p "$image_dir"
  local image_file="$image_dir/$(basename $image_url)"

  printf "\n${BOLD}── Setting up VM $vmid: $name ──${NC}\n"

  # Skip if already snapshotted
  if pve qm listsnapshot $vmid 2>/dev/null | grep -q "^.*$snapshot"; then
    warn "VM $vmid already has snapshot '$snapshot' — skipping"
    return 0
  fi

  # Destroy existing VM if present (but no clean snapshot yet)
  if pve qm status $vmid &>/dev/null; then
    warn "VM $vmid exists but has no '$snapshot' snapshot — recreating..."
    pve qm stop $vmid --skiplock 1 2>/dev/null || true
    sleep 2
    pve qm destroy $vmid --purge 1
  fi

  # Download cloud image if not cached
  if ! pve test -f "$image_file"; then
    info "Downloading $image_url ..."
    pve wget -q --show-progress -O "$image_file" "$image_url" \
      || pve curl -L --progress-bar -o "$image_file" "$image_url"
    ok "Downloaded $(basename $image_url)"
  else
    info "Using cached $(basename $image_url)"
  fi

  # Create VM
  info "Creating VM $vmid ($name)..."
  pve qm create $vmid \
    --name "$name" \
    --ostype l26 \
    --machine q35 \
    --bios ovmf \
    --efidisk0 "$STORAGE:0,efitype=4m,pre-enrolled-keys=0" \
    --cpu host \
    --cores 2 \
    --memory 4096 \
    --net0 "virtio,bridge=$BRIDGE" \
    --serial0 socket \
    --vga virtio \
    --agent enabled=1 \
    --scsihw virtio-scsi-pci

  # Import disk
  info "Importing disk..."
  pve qm importdisk $vmid "$image_file" "$STORAGE" --format qcow2
  pve qm set $vmid \
    --virtio0 "$STORAGE:vm-$vmid-disk-1,discard=on" \
    --boot order=virtio0

  # Cloud-init drive
  # Write SSH key to a temp file on Proxmox via stdin (avoids shell-quoting issues)
  local keyfile="/tmp/nf-test-sshkey-$vmid.pub"
  printf '%s\n' "$SSH_PUBKEY" | $PROXMOX_SSH "cat > $keyfile"
  pve qm set $vmid \
    --ide2 "$STORAGE:cloudinit" \
    --ciuser "$user" \
    --sshkeys "$keyfile" \
    --ipconfig0 "ip=$ip/24,gw=$GATEWAY" \
    --nameserver "8.8.8.8" \
    --cicustom "vendor=local:snippets/nf-test-vendor.yml"
  pve rm -f "$keyfile"

  # Resize disk to 20G
  pve qm resize $vmid virtio0 20G

  ok "VM $vmid created"

  # Start VM and wait for SSH
  info "Starting VM $vmid, waiting for SSH on $ip..."
  pve qm start $vmid
  local waited=0
  while [ $waited -lt 180 ]; do
    if $PROXMOX_SSH ssh $VM_SSH_OPTS "$user@$ip" true < /dev/null 2>/dev/null; then
      break
    fi
    sleep 5; waited=$((waited+5)); printf "."
  done
  printf "\n"
  if [ $waited -ge 180 ]; then
    die "VM $vmid SSH timeout after 180s (check console: qm terminal $vmid)"
  fi
  ok "SSH up"

  # cloud-init status --wait blocks until all modules finish (installs, runcmds, etc.)
  info "Waiting for cloud-init to complete (may take a few minutes)..."
  $PROXMOX_SSH ssh $VM_SSH_OPTS "$user@$ip" \
    "sudo cloud-init status --wait --long" < /dev/null 2>/dev/null || true
  ok "cloud-init done"

  # Snapshot
  info "Taking snapshot '$snapshot'..."
  pve qm stop $vmid
  sleep 3
  pve qm snapshot $vmid "$snapshot" --description "Clean OS install, no NF"
  ok "Snapshot '$snapshot' taken for VM $vmid ($name)"
}

# Run setup for each row in MATRIX
while read -r vmid name ip user snapshot; do
  [ -z "$vmid" ] && continue
  # Get image URL from IMAGE_<vmid> variable
  image_var="IMAGE_$vmid"
  image_url="${!image_var}"
  [ -z "$image_url" ] && die "No image URL defined for VMID $vmid (set IMAGE_$vmid in matrix.sh)"
  setup_vm "$vmid" "$name" "$ip" "$user" "$snapshot" "$image_url"
done < <(printf '%s\n' "$MATRIX")

printf "\n${GREEN}${BOLD}All VMs created and snapshotted.${NC}\n"
printf "Run tests with: bash tests/run.sh\n\n"
