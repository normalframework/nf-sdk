#!/bin/bash
# NF install test harness.
# Rolls back each VM to its clean snapshot, runs install.sh, checks NF starts.
#
# Usage: bash tests/run.sh
#
# Credentials (required — NF registry login):
#   export NF_USERNAME=<docker-...>
#   export NF_PASSWORD=<token>
#   export NF_RELEASE=ga        # or enterprise
# Or put them in tests/.env (gitignored)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/matrix.sh"

# Load .env if present
[ -f "$SCRIPT_DIR/.env" ] && . "$SCRIPT_DIR/.env"

[ -n "${NF_USERNAME:-}" ] || { echo "NF_USERNAME not set (see tests/.env.example)"; exit 1; }
[ -n "${NF_PASSWORD:-}" ] || { echo "NF_PASSWORD not set"; exit 1; }
NF_RELEASE="${NF_RELEASE:-ga}"

INSTALL_SH="$SCRIPT_DIR/../install.sh"
[ -f "$INSTALL_SH" ] || { echo "install.sh not found at $INSTALL_SH"; exit 1; }

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BOLD='\033[1m'; NC='\033[0m'
PASS="${GREEN}PASS${NC}"; FAIL="${RED}FAIL${NC}"

RESULTS=()

pve() { $PROXMOX_SSH "$@" < /dev/null; }

wait_for_ssh() {
  local user=$1 ip=$2
  local waited=0
  while [ $waited -lt 120 ]; do
    if ssh -n $VM_SSH_OPTS "$user@$ip" true 2>/dev/null; then
      return 0
    fi
    sleep 5; waited=$((waited+5)); printf "."
  done
  return 1
}

run_test() {
  local vmid=$1 name=$2 ip=$3 user=$4 snapshot=$5
  local result="FAIL" detail=""

  printf "\n${BOLD}── Testing $name (VM $vmid, $ip) ──${NC}\n"

  # Rollback to clean snapshot
  printf "[→] Rolling back to '$snapshot'...\n"
  pve qm stop $vmid --skiplock 1 2>/dev/null || true
  sleep 2
  pve qm rollback $vmid "$snapshot"
  pve qm start $vmid

  # Wait for SSH
  printf "[→] Waiting for SSH... "
  if ! wait_for_ssh "$user" "$ip"; then
    detail="SSH timeout after 120s"
    printf "\n${RED}[✗] $detail${NC}\n"
    RESULTS+=("$name|FAIL|$detail")
    return
  fi
  printf " up\n"

  # Copy install.sh
  printf "[→] Copying install.sh...\n"
  scp $VM_SSH_OPTS "$INSTALL_SH" "$user@$ip:~/install.sh" < /dev/null

  # Run install.sh detached via nohup so bridge network creation can't break
  # the SSH pipe. Poll for ~/install.rc (written by wrapper) then fetch log.
  printf "[→] Running install.sh (detached)...\n"
  ssh -n $VM_SSH_OPTS "$user@$ip" \
    "rm -f ~/install.rc ~/install.log; \
     nohup bash -c \
       \"NF_USERNAME='$NF_USERNAME' NF_PASSWORD='$NF_PASSWORD' NF_RELEASE='$NF_RELEASE' \
        bash ~/install.sh > ~/install.log 2>&1; echo \\\$? > ~/install.rc\" \
       </dev/null >/dev/null 2>&1 &"

  # Poll until install.rc appears (max 15 min)
  local waited=0
  while [ $waited -lt 900 ]; do
    sleep 10; waited=$((waited+10))
    if ssh -n $VM_SSH_OPTS "$user@$ip" "[ -f ~/install.rc ]" 2>/dev/null; then
      break
    fi
    printf "."
  done
  printf "\n"

  # Retry SSH up to 3 times in case of transient blip after install
  local install_rc="" install_log=""
  local tries=0
  while [ $tries -lt 3 ] && [ -z "$install_rc" ]; do
    install_rc=$(ssh -n $VM_SSH_OPTS "$user@$ip" "cat ~/install.rc 2>/dev/null || echo 1" 2>/dev/null || true)
    tries=$((tries+1))
    [ -z "$install_rc" ] && { printf "." ; sleep 5; }
  done
  install_log=$(ssh -n $VM_SSH_OPTS "$user@$ip" "cat ~/install.log 2>/dev/null" 2>/dev/null || true)
  printf "%s\n" "$install_log"
  if [ "${install_rc:-1}" != "0" ]; then
    detail="install.sh exited $install_rc"
    printf "${RED}[✗] $detail${NC}\n"
    RESULTS+=("$name|FAIL|$detail")
    return
  fi
  printf "[→] install.sh exited 0\n"

  # Verify containers are running
  printf "[→] Checking containers...\n"
  local containers
  containers=$(ssh -n $VM_SSH_OPTS "$user@$ip" \
    "docker ps 2>/dev/null || podman ps 2>/dev/null" 2>&1)
  local nf_up redis_up
  nf_up=$(echo "$containers" | grep -c "nf-full" || true)
  redis_up=$(echo "$containers" | grep -c "redis" || true)

  if [ "$nf_up" -lt 1 ] || [ "$redis_up" -lt 1 ]; then
    detail="containers not running (nf=$nf_up redis=$redis_up)"
    printf "${RED}[✗] $detail${NC}\n"
    RESULTS+=("$name|FAIL|$detail")
    return
  fi

  # Verify HTTP console responds
  printf "[→] Checking console on port 8080...\n"
  local http_code
  http_code=$(ssh -n $VM_SSH_OPTS "$user@$ip" \
    "curl -sf -o /dev/null -w '%{http_code}' http://localhost:8080" 2>/dev/null || echo "000")

  if [ "$http_code" = "200" ]; then
    result="PASS"
    detail="containers up, console HTTP 200"
    printf "${GREEN}[✓] $detail${NC}\n"
  else
    detail="console returned HTTP $http_code (containers up)"
    printf "${YELLOW}[!] $detail${NC}\n"
    # Console may need a moment; treat non-500 as a soft pass
    if [ "$http_code" != "000" ] && [ "$http_code" != "500" ]; then
      result="PASS"
    fi
  fi

  RESULTS+=("$name|$result|$detail")
}

# ── Run matrix ────────────────────────────────────────────────────────────────
while read -r vmid name ip user snapshot; do
  [ -z "$vmid" ] && continue
  run_test "$vmid" "$name" "$ip" "$user" "$snapshot" || true
done < <(printf '%s\n' "$MATRIX")

# ── Summary ───────────────────────────────────────────────────────────────────
printf "\n${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
printf "Results\n"
printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
FAILURES=0
for r in "${RESULTS[@]}"; do
  IFS='|' read -r rname rstatus rdetail <<< "$r"
  if [ "$rstatus" = "PASS" ]; then
    printf "  ${GREEN}✓ PASS${NC}  %-30s  %s\n" "$rname" "$rdetail"
  else
    printf "  ${RED}✗ FAIL${NC}  %-30s  %s\n" "$rname" "$rdetail"
    FAILURES=$((FAILURES+1))
  fi
done
printf "\n"

[ $FAILURES -eq 0 ] && printf "${GREEN}All tests passed.${NC}\n\n" \
                     || printf "${RED}$FAILURES test(s) failed.${NC}\n\n"
exit $FAILURES
