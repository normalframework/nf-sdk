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
#   NF_RELEASE      "ga" or "enterprise" (auto-detected from login command)
#   NF_USERNAME     Registry username  }  skip the paste prompt when
#   NF_PASSWORD     Registry password  }  both are set via env vars
#   NF_REGISTRY     Registry hostname  }  (also set NF_REGISTRY or NF_RELEASE)

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
# Directory defaults are set after rootless detection below

GA_REGISTRY="normal.azurecr.io"
ENT_REGISTRY="normalframework.azurecr.io"
PORTAL_URL="https://portal.normal-online.net"

# ── Helpers ───────────────────────────────────────────────────────────────────
info()    { printf "${BLUE}[→]${NC} %s\n" "$*"; }
ok()      { printf "${GREEN}[✓]${NC} %s\n" "$*"; }
warn()    { printf "${YELLOW}[!]${NC} %s\n" "$*"; }
die()     { printf "${RED}[✗]${NC} %s\n" "$*" >&2; exit 1; }
step()    { printf "\n${BOLD}── %s ──${NC}\n" "$*"; }

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
      # Stop unattended-upgrades so it doesn't hold the dpkg lock for 5-10 min
      $SUDO systemctl stop unattended-upgrades apt-daily.service apt-daily-upgrade.service 2>/dev/null || true
      $SUDO systemctl kill --kill-who=all apt-daily.service apt-daily-upgrade.service 2>/dev/null || true
      # Wait for any lingering apt/dpkg lock to clear
      _waited=0
      while $SUDO fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do
        [ $_waited -eq 0 ] && info "Waiting for apt lock to clear..."
        sleep 2; _waited=$((_waited+2))
        [ $_waited -gt 60 ] && break
      done
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

# ── Registry login ────────────────────────────────────────────────────────────
step "Registry login"

# Parse "docker login REGISTRY -u USER -p PASS" into REGISTRY / NF_USERNAME / NF_PASSWORD
parse_login_cmd() {
  _raw="$1"
  _raw="${_raw#sudo }"        # strip optional sudo prefix
  _raw="${_raw#docker login}" # strip "docker login"
  _raw="${_raw# }"
  # shellcheck disable=SC2086
  set -- $_raw
  while [ $# -gt 0 ]; do
    case "$1" in
      -u|--username)  NF_USERNAME="$2"; shift 2 ;;
      -p|--password)  NF_PASSWORD="$2"; shift 2 ;;
      --username=*)   NF_USERNAME="${1#*=}"; shift ;;
      --password=*)   NF_PASSWORD="${1#*=}"; shift ;;
      --password-stdin) shift ;;
      -*)             shift ;;
      *)              REGISTRY="$1"; shift ;;  # first bare word = registry
    esac
  done
}

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

REGISTRY="${NF_REGISTRY:-}"

if [ -n "$NF_USERNAME" ] && [ -n "$NF_PASSWORD" ]; then
  # Env-var path: derive registry from NF_REGISTRY or NF_RELEASE
  if [ -z "$REGISTRY" ]; then
    case "${NF_RELEASE:-ga}" in
      [Ee]nterprise|[Ee]nt) REGISTRY="$ENT_REGISTRY" ;;
      *) REGISTRY="$GA_REGISTRY" ;;
    esac
  fi
else
  # Interactive: ask for the full docker login command from the portal
  printf "\nGet your ${BOLD}docker login${NC} command from the Normal Portal:\n"
  printf "  1. Open  ${BOLD}%s${NC}\n" "$PORTAL_URL"
  printf "  2. Log in → ${BOLD}Settings → API Keys${NC}\n"
  printf "  3. Copy the login command and paste it below\n\n"
  ask _LOGIN_CMD "Paste docker login command"
  parse_login_cmd "$_LOGIN_CMD"
fi

[ -n "$NF_USERNAME" ] || die "Could not determine registry username."
[ -n "$NF_PASSWORD" ] || die "Could not determine registry password."
[ -n "$REGISTRY"    ] || die "Could not determine registry from login command."

# Detect GA vs enterprise from the registry hostname
case "$REGISTRY" in
  *normalframework.azurecr.io*) NF_RELEASE="enterprise" ;;
  *normal.azurecr.io*)          NF_RELEASE="ga" ;;
  *)                            NF_RELEASE="${NF_RELEASE:-ga}" ;;
esac
ok "Registry: $REGISTRY ($NF_RELEASE)"

# Skip login if already authenticated with a valid token
if check_auth "$REGISTRY"; then
  ok "Already authenticated with $REGISTRY (token valid)"
else
  info "Logging in to $REGISTRY..."
  if printf '%s' "$NF_PASSWORD" \
      | $SUDO_DCMD $DCMD login --username "$NF_USERNAME" --password-stdin "$REGISTRY"; then
    ok "Authenticated with $REGISTRY"
  else
    die "Login failed. Check your credentials and try again."
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
ENV_FILE="$INSTALL_DIR/.env"

if [ "$ROOTLESS" = "true" ]; then
  COMPOSE_VARIANT="linux-rootless"
else
  COMPOSE_VARIANT="linux"
fi

if [ -f "$COMPOSE_FILE" ]; then
  warn "docker-compose.yml already exists at $COMPOSE_FILE — skipping"
  warn "Delete it and re-run to pull a fresh copy"
else
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
fi

# Write .env so docker compose picks up the right values on future runs too
_env_tmp="$(mktemp)"
cat >"$_env_tmp" <<EOF
REGISTRY=${REGISTRY}
NF_TAG=${NF_TAG}
NF_PORT=${NF_PORT}
NF_DATA_DIR=${NF_DATA_DIR}
NF_REDIS_DIR=${NF_REDIS_DIR}
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
$SUDO_DCMD $CCMD pull --quiet
ok "Images pulled"

# ── Start NF ──────────────────────────────────────────────────────────────────
step "Starting Normal Framework"
$SUDO_DCMD $CCMD up -d --quiet-pull
ok "Containers started"

# ── Wait for console ──────────────────────────────────────────────────────────
step "Waiting for console to come up"
MAX_WAIT=90
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

# ── Done ──────────────────────────────────────────────────────────────────────
printf "\n${GREEN}${BOLD}"
printf "╔══════════════════════════════════════════════╗\n"
printf "║   Normal Framework is running!               ║\n"
printf "╚══════════════════════════════════════════════╝\n"
printf "${NC}\n"
printf "  ${BOLD}Console${NC}   http://localhost:%s\n" "$NF_PORT"
printf "  ${BOLD}Data${NC}      %s\n" "$NF_DATA_DIR"
printf "  ${BOLD}Compose${NC}   %s\n" "$COMPOSE_FILE"
printf "\n"
printf "  Manage:  cd %s && %s [logs|ps|down|up]\n" "$INSTALL_DIR" "$CCMD"
if [ "$NF_RELEASE" = "ga" ]; then
  printf "\n  ${YELLOW}GA release:${NC} activate your license at %s\n" "$PORTAL_URL"
fi
printf "\n"
