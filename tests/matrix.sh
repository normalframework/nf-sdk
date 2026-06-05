# Test matrix — sourced by setup.sh and run.sh
# Format: VMID  NAME                  IP                USER     SNAPSHOT
#
# IPs are static (set via cloud-init), gateway 192.168.103.1

MATRIX="
120 ubuntu-2204-nf-test 192.168.103.120 ubuntu  clean
121 ubuntu-2404-nf-test 192.168.103.121 ubuntu  clean
122 fedora-nf-test      192.168.103.122 fedora  clean
"

# Cloud images (downloaded once to Proxmox)
IMAGE_120="https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
IMAGE_121="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
IMAGE_122="https://download.fedoraproject.org/pub/fedora/linux/releases/41/Cloud/x86_64/images/Fedora-Cloud-Base-Generic.x86_64-41-1.4.qcow2"

PROXMOX_HOST="root@n1"
PROXMOX_SSH="ssh -i $HOME/.ssh/id_ed25519_nftest $PROXMOX_HOST"
VM_SSH_KEY="$HOME/.ssh/id_ed25519_nftest"
VM_SSH_OPTS="-i $VM_SSH_KEY -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o ServerAliveInterval=15 -o ServerAliveCountMax=40"
STORAGE="local-lvm"
BRIDGE="vmbr0"
GATEWAY="192.168.103.1"
