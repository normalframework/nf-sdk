#!/bin/sh
# Normal Framework (NF) installer
# Usage: curl -fsSL https://raw.githubusercontent.com/normalframework/nf-sdk/master/install.sh | sh
#
# Environment variables (all optional):
#   NF_TAG          Image tag to install (default: 3.10)
#   NF_PORT         Console port (default: 8080)
#   NF_DATA_DIR     NF data directory (rootless/macOS default: ~/nf/data, root default: /var/nf)
#   NF_REDIS_DIR    Redis data directory (rootless/macOS default: ~/nf/redis, root default: /var/nf-redis)
#   INSTALL_DIR     Where to write docker-compose.yml (rootless/macOS default: ~/nf, root default: /opt/nf)
#   NF_TZ           Timezone for the container (macOS only; default: host timezone)
#   NF_ASSUME_YES   Set to 1 to skip the Docker Desktop confirmation prompt
#   NF_COMPOSE_REF  Git ref to fetch compose templates from (default: master)
# By default the installer prints a sign-in link: you sign in and set up the site, and that
# same approval licenses the box once it boots — no second sign-in. GA images pull
# anonymously, so there's no registry login in that path.
#
#   NF_USERNAME     Registry username  }  private/enterprise registry, or an air-gapped /
#   NF_PASSWORD     Registry password  }  CI install: set these to skip the browser sign-in
#   NF_REGISTRY     Registry hostname  }  and pull directly (box stays unlicensed until you
#                                          license it from the console)

# NF_COMPOSE_REF selects the git ref the compose templates are fetched from, so a
# branch can be tested end to end before it lands on master.
COMPOSE_BASE_URL="${NF_COMPOSE_BASE_URL:-https://raw.githubusercontent.com/normalframework/nf-sdk/${NF_COMPOSE_REF:-master}/compose}"
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

# Return 0 if something is already listening on TCP port $1.
# ss is Linux-only; macOS has neither ss nor a useful netstat -p, so fall back
# to lsof there.  If no tool answers, assume the port is free.
_port_in_use() {
  if command -v ss >/dev/null 2>&1; then
    ss -tln 2>/dev/null | grep -q "[:.]$1 "
  elif command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1
  else
    netstat -an 2>/dev/null | grep -q "[:.]$1 .*LISTEN"
  fi
}

# Find the first free TCP port starting from $1
find_free_port() {
  _p="$1"
  while _port_in_use "$_p"; do
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

# Warning shown whenever NF ends up on a VM-backed Docker (Docker Desktop on
# any platform, or podman machine).  These limits are the runtime's, not ours.
desktop_limitations() {
  warn "Docker Desktop is fine for evaluation and development, but is NOT recommended for production:"
  printf "      • ${BOLD}BACnet/IP broadcast does not work.${NC} Containers run in a Linux VM behind\n"
  printf "        NAT, so Who-Is/I-Am discovery neither reaches the LAN nor arrives from it.\n"
  printf "        Devices must be polled by unicast address, or reached via a BBMD using\n"
  printf "        foreign-device registration. BACnet/Ethernet and MS/TP do not work at all.\n"
  printf "      • ${BOLD}It does not reliably come back after a restart.${NC} Containers only run\n"
  printf "        while Docker Desktop runs, which needs a desktop login — after a reboot NF\n"
  printf "        stays down until someone logs in, and sleep/resume can wedge the VM.\n"
  printf "      • ${BOLD}Volumes are slower${NC} and redis persistence goes through the VM's file\n"
  printf "        sharing layer rather than a real filesystem.\n"
  printf "      For production, run NF on Linux with host networking.\n"
}

# ── Prerequisites ─────────────────────────────────────────────────────────────
step "Checking prerequisites"

# Port selection happens after the install dir is known (below), so a re-run can
# reuse the existing install's ports instead of remapping onto a second instance.

# OS
OS_ID="unknown"
IS_MACOS=false
case "$(uname -s 2>/dev/null)" in
  Darwin)
    IS_MACOS=true
    OS_ID="macos"
    ;;
  *)
    if [ -f /etc/os-release ]; then
      # shellcheck disable=SC1091
      OS_ID="$(. /etc/os-release && echo "$ID")"
    fi
    ;;
esac

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
    macos)
      # Nothing to install unattended here: Docker Desktop is a signed .app
      # that needs an admin install, and Homebrew may not be present.
      warn "No container runtime found on this Mac."
      printf "\n  Install Docker Desktop, then re-run this installer:\n"
      printf "    ${BOLD}brew install --cask docker${NC}\n"
      printf "    …or download it from ${BOLD}https://docs.docker.com/desktop/setup/install/mac-install/${NC}\n"
      printf "\n  Launch Docker Desktop once after installing so the daemon starts.\n\n"
      desktop_limitations
      printf "\n"
      die "Docker is not installed."
      ;;
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
  if [ "$IS_MACOS" = "true" ]; then
    # Docker Desktop / Podman Desktop are GUI apps — launch and wait for the
    # VM to boot, which takes appreciably longer than starting a daemon.
    if [ "$DCMD" = "podman" ]; then
      podman machine start >/dev/null 2>&1 || true
    else
      open -a Docker 2>/dev/null || open -a "Docker Desktop" 2>/dev/null || true
    fi
    _waited=0
    printf "Waiting for the Docker Desktop VM "
    while [ "$_waited" -lt 90 ]; do
      $DCMD info >/dev/null 2>&1 && break
      printf "."
      sleep 3
      _waited=$((_waited + 3))
    done
    printf "\n"
  else
    $SUDO systemctl start docker 2>/dev/null || true
    sleep 3
  fi
fi

if ! $DCMD info >/dev/null 2>&1; then
  if [ "$IS_MACOS" = "true" ]; then
    die "Cannot reach the Docker daemon. Start Docker Desktop from Applications, wait for the whale icon to stop animating, then re-run this installer."
  fi
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

DOCKER_INFO="$($DCMD info 2>/dev/null)"

# ── Docker Desktop (VM-backed runtime) ────────────────────────────────────────
# Docker Desktop reports "Operating System: Docker Desktop" on every platform,
# so this catches Mac and Windows/WSL as well as Docker Desktop on Linux.
IS_DESKTOP=false
if [ "$IS_MACOS" = "true" ] || echo "$DOCKER_INFO" | grep -qi "Docker Desktop"; then
  IS_DESKTOP=true
fi

if [ "$IS_DESKTOP" = "true" ]; then
  step "Docker Desktop detected"
  desktop_limitations
  if [ "${NF_ASSUME_YES:-}" != "1" ] && [ -e /dev/tty ]; then
    printf "\n"
    ask _CONTINUE "Continue installing on Docker Desktop? (y/n)" "y"
    case "$_CONTINUE" in
      [Yy]*) ;;
      *) die "Aborted. Set NF_ASSUME_YES=1 to skip this prompt." ;;
    esac
  fi
fi

# ── Detect rootless mode ──────────────────────────────────────────────────────
ROOTLESS=false
if echo "$DOCKER_INFO" | grep -qi rootless; then
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
    macos)
      warn "Docker Desktop bundles the compose plugin — update Docker Desktop to get it."
      ;;
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
# On macOS the daemon lives in a VM owned by the logged-in user: sudo buys
# nothing, and paths must stay under $HOME so Docker Desktop's file sharing
# will mount them without extra configuration.
if [ "$ROOTLESS" = "true" ] || [ "$IS_MACOS" = "true" ]; then
  SUDO_DCMD=""
  SUDO_FS=""            # everything lives under $HOME
  # "nf" is also the name of the product binary, so ~/nf is quite likely to
  # already exist as a file.  Fall back rather than failing on mkdir.
  NF_BASE="$HOME/nf"
  if [ -e "$NF_BASE" ] && [ ! -d "$NF_BASE" ]; then
    NF_BASE="$HOME/.nf"
    warn "$HOME/nf exists and is not a directory — installing to $NF_BASE instead"
  fi
  INSTALL_DIR="${INSTALL_DIR:-$NF_BASE}"
  NF_DATA_DIR="${NF_DATA_DIR:-$NF_BASE/data}"
  NF_REDIS_DIR="${NF_REDIS_DIR:-$NF_BASE/redis}"
else
  # If sudo is already baked into DCMD (docker group workaround), don't double-prefix
  case "$DCMD" in sudo*) SUDO_DCMD="" ;; *) SUDO_DCMD="$SUDO" ;; esac
  SUDO_FS="$SUDO"
  INSTALL_DIR="${INSTALL_DIR:-/opt/nf}"
  NF_DATA_DIR="${NF_DATA_DIR:-/var/nf}"
  NF_REDIS_DIR="${NF_REDIS_DIR:-/var/nf-redis}"
fi

# Container timezone: /etc/localtime can't be bind-mounted into the Docker
# Desktop VM, so the macOS compose file takes TZ as a variable instead.
if [ "$IS_MACOS" = "true" ] && [ -z "${NF_TZ:-}" ]; then
  NF_TZ="$(readlink /etc/localtime 2>/dev/null | sed -e 's|.*/zoneinfo/||')"
  NF_TZ="${NF_TZ:-UTC}"
fi

ENV_FILE="$INSTALL_DIR/.env"

# ── Port selection ────────────────────────────────────────────────────────────
# On a re-run (existing install) reuse the ports already in .env so we upgrade the
# same instance in place. On a fresh install, pick free ports if the defaults are taken.
# SUDO_FS (not SUDO) so an install under $HOME — rootless or macOS — never
# shells out to sudo here; its password prompt would be swallowed by 2>/dev/null
# and the installer would look like it had hung.
_read_env_var() {  # $1=var name -> value from an existing (possibly root-owned) .env
  { cat "$ENV_FILE" 2>/dev/null || $SUDO_FS cat "$ENV_FILE" 2>/dev/null; } \
    | grep "^$1=" | head -1 | cut -d= -f2-
}
if [ -f "$ENV_FILE" ] || { [ -n "$SUDO_FS" ] && $SUDO_FS test -f "$ENV_FILE"; }; then
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

  # Docker Desktop keeps credentials in an external helper (credsStore), so
  # config.json holds an empty entry and there's nothing to verify here — just
  # log in again, which is cheap and idempotent.
  grep -q '"credsStore"' "$_cfg" 2>/dev/null && return 1

  # python3 may be an unconfigured stub on macOS; don't invoke it if calling it
  # would just pop the Xcode command line tools installer.
  command -v python3 >/dev/null 2>&1 || return 1

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
# GA images ($GA_REGISTRY) pull anonymously — no registry login. What the browser sign-in
# buys you is the site setup and the license, not the download. Three ways in, in priority
# order:
#   1. NF_USERNAME + NF_PASSWORD in the environment (private/enterprise registry, CI).
#   2. Credentials already cached from a previous run (upgrades reuse them).
#   3. A browser sign-in (device-authorization grant): the installer prints a link, you
#      sign in and set up the site, and that same approval licenses the box once it boots
#      — see "Activate a license" below. Enterprise tenants also get pull credentials back;
#      GA tenants get none and pull anonymously.
REGISTRY="${NF_REGISTRY:-}"
DEVICE_CODE=""   # set by the sign-in path; possession of it licenses the box later

# The setup token minted by the sign-in is only good for an hour, and the
# licensing exchange happens much later — after the images pull and the box
# boots. Stamp when the grant started and stop using the token a few minutes
# before it dies, so a slow pull ends in a clear "sign in again" message
# instead of an opaque licensing failure.
SIGNIN_TTL=3600
SIGNIN_MARGIN=300
SIGNIN_STARTED=""
_signin_age()     { echo $(( $(date +%s) - SIGNIN_STARTED )); }
_signin_expired() {
  [ -n "$SIGNIN_STARTED" ] || return 1
  [ "$(_signin_age)" -ge "$((SIGNIN_TTL - SIGNIN_MARGIN))" ]
}

# device_sign_in: run the device-authorization grant to set up the site (and, for enterprise
# tenants, mint pull credentials). Sets DEVICE_CODE, REGISTRY, NF_USERNAME, NF_PASSWORD.
device_sign_in() {
  # step 1: start a grant. No machine id yet — the box isn't running.
  SIGNIN_STARTED="$(date +%s)"   # the token's hour starts ticking here
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
  printf "    This link is good for one hour, and the install has to finish\n"
  printf "    within that hour to license the box automatically.\n\n"

  # steps 2/3: poll until you approve in the browser. Approval returns pull credentials for
  # enterprise tenants; for GA it returns none and the images pull anonymously.
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
  # (3) browser sign-in sets up the site; pull credentials only come back for enterprise
  step "Sign in to Normal"
  device_sign_in
  if [ -z "$NF_USERNAME" ] || [ -z "$NF_PASSWORD" ]; then
    # GA: nothing to log in with, and nothing to log in for.
    [ -n "$REGISTRY" ] || REGISTRY="$GA_REGISTRY"
    info "Pulling from $REGISTRY (no registry login needed)"
    SKIP_LOGIN=true
  fi
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

_mkdir() {
  $SUDO_FS mkdir -p "$1" \
    || die "Couldn't create $1 — a component of that path already exists as a file. Set INSTALL_DIR / NF_DATA_DIR / NF_REDIS_DIR to another location and re-run."
}

for d in "$NF_DATA_DIR" "$NF_REDIS_DIR" "$INSTALL_DIR"; do
  if [ ! -d "$d" ]; then
    _mkdir "$d"
    ok "Created $d"
  else
    info "$d already exists"
  fi
done

# ── Docker daemon log rotation ────────────────────────────────────────────────
# Only configure if running as root on Linux with Docker as the runtime; the
# Docker Desktop VM has its own daemon config, and the compose file sets
# per-container log limits anyway.
if [ "$IS_ROOT" = "true" ] && [ "$DCMD" = "docker" ] && [ "$IS_MACOS" = "false" ]; then
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

if [ "$IS_MACOS" = "true" ]; then
  # No host networking in the Docker Desktop VM — the macOS variant publishes
  # ports and talks to redis over the compose network instead.
  COMPOSE_VARIANT="macos"
elif [ "$ROOTLESS" = "true" ]; then
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
  $SUDO_FS cp "$_tmp" "$COMPOSE_FILE"
  rm -f "$_tmp"
fi
ok "Downloaded compose/$COMPOSE_VARIANT.yml → $COMPOSE_FILE"

# Compute memory limits from available RAM (redis=33%, nf=50%).  /proc/meminfo
# is Linux-only; on macOS ask Docker for the VM's memory, since that — not the
# Mac's RAM — is what the containers actually get.
_mem_kb=$(awk '/MemTotal/ { print $2 }' /proc/meminfo 2>/dev/null || echo 0)
if [ "${_mem_kb:-0}" -le 0 ] 2>/dev/null && [ "$IS_MACOS" = "true" ]; then
  _mem_kb=$(echo "$DOCKER_INFO" | awk '/Total Memory:/ {
    v = $3
    sub(/GiB/, "", v); if (v != $3) { printf "%d", v * 1024 * 1024; exit }
    v = $3; sub(/MiB/, "", v); if (v != $3) { printf "%d", v * 1024; exit }
  }')
fi
if [ "${_mem_kb:-0}" -gt 0 ] 2>/dev/null; then
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
if [ -n "${NF_TZ:-}" ]; then
  printf 'NF_TZ=%s\n' "$NF_TZ" >>"$_env_tmp"
fi
if [ -w "$INSTALL_DIR" ]; then
  mv "$_env_tmp" "$ENV_FILE"
else
  $SUDO_FS cp "$_env_tmp" "$ENV_FILE"
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

# If the install ran long (slow pull, slow first boot) the setup token from the
# sign-in is at or near its one-hour expiry. Say so plainly and drop it rather
# than spending two minutes polling an exchange that cannot succeed.
if [ -n "$DEVICE_CODE" ] && _signin_expired; then
  warn "The sign-in from $(( $(_signin_age) / 60 )) minutes ago has expired — setup tokens are good for one hour."
  warn "Everything else is installed and running. Re-run the installer to license this box;"
  warn "it upgrades in place and will just ask you to sign in again."
  DEVICE_CODE=""
fi

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
if [ "$IS_DESKTOP" = "true" ]; then
  printf "\n"
  warn "Reminder — this is a Docker Desktop install, for evaluation only:"
  printf "      • BACnet discovery by broadcast will find nothing; add devices by unicast\n"
  printf "        address or register with a BBMD as a foreign device.\n"
  printf "      • NF is down whenever Docker Desktop is not running, including after a\n"
  printf "        reboot. Enable ${BOLD}Settings → General → Start Docker Desktop when you sign in${NC},\n"
  printf "        and restart with: cd %s && %s up -d\n" "$INSTALL_DIR" "$CCMD"
fi
printf "\n"
