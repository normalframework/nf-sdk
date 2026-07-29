#!/bin/sh
# Normal Framework (NF) installer
# Usage: curl -fsSL https://raw.githubusercontent.com/normalframework/nf-sdk/master/install.sh | sh
#
# Environment variables (all optional):
#   NF_TAG          Image tag to install (default: 3.10)
#   NF_PORT         Console port (default: 8080)
#   NF_DATA_DIR     NF data directory (rootless default: ~/nf/data, root default: /var/nf)
#   NF_REDIS_DIR    Redis data directory (rootless default: ~/nf/redis, root default: /var/nf-redis)
#   INSTALL_DIR     Where to write docker-compose.yml (rootless default: ~/nf, root default: /opt/nf)
# Every image pull is gated behind a Normal portal account. By default the installer prints
# a sign-in link: you sign in, set up the site, and the portal hands back registry pull
# credentials. That same approval licenses the box once it boots — no second sign-in.
#
#   NF_USERNAME     Registry username  }  escape hatch for CI / air-gapped installs: set
#   NF_PASSWORD     Registry password  }  both to skip the browser sign-in and pull with
#   NF_REGISTRY     Registry hostname  }  these credentials directly (box stays unlicensed
#                                          until you license it from the console)

COMPOSE_BASE_URL="https://raw.githubusercontent.com/normalframework/nf-sdk/master/compose"
set -e

# ── Colors ────────────────────────────────────────────────────────────────────
if [ -t 1 ]; then
  RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
  BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'
else
  RED=''; GREEN=''; YELLOW=''; BLUE=''; BOLD=''; NC=''
fi

# ── Config ────────────────────────────────────────────────────────────────────
NF_TAG="${NF_TAG:-3.10}"
NF_PORT="${NF_PORT:-8080}"
NF_REDIS_PORT="${NF_REDIS_PORT:-6379}"
# Directory defaults are set after rootless detection below

GA_REGISTRY="normal.azurecr.io"
# NF_PORTAL_URL overrides the portal the installer signs in against (e.g. a dev/staging
# portal). Defaults to production.
PORTAL_URL="${NF_PORTAL_URL:-https://portal.normal-online.net}"

# ── Helpers ───────────────────────────────────────────────────────────────────
info()    { printf "${BLUE}[→]${NC} %s\n" "$*"; }
ok()      { printf "${GREEN}[✓]${NC} %s\n" "$*"; }
warn()    { printf "${YELLOW}[!]${NC} %s\n" "$*"; }
die()     { printf "${RED}[✗]${NC} %s\n" "$*" >&2; exit 1; }
step()    { printf "\n${BOLD}── %s ──${NC}\n" "$*"; }

# Extract a top-level string field from a JSON body on stdin (camelCase, as emitted by the
# gRPC-JSON transcoder). Usage: printf '%s' "$json" | _json_str <key>
_json_str() {
  grep -o "\"$1\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | head -1 \
    | sed 's/.*:[[:space:]]*"\(.*\)"$/\1/'
}

# Find the first free TCP port starting from $1
find_free_port() {
  _p="$1"
  while ss -tlnp 2>/dev/null | grep -q ":$_p "; do
    _p=$((_p + 1))
  done
  echo "$_p"
}

# Read from /dev/tty so prompts work when stdin is a pipe (curl | sh)
ask() {
  _var="$1"; _msg="$2"; _default="$3"
  if [ -n "$_default" ]; then
    printf "%s [%s]: " "$_msg" "$_default" >/dev/tty
  else
    printf "%s: " "$_msg" >/dev/tty
  fi
  read -r _input </dev/tty
  eval "$_var=\${_input:-\$_default}"
}

ask_secret() {
  _var="$1"; _msg="$2"
  printf "%s: " "$_msg" >/dev/tty
  stty -echo </dev/tty 2>/dev/null || true
  read -r _input </dev/tty
  stty echo </dev/tty 2>/dev/null || true
  printf "\n" >/dev/tty
  eval "$_var=\$_input"
}

# ── Banner ────────────────────────────────────────────────────────────────────
printf "${BOLD}"
cat <<'BANNER'
    ___           ___
   /__/\         /  /\
   \  \:\       /  /:/_
    \  \:\     /  /:/ /\
_____\__\:\   /  /:/ /:/
/__/::::::::\ /__/:/ /:/
\  \:\~~\~~\/ \  \:\/:/
 \  \:\  ~~~   \  \::/
  \  \:\        \  \:\
   \  \:\        \  \:\
    \__\/         \__\/
BANNER
printf "${NC}"
printf "${BOLD}Normal Framework Installer${NC} — version %s\n\n" "$NF_TAG"

# ── Prerequisites ─────────────────────────────────────────────────────────────
step "Checking prerequisites"

# Port selection happens after the install dir is known (below), so a re-run can
# reuse the existing install's ports instead of remapping onto a second instance.

# OS
OS_ID="unknown"
if [ -f /etc/os-release ]; then
  # shellcheck disable=SC1091
  OS_ID="$(. /etc/os-release && echo "$ID")"
fi

# Privilege
if [ "$(id -u)" -eq 0 ]; then
  SUDO=""
  IS_ROOT=true
else
  SUDO="sudo"
  IS_ROOT=false
fi

# Docker / Podman
DCMD=""
if command -v docker >/dev/null 2>&1; then
  DCMD="docker"
elif command -v podman >/dev/null 2>&1; then
  DCMD="podman"
fi

# ── Install Docker if missing ─────────────────────────────────────────────────
if [ -z "$DCMD" ]; then
  step "Installing Docker"
  case "$OS_ID" in
    ubuntu|debian|raspbian)
      info "Installing Docker CE via apt..."
      # DEBIAN_FRONTEND and NEEDRESTART_MODE suppress interactive prompts
      # (needrestart otherwise blocks on "pending kernel upgrade" dialogs)
      export DEBIAN_FRONTEND=noninteractive
      export NEEDRESTART_MODE=a
      # Kill unattended-upgrades to release the dpkg lock (SIGKILL, non-blocking)
      $SUDO systemctl kill --signal=SIGKILL unattended-upgrades apt-daily.service \
        apt-daily-upgrade.service 2>/dev/null || true
      sleep 1  # let kernel release dpkg locks from killed processes
      # Fix any interrupted dpkg state from the SIGKILL
      $SUDO dpkg --configure -a 2>/dev/null || true
      $SUDO apt-get update -q
      $SUDO apt-get install -y -q ca-certificates curl gnupg lsb-release
      $SUDO install -m 0755 -d /etc/apt/keyrings
      curl -fsSL "https://download.docker.com/linux/$OS_ID/gpg" \
        | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
      $SUDO chmod a+r /etc/apt/keyrings/docker.gpg
      echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/$OS_ID $(lsb_release -cs) stable" \
        | $SUDO tee /etc/apt/sources.list.d/docker.list >/dev/null
      $SUDO apt-get update -q
      $SUDO apt-get install -y -q docker-ce docker-ce-cli containerd.io docker-compose-plugin
      ;;
    fedora|rhel|centos|rocky|almalinux)
      info "Installing Docker CE via dnf..."
      $SUDO dnf -y install dnf-plugins-core
      $SUDO dnf config-manager --add-repo \
        https://download.docker.com/linux/fedora/docker-ce.repo 2>/dev/null || true
      $SUDO dnf -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin
      $SUDO systemctl enable --now docker
      ;;
    *)
      die "Unsupported OS '$OS_ID'. Install Docker manually from https://docs.docker.com/get-docker/ then re-run."
      ;;
  esac
  DCMD="docker"
  ok "Docker installed"
fi

# ── Make sure Docker daemon is reachable ──────────────────────────────────────
if ! $DCMD info >/dev/null 2>&1; then
  info "Docker daemon not responding, trying to start it..."
  $SUDO systemctl start docker 2>/dev/null || true
  sleep 3
fi

if ! $DCMD info >/dev/null 2>&1; then
  # Not in docker group yet — add user and bake sudo into DCMD for this session
  if [ "$IS_ROOT" = "false" ] && $SUDO $DCMD info >/dev/null 2>&1; then
    $SUDO usermod -aG docker "$USER" 2>/dev/null || true
    info "Added $USER to docker group; using sudo docker for this install"
    DCMD="sudo $DCMD"   # sudo baked in; SUDO stays intact for filesystem ops
  else
    die "Cannot reach Docker daemon. Make sure Docker is running and try again."
  fi
fi

ok "Docker is running: $(${DCMD} --version 2>/dev/null | head -1)"

# ── Detect rootless mode ──────────────────────────────────────────────────────
ROOTLESS=false
if $DCMD info 2>/dev/null | grep -qi rootless; then
  ROOTLESS=true
fi
if [ "$DCMD" = "podman" ]; then
  ROOTLESS=true
fi
if [ "$ROOTLESS" = "true" ]; then
  ok "Rootless container runtime detected"
fi

# ── docker compose ────────────────────────────────────────────────────────────
CCMD=""
if $DCMD compose version >/dev/null 2>&1; then
  CCMD="$DCMD compose"
elif command -v podman-compose >/dev/null 2>&1; then
  CCMD="podman-compose"
elif command -v docker-compose >/dev/null 2>&1; then
  CCMD="docker-compose"
fi

if [ -z "$CCMD" ]; then
  info "Installing docker compose plugin..."
  case "$OS_ID" in
    ubuntu|debian|raspbian)
      $SUDO apt-get install -y -q docker-compose-plugin 2>/dev/null || true
      ;;
    fedora|rhel|centos|rocky|almalinux)
      # podman-compose provides "podman compose"; docker-compose-plugin is for Docker installs
      if [ "$DCMD" = "podman" ]; then
        $SUDO dnf -y install podman-compose 2>/dev/null || true
      else
        $SUDO dnf -y install docker-compose-plugin 2>/dev/null || true
      fi
      ;;
  esac
  if $DCMD compose version >/dev/null 2>&1; then
    CCMD="$DCMD compose"
  elif command -v podman-compose >/dev/null 2>&1; then
    CCMD="podman-compose"
  else
    die "docker compose not found. Install it from https://docs.docker.com/compose/install/ and try again."
  fi
fi

ok "Compose: $CCMD ($($CCMD version --short 2>/dev/null || echo 'version unknown'))"

# ── If rootless, we don't need sudo for docker commands ───────────────────────
if [ "$ROOTLESS" = "true" ]; then
  SUDO_DCMD=""
  INSTALL_DIR="${INSTALL_DIR:-$HOME/nf}"
  NF_DATA_DIR="${NF_DATA_DIR:-$HOME/nf/data}"
  NF_REDIS_DIR="${NF_REDIS_DIR:-$HOME/nf/redis}"
else
  # If sudo is already baked into DCMD (docker group workaround), don't double-prefix
  case "$DCMD" in sudo*) SUDO_DCMD="" ;; *) SUDO_DCMD="$SUDO" ;; esac
  INSTALL_DIR="${INSTALL_DIR:-/opt/nf}"
  NF_DATA_DIR="${NF_DATA_DIR:-/var/nf}"
  NF_REDIS_DIR="${NF_REDIS_DIR:-/var/nf-redis}"
fi

ENV_FILE="$INSTALL_DIR/.env"

# ── Port selection ────────────────────────────────────────────────────────────
# On a re-run (existing install) reuse the ports already in .env so we upgrade the
# same instance in place. On a fresh install, pick free ports if the defaults are taken.
_read_env_var() {  # $1=var name -> value from an existing (possibly root-owned) .env
  { $SUDO cat "$ENV_FILE" 2>/dev/null || cat "$ENV_FILE" 2>/dev/null; } \
    | grep "^$1=" | head -1 | cut -d= -f2-
}
if [ -f "$ENV_FILE" ] || { [ -n "$SUDO" ] && $SUDO test -f "$ENV_FILE"; }; then
  IS_UPGRADE=true
  _e_nf="$(_read_env_var NF_PORT)";       [ -n "$_e_nf" ] && NF_PORT="$_e_nf"
  _e_redis="$(_read_env_var NF_REDIS_PORT)"; [ -n "$_e_redis" ] && NF_REDIS_PORT="$_e_redis"
  info "Existing install at $INSTALL_DIR — upgrading in place (console :$NF_PORT, redis :$NF_REDIS_PORT)"
else
  _free_nf=$(find_free_port "$NF_PORT")
  if [ "$_free_nf" != "$NF_PORT" ]; then
    warn "Port $NF_PORT in use, using $_free_nf instead"; NF_PORT="$_free_nf"
  fi
  _free_redis=$(find_free_port "$NF_REDIS_PORT")
  if [ "$_free_redis" != "$NF_REDIS_PORT" ]; then
    warn "Port $NF_REDIS_PORT in use, using $_free_redis for Redis instead"; NF_REDIS_PORT="$_free_redis"
  fi
fi

# ── Registry ──────────────────────────────────────────────────────────────────
step "Registry"

# Return 0 if we're already authenticated and the token still works.
# ACR tokens have a 1-year expiry so a config-file check is usually enough,
# but we also do a quick /v2/ ping via curl to be sure.
check_auth() {
  _reg="$1"
  # docker commands run as root when SUDO_DCMD=sudo, so credentials live in
  # root's home; otherwise use the current user's config.
  if [ "$DCMD" = "podman" ]; then
    _cfg="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/containers/auth.json"
    [ -f "$_cfg" ] || _cfg="$HOME/.config/containers/auth.json"
  elif [ -n "$SUDO_DCMD" ]; then
    _cfg="/root/.docker/config.json"
  else
    _cfg="$HOME/.docker/config.json"
  fi

  # Not in config at all → not logged in
  grep -q "\"$_reg\"" "$_cfg" 2>/dev/null || return 1

  # Extract stored auth (base64 user:pass) and ping the registry v2 API.
  # A 200/401-with-challenge means the server is reachable; a 401 without our
  # creds being accepted means the token is stale.
  _auth=$(python3 -c "
import json,sys
cfg=json.load(open('$_cfg'))
print(cfg.get('auths',{}).get('$_reg',{}).get('auth',''))
" 2>/dev/null)
  [ -z "$_auth" ] && return 1

  _http=$(curl -sf -o /dev/null -w "%{http_code}" \
    -H "Authorization: Basic $_auth" \
    "https://${_reg}/v2/" 2>/dev/null)
  # 200 = OK, 401 = server up but auth failed (stale), anything else = network issue
  [ "$_http" = "200" ]
}

# ── Registry access ───────────────────────────────────────────────────────────
# Every image pull is gated behind a Normal portal account. There are three ways in, in
# priority order:
#   1. NF_USERNAME + NF_PASSWORD in the environment (CI / air-gapped escape hatch).
#   2. Credentials already cached from a previous run (upgrades reuse them).
#   3. A browser sign-in (device-authorization grant): the installer prints a link, you
#      sign in and set up the site, and the portal returns short-lived pull credentials.
#      The same approval later licenses the box — see "Activate a license" below.
REGISTRY="${NF_REGISTRY:-}"
DEVICE_CODE=""   # set by the sign-in path; possession of it licenses the box later

# device_sign_in: run the device-authorization grant to (a) mint registry pull credentials
# and (b) set up the site. Sets DEVICE_CODE, REGISTRY, NF_USERNAME, NF_PASSWORD.
device_sign_in() {
  # step 1: start a grant. No machine id yet — the box isn't running.
  _start="$(curl -sf -X POST "$PORTAL_URL/api/v1/license/device/start" \
    -H 'Content-Type: application/json' \
    -d "$(printf '{"version":"%s"}' "$NF_TAG")" 2>/dev/null || true)"
  DEVICE_CODE="$(printf '%s' "$_start" | _json_str deviceCode)"
  _user_code="$(printf '%s' "$_start" | _json_str userCode)"
  _verify_url="$(printf '%s' "$_start" | _json_str verificationUriComplete)"
  [ -n "$DEVICE_CODE" ] || die "Couldn't reach the sign-in service at $PORTAL_URL. Set NF_USERNAME/NF_PASSWORD to install without a browser."

  _name="$(hostname 2>/dev/null || echo 'Normal Site')"
  _verify_url="${_verify_url}&name=$(printf '%s' "$_name" | sed 's/ /%20/g')"
  printf "\n  ${BOLD}Sign in to Normal to set up this site and authorize the download:${NC}\n\n"
  printf "    ${BOLD}${BLUE}%s${NC}\n\n" "$_verify_url"
  [ -n "$_user_code" ] && printf "    (code: ${BOLD}%s${NC})\n\n" "$_user_code"

  # steps 2/3: poll until you approve in the browser; approval returns pull credentials.
  printf "Waiting for you to sign in "
  _waited=0
  while [ "$_waited" -lt 900 ]; do
    _poll="$(curl -sf -X POST "$PORTAL_URL/api/v1/license/device/poll" \
      -H 'Content-Type: application/json' \
      -d "$(printf '{"deviceCode":"%s"}' "$DEVICE_CODE")" 2>/dev/null || true)"
    _status="$(printf '%s' "$_poll" | _json_str status)"
    case "$_status" in
      *APPROVED*|*COMPLETED*)
        REGISTRY="$(printf '%s' "$_poll" | _json_str registry)"
        NF_USERNAME="$(printf '%s' "$_poll" | _json_str dockerUsername)"
        NF_PASSWORD="$(printf '%s' "$_poll" | _json_str dockerPassword)"
        printf " ${GREEN}done!${NC}\n"
        return 0
        ;;
      *DENIED*|*EXPIRED*)
        die "Sign-in ${_status#DEVICE_PROVISION_STATUS_} — re-run the installer."
        ;;
    esac
    printf "."; sleep 3; _waited=$((_waited + 3))
  done
  die "Timed out waiting for sign-in — re-run the installer."
}

if [ -n "$NF_USERNAME" ] && [ -n "$NF_PASSWORD" ]; then
  # (1) explicit credentials from the environment
  [ -n "$REGISTRY" ] || REGISTRY="$GA_REGISTRY"
  info "Using registry credentials from the environment"
elif [ -n "$REGISTRY" ] && check_auth "$REGISTRY"; then
  # (2) already authenticated from a previous run (upgrade in place)
  ok "Already authenticated with $REGISTRY (token valid)"
  SKIP_LOGIN=true
else
  # (3) browser sign-in mints pull credentials and sets up the site
  step "Sign in to Normal"
  device_sign_in
fi

if [ "${SKIP_LOGIN:-false}" != "true" ]; then
  [ -n "$REGISTRY"    ] || die "Could not determine the registry."
  [ -n "$NF_USERNAME" ] || die "Could not determine the registry username."
  [ -n "$NF_PASSWORD" ] || die "Could not determine the registry password."
  info "Logging in to $REGISTRY..."
  if printf '%s' "$NF_PASSWORD" \
      | $SUDO_DCMD $DCMD login --username "$NF_USERNAME" --password-stdin "$REGISTRY"; then
    ok "Authenticated with $REGISTRY"
  else
    die "Registry login failed. Check your credentials and try again."
  fi
fi

# ── Directories ───────────────────────────────────────────────────────────────
step "Setting up directories"

_mkdir() { if [ "$ROOTLESS" = "true" ]; then mkdir -p "$1"; else $SUDO mkdir -p "$1"; fi; }

for d in "$NF_DATA_DIR" "$NF_REDIS_DIR" "$INSTALL_DIR"; do
  if [ ! -d "$d" ]; then
    _mkdir "$d"
    ok "Created $d"
  else
    info "$d already exists"
  fi
done

# ── Docker daemon log rotation ────────────────────────────────────────────────
# Only configure if running as root and Docker is the runtime (not Podman)
if [ "$IS_ROOT" = "true" ] && [ "$DCMD" = "docker" ]; then
  DAEMON_JSON="/etc/docker/daemon.json"
  if [ ! -f "$DAEMON_JSON" ]; then
    step "Configuring Docker log rotation"
    mkdir -p /etc/docker
    cat >"$DAEMON_JSON" <<'DAEMON'
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
DAEMON
    systemctl reload docker 2>/dev/null || systemctl restart docker 2>/dev/null || true
    ok "Log rotation configured in $DAEMON_JSON"
  fi
fi

# ── Download docker-compose.yml ───────────────────────────────────────────────
step "Setting up docker-compose.yml"

COMPOSE_FILE="$INSTALL_DIR/docker-compose.yml"

if [ "$ROOTLESS" = "true" ]; then
  COMPOSE_VARIANT="linux-rootless"
else
  COMPOSE_VARIANT="linux"
fi

# Always fetch the current compose. It's a parameterized template driven entirely by
# .env, so overwriting is safe and ensures re-runs/upgrades pick up compose fixes
# instead of keeping a stale copy.
info "Downloading compose/$COMPOSE_VARIANT.yml..."
_tmp="$(mktemp)"
curl -fsSL "$COMPOSE_BASE_URL/$COMPOSE_VARIANT.yml" -o "$_tmp" \
  || die "Failed to download compose file from $COMPOSE_BASE_URL/$COMPOSE_VARIANT.yml"
if [ -w "$INSTALL_DIR" ]; then
  mv "$_tmp" "$COMPOSE_FILE"
else
  $SUDO cp "$_tmp" "$COMPOSE_FILE"
  rm -f "$_tmp"
fi
ok "Downloaded compose/$COMPOSE_VARIANT.yml → $COMPOSE_FILE"

# Compute memory limits from total RAM (redis=33%, nf=50%)
_mem_kb=$(awk '/MemTotal/ { print $2 }' /proc/meminfo 2>/dev/null || echo 0)
if [ "$_mem_kb" -gt 0 ] 2>/dev/null; then
  NF_REDIS_MEM_LIMIT="$(awk "BEGIN { printf \"%dm\", $_mem_kb / 3 / 1024 }")"
  NF_MEM_LIMIT="$(awk "BEGIN { printf \"%dm\", $_mem_kb / 2 / 1024 }")"
else
  NF_REDIS_MEM_LIMIT="1g"
  NF_MEM_LIMIT="2g"
fi

# Write .env so docker compose picks up the right values on future runs too
_env_tmp="$(mktemp)"
cat >"$_env_tmp" <<EOF
REGISTRY=${REGISTRY}
NF_TAG=${NF_TAG}
NF_PORT=${NF_PORT}
NF_DATA_DIR=${NF_DATA_DIR}
NF_REDIS_DIR=${NF_REDIS_DIR}
NF_REDIS_PORT=${NF_REDIS_PORT}
NF_REDIS_MEM_LIMIT=${NF_REDIS_MEM_LIMIT}
NF_MEM_LIMIT=${NF_MEM_LIMIT}
EOF
if [ -w "$INSTALL_DIR" ]; then
  mv "$_env_tmp" "$ENV_FILE"
else
  $SUDO cp "$_env_tmp" "$ENV_FILE"
  rm -f "$_env_tmp"
fi
ok "Wrote $ENV_FILE"

# ── Pull images ───────────────────────────────────────────────────────────────
step "Pulling containers (this may take a few minutes on first install)"
cd "$INSTALL_DIR"
# podman-compose (called as "podman compose" or directly) doesn't accept --quiet
if [ "$DCMD" = "podman" ]; then
  $SUDO_DCMD $CCMD pull
else
  $SUDO_DCMD $CCMD pull --quiet
fi
ok "Images pulled"

# ── Start NF ──────────────────────────────────────────────────────────────────
step "Starting Normal Framework"
if [ "$DCMD" = "podman" ]; then
  $SUDO_DCMD $CCMD up -d
else
  $SUDO_DCMD $CCMD up -d --quiet-pull
fi
ok "Containers started"

# ── Wait for console ──────────────────────────────────────────────────────────
step "Waiting for console to come up"
MAX_WAIT=180
WAITED=0
printf "Polling http://localhost:%s " "$NF_PORT"
while [ "$WAITED" -lt "$MAX_WAIT" ]; do
  if curl -sf "http://localhost:$NF_PORT" >/dev/null 2>&1; then
    printf " ${GREEN}ready!${NC}\n"
    READY=true
    break
  fi
  printf "."
  sleep 3
  WAITED=$((WAITED + 3))
done

if [ "${READY:-false}" != "true" ]; then
  printf "\n"
  warn "Console didn't respond within ${MAX_WAIT}s — it may still be initializing"
  warn "Check logs with: cd $INSTALL_DIR && ${CCMD} logs -f"
fi

# ── Activate a license ────────────────────────────────────────────────────────
# The site was already set up during the sign-in at the start; possession of DEVICE_CODE is
# the approved capability. Now that the box is running and has a machine id, exchange the
# device_code for a license (no second sign-in) and install it over localhost. The box uses
# only endpoints it already ships (/api/v1/platform/info and /api/v1/platform/license).
step "Activate a license"
BOX="http://localhost:$NF_PORT"

# Wait for the box API to answer — a fresh first boot can take a couple of minutes, and
# activation needs it (to read the machine id and later install the license). This is a
# more accurate readiness check than the console poll above.
printf "Waiting for the box API "
_waited=0; INFO=""
while [ "$_waited" -lt 180 ]; do
  INFO="$(curl -sf "$BOX/api/v1/platform/info" 2>/dev/null || true)"
  [ -n "$INFO" ] && { printf " ${GREEN}ready!${NC}\n"; break; }
  printf "."; sleep 3; _waited=$((_waited + 3))
done
if [ -z "$INFO" ]; then
  printf "\n"
  warn "The box API isn't responding yet — re-run the installer once it's up to license."
fi

if printf '%s' "$INFO" | grep -q '"license"[^}]*"name"[[:space:]]*:[[:space:]]*"[^"]'; then
  # Already licensed (upgrade / re-run) — nothing to do.
  info "This site is already licensed."
elif [ -z "$DEVICE_CODE" ]; then
  # No sign-in happened this run (env-credential or cached-auth path). Nothing to exchange —
  # the box auto-provisions if AUTO_PROVISION_KEY is set, otherwise license from the console.
  if [ -n "$INFO" ]; then
    info "Box is running unlicensed — license it from the console at $BOX"
  fi
elif [ -z "$INFO" ]; then
  warn "Couldn't reach the box to license it — re-run the installer once it's up."
else
  MID="$(printf '%s' "$INFO" | _json_str machineInfo)"
  BOX_VERSION="$(printf '%s' "$INFO" | _json_str version)"

  printf "Requesting your license "
  _waited=0
  while [ "$_waited" -lt 120 ]; do
    COMPLETE="$(curl -sf -X POST "$PORTAL_URL/api/v1/license/device/complete" \
      -H 'Content-Type: application/json' \
      -d "$(printf '{"deviceCode":"%s","mid":"%s","version":"%s"}' "$DEVICE_CODE" "$MID" "$BOX_VERSION")" 2>/dev/null || true)"
    LICENSE_JWT="$(printf '%s' "$COMPLETE" | _json_str license)"
    if [ -n "$LICENSE_JWT" ]; then
      INSTANCE_UUID="$(printf '%s' "$COMPLETE" | _json_str instanceUuid)"
      if curl -sf -X POST "$BOX/api/v1/platform/license" \
          -H 'Content-Type: application/json' \
          -d "$(printf '{"license":"%s"}' "$LICENSE_JWT")" >/dev/null 2>&1; then
        printf " ${GREEN}licensed!${NC}\n"
        ok "Your site is activated."
        LINKED=true
      else
        printf "\n"; warn "Got a license, but applying it on the box failed."
      fi
      break
    fi
    printf "."; sleep 3; _waited=$((_waited + 3))
  done
  if [ "${LINKED:-false}" != "true" ]; then
    printf "\n"; warn "Couldn't obtain the license yet — re-run the installer to retry."
  fi
fi

# ── Done ──────────────────────────────────────────────────────────────────────
printf "\n${GREEN}${BOLD}"
printf "╔══════════════════════════════════════════════╗\n"
printf "║   Normal Framework is running!               ║\n"
printf "╚══════════════════════════════════════════════╝\n"
printf "${NC}\n"
printf "  ${BOLD}Console${NC}   http://localhost:%s\n" "$NF_PORT"
# Remote access: online services expose the console at <instance-uuid>.<tunnel-base>.
if [ "${LINKED:-false}" = "true" ] && [ -n "${INSTANCE_UUID:-}" ]; then
  printf "  ${BOLD}Remote${NC}    https://%s.%s\n" "$INSTANCE_UUID" "${NF_TUNNEL_BASE:-normal-online.net}"
fi
printf "  ${BOLD}Data${NC}      %s\n" "$NF_DATA_DIR"
printf "  ${BOLD}Compose${NC}   %s\n" "$COMPOSE_FILE"
printf "\n"
printf "  Manage:  cd %s && %s [logs|ps|down|up]\n" "$INSTALL_DIR" "$CCMD"
if [ "${LINKED:-false}" = "true" ] && [ -n "${INSTANCE_UUID:-}" ]; then
  printf "\n  ${BLUE}Remote access may take a minute to come online (DNS + tunnel).${NC}\n"
fi
if [ "${LINKED:-false}" != "true" ]; then
  printf "\n  ${YELLOW}Not licensed yet:${NC} re-run this installer to finish, or license\n"
  printf "  from the console at http://localhost:%s\n" "$NF_PORT"
fi
printf "\n"
