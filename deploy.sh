#!/usr/bin/env bash
#
# deploy.sh — provision dcrdata on a fresh Debian 13 / Ubuntu 24.04 LTS VPS
#             (amd64 or arm64).
#
# Installs Go, PostgreSQL, dcrd, and Caddy (automatic HTTPS); creates the
# database; builds the explorer; and runs it all as hardened systemd services
# behind Caddy. The front end is plain CSS + native ES modules (no Node.js).
# Safe to re-run: every step is idempotent, so this doubles as an updater.
# The first run records its network/topology choices in
# /etc/default/dcrdata-deploy; re-runs inherit them, so a bare
# `sudo ./deploy.sh` upgrades in place. Changing network or dcrd topology
# against the recorded state aborts with an explanation (delete the state
# file after wiping the data to start over).
#
# Usage (run as root or with sudo):
#   sudo ./deploy.sh --domain explorer.example.com
#   sudo ./deploy.sh --domain explorer.example.com --repo https://github.com/me/dcrdata
#   sudo ./deploy.sh --http                       # no domain: serve plain HTTP on :80
#   sudo ./deploy.sh --domain x --skip-dcrd \      # use an existing dcrd node
#        --dcrdserv 10.0.0.5:9109 --dcrduser u --dcrdpass p --dcrdcert /path/rpc.cert
#   sudo ./deploy.sh                               # re-run: upgrade with recorded options
#
# Options:
#   --domain <host>    Domain to serve (Caddy provisions a TLS cert for it).
#   --http             Serve plain HTTP on :80 instead of HTTPS (for testing).
#   --repo <url>       Git repo to deploy        (default: jzbz/dcrdata-remix; use a fork to override).
#   --testnet          Index testnet instead of mainnet.
#   --skip-dcrd        Do not install dcrd; connect to an existing node.
#   --dcrdserv <addr>  Existing dcrd RPC host:port (with --skip-dcrd).
#   --dcrduser <user>  Existing dcrd RPC username  (with --skip-dcrd).
#   --dcrdpass <pass>  Existing dcrd RPC password  (with --skip-dcrd).
#   --dcrdcert <path>  Existing dcrd rpc.cert file (with --skip-dcrd).
#   --go-version <v>   Go toolchain version        (default: 1.26.4).
#   --dcrd-version <v> dcrd version to go install  (default: latest).
#   --listen <addr>    dcrdata internal listen     (default: 127.0.0.1:7777).
#   -h, --help         Show this help.
#
set -euo pipefail

# ---- Configuration --------------------------------------------------------

GO_VERSION="1.26.4"
DCRD_VERSION="latest"
REPO_URL="https://github.com/jzbz/dcrdata-remix"
LISTEN="127.0.0.1:7777"
DOMAIN=""
HTTP_ONLY=0
TESTNET=0
SKIP_DCRD=0
EXT_DCRDSERV=""; EXT_DCRDUSER=""; EXT_DCRDPASS=""; EXT_DCRDCERT=""
REPO_SET=0; LISTEN_SET=0; MODE_SET=0

# Options recorded by the previous run; re-runs inherit them and refuse
# incompatible changes (see the state checks after argument parsing).
STATE_FILE=/etc/default/dcrdata-deploy

DATA_USER="dcrdata"
DATA_HOME="/opt/dcrdata"
APP_DIR="${DATA_HOME}/app"
APP_CMD="${APP_DIR}/cmd/dcrdata"
APPDATA="${DATA_HOME}/appdata"
DCRD_CERT_DST="${DATA_HOME}/dcrd-rpc.cert"

NODE_USER="dcrd"
NODE_HOME="/opt/dcrd"
NODE_DCRD_DIR="${NODE_HOME}/.dcrd"

GO=/usr/local/go/bin/go

# ---- Logging --------------------------------------------------------------

if [[ -t 1 ]]; then
  C_BLUE=$'\e[34m'; C_GREEN=$'\e[32m'; C_RED=$'\e[31m'; C_DIM=$'\e[2m'; C_OFF=$'\e[0m'
else
  C_BLUE=""; C_GREEN=""; C_RED=""; C_DIM=""; C_OFF=""
fi
log()  { printf '%s==>%s %s\n' "$C_BLUE" "$C_OFF" "$*"; }
ok()   { printf '%s  ✓%s %s\n' "$C_GREEN" "$C_OFF" "$*"; }
warn() { printf '%s  !%s %s\n' "$C_RED" "$C_OFF" "$*" >&2; }
die()  { printf '%serror:%s %s\n' "$C_RED" "$C_OFF" "$*" >&2; exit 1; }

# Print the header comment block (everything after the shebang up to the first
# non-comment line) as usage text.
usage() {
  local line
  while IFS= read -r line; do
    case "$line" in
      '#!'*) continue ;;
      '# '*) printf '%s\n' "${line#'# '}" ;;
      '#')   printf '\n' ;;
      '#'*)  printf '%s\n' "${line#'#'}" ;;
      *)     break ;;
    esac
  done < "$0"
  exit "${1:-0}"
}

# ---- Argument parsing -----------------------------------------------------

while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain)       DOMAIN="${2:?--domain needs a value}"; MODE_SET=1; shift 2 ;;
    --http)         HTTP_ONLY=1; MODE_SET=1; shift ;;
    --repo)         REPO_URL="${2:?--repo needs a value}"; REPO_SET=1; shift 2 ;;
    --testnet)      TESTNET=1; shift ;;
    --skip-dcrd)    SKIP_DCRD=1; shift ;;
    --dcrdserv)     EXT_DCRDSERV="${2:?}"; shift 2 ;;
    --dcrduser)     EXT_DCRDUSER="${2:?}"; shift 2 ;;
    --dcrdpass)     EXT_DCRDPASS="${2:?}"; shift 2 ;;
    --dcrdcert)     EXT_DCRDCERT="${2:?}"; shift 2 ;;
    --go-version)   GO_VERSION="${2:?}"; shift 2 ;;
    --dcrd-version) DCRD_VERSION="${2:?}"; shift 2 ;;
    --listen)       LISTEN="${2:?}"; LISTEN_SET=1; shift 2 ;;
    -h|--help)      usage 0 ;;
    *)              warn "unknown option: $1"; usage 1 ;;
  esac
done

[[ $EUID -eq 0 ]] || die "must run as root (try: sudo $0 ...)"

# ---- Recorded deployment state --------------------------------------------
# The config files are regenerated from this invocation's flags on every run,
# while the database and dcrd data dir carry the *previous* run's choices.
# Inherit unspecified options from the recorded state and refuse changes that
# would desync the configs from the data (mainnet configs over a testnet
# database, a fresh local dcrd over an external-node install, ...).

PREV_NETWORK=""; PREV_SKIP_DCRD=""; PREV_DOMAIN=""; PREV_HTTP_ONLY=""
PREV_REPO_URL=""; PREV_LISTEN=""; PREV_DCRDSERV=""; PREV_DCRDUSER=""
if [[ -f "$STATE_FILE" ]]; then
  PREV_NETWORK=$(sed -n 's/^NETWORK=//p' "$STATE_FILE")
  PREV_SKIP_DCRD=$(sed -n 's/^SKIP_DCRD=//p' "$STATE_FILE")
  PREV_DOMAIN=$(sed -n 's/^DOMAIN=//p' "$STATE_FILE")
  PREV_HTTP_ONLY=$(sed -n 's/^HTTP_ONLY=//p' "$STATE_FILE")
  PREV_REPO_URL=$(sed -n 's/^REPO_URL=//p' "$STATE_FILE")
  PREV_LISTEN=$(sed -n 's/^LISTEN=//p' "$STATE_FILE")
  PREV_DCRDSERV=$(sed -n 's/^EXT_DCRDSERV=//p' "$STATE_FILE")
  PREV_DCRDUSER=$(sed -n 's/^EXT_DCRDUSER=//p' "$STATE_FILE")

  # Network: absence of --testnet is ambiguous (it is also the mainnet
  # default), so inherit the recorded network; an explicit --testnet against
  # a mainnet install is a real conflict.
  if [[ $TESTNET -eq 0 && "$PREV_NETWORK" == "testnet" ]]; then
    TESTNET=1
    log "Inheriting testnet from the previous deploy (${STATE_FILE})"
  elif [[ $TESTNET -eq 1 && "$PREV_NETWORK" == "mainnet" ]]; then
    die "this host is deployed for mainnet; --testnet would point the configs at a
       database and dcrd data dir full of mainnet data. Wipe them and delete
       ${STATE_FILE} to redeploy on testnet."
  fi

  # dcrd topology: same rules.
  if [[ $SKIP_DCRD -eq 0 && "$PREV_SKIP_DCRD" == "1" ]]; then
    SKIP_DCRD=1
    log "Inheriting --skip-dcrd (external node) from the previous deploy"
  elif [[ $SKIP_DCRD -eq 1 && "$PREV_SKIP_DCRD" == "0" ]]; then
    die "this host runs its own dcrd; switching to --skip-dcrd would leave it
       running and orphaned. Stop/disable the dcrd service and delete
       ${STATE_FILE} first."
  fi

  # Serving mode and repo: plain preferences, inherit when not specified.
  if [[ $MODE_SET -eq 0 ]]; then
    DOMAIN="$PREV_DOMAIN"
    [[ "$PREV_HTTP_ONLY" == "1" ]] && HTTP_ONLY=1
  fi
  [[ $REPO_SET -eq 0 && -n "$PREV_REPO_URL" ]] && REPO_URL="$PREV_REPO_URL"
  [[ $LISTEN_SET -eq 0 && -n "$PREV_LISTEN" ]] && LISTEN="$PREV_LISTEN"
fi

if [[ $HTTP_ONLY -eq 0 && -z "$DOMAIN" ]]; then
  die "provide --domain <host>, or --http to serve plain HTTP for testing"
fi
if [[ $SKIP_DCRD -eq 1 && -z "$PREV_DCRDSERV" ]]; then
  # First skip-dcrd run: all four connection parameters are required. Re-runs
  # inherit the server/user from the state file and the password/cert from the
  # previous run's dcrdata.conf and cert copy.
  [[ -n "$EXT_DCRDSERV" && -n "$EXT_DCRDUSER" && -n "$EXT_DCRDPASS" && -n "$EXT_DCRDCERT" ]] \
    || die "--skip-dcrd requires --dcrdserv, --dcrduser, --dcrdpass and --dcrdcert"
fi
if [[ -n "$EXT_DCRDCERT" ]]; then
  [[ -f "$EXT_DCRDCERT" ]] || die "dcrd cert not found: $EXT_DCRDCERT"
fi

NETWORK="mainnet"; RPC_PORT="9109"
if [[ $TESTNET -eq 1 ]]; then NETWORK="testnet"; RPC_PORT="19109"; fi

# ---- 1. Base packages -----------------------------------------------------

log "Installing base packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq git ufw curl gnupg ca-certificates openssl >/dev/null
ok "base packages installed"

# ---- 2. Firewall ----------------------------------------------------------

log "Configuring firewall (ufw)"
# Never enable a default-deny firewall without a working SSH rule: the OpenSSH
# app profile may be missing, and sshd may listen on a nonstandard port. Find
# the ports sshd actually listens on (fall back to sshd_config, then 22) and
# abort loudly if a rule cannot be added — a silent failure here locks the
# operator out of the VPS.
SSH_PORTS=$(ss -tlnpH 2>/dev/null | awk '/"sshd"/ {sub(/.*:/, "", $4); print $4}' | sort -un)
if [[ -z "$SSH_PORTS" ]]; then
  SSH_PORTS=$(sed -n 's/^[[:space:]]*[Pp]ort[[:space:]]\+//p' /etc/ssh/sshd_config 2>/dev/null | awk '{print $1}' | sort -un)
fi
[[ -z "$SSH_PORTS" ]] && SSH_PORTS="22"
for p in $SSH_PORTS; do
  ufw allow "${p}/tcp" >/dev/null || die "failed to allow SSH port ${p}/tcp in ufw; not enabling the firewall"
done
ufw allow 80/tcp  >/dev/null
ufw allow 443/tcp >/dev/null
ufw --force enable >/dev/null
ok "SSH ($(echo "$SSH_PORTS" | tr '\n' ' ' | sed 's/ $//')), 80/tcp and 443/tcp allowed"

# ---- 3. Go toolchain ------------------------------------------------------

case "$(uname -m)" in
  x86_64|amd64)  GO_ARCH="amd64" ;;
  aarch64|arm64) GO_ARCH="arm64" ;;
  *)             die "unsupported CPU architecture: $(uname -m)" ;;
esac

if $GO version 2>/dev/null | grep -q "go${GO_VERSION} "; then
  ok "Go ${GO_VERSION} already installed"
else
  log "Installing Go ${GO_VERSION} (${GO_ARCH})"
  tmp="$(mktemp -d)"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o "${tmp}/go.tar.gz"
  # Verify against the published checksum: this tarball is extracted as root
  # and its toolchain builds everything else in the deploy.
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz.sha256" -o "${tmp}/go.tar.gz.sha256"
  echo "$(awk '{print $1}' "${tmp}/go.tar.gz.sha256")  ${tmp}/go.tar.gz" | sha256sum -c --quiet - \
    || die "Go tarball sha256 mismatch (go${GO_VERSION}.linux-${GO_ARCH}.tar.gz)"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "${tmp}/go.tar.gz"
  rm -rf "$tmp"
  # shellcheck disable=SC2016
  echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
  ok "Go installed: $($GO version)"
fi

# ---- 4. PostgreSQL --------------------------------------------------------

if command -v psql >/dev/null 2>&1; then
  ok "PostgreSQL already installed"
else
  log "Installing PostgreSQL"
  apt-get install -y -qq postgresql >/dev/null
  ok "PostgreSQL installed"
fi

# Service user must exist before the DB role (peer auth maps OS user -> role).
if ! id "$DATA_USER" >/dev/null 2>&1; then
  log "Creating service user '${DATA_USER}'"
  useradd --system --create-home --home-dir "$DATA_HOME" --shell /usr/sbin/nologin "$DATA_USER"
  ok "created ${DATA_USER}"
fi

log "Ensuring PostgreSQL role and database"
if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='${DATA_USER}'" | grep -q 1; then
  sudo -u postgres createuser "$DATA_USER"
fi
if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='${DATA_USER}'" | grep -q 1; then
  sudo -u postgres createdb -O "$DATA_USER" "$DATA_USER"
fi
ok "role/database '${DATA_USER}' ready"

log "Applying PostgreSQL tuning (scaled to RAM)"
RAM_MB=$(awk '/MemTotal/ {print int($2/1024)}' /proc/meminfo)
SHARED=$(( RAM_MB / 4 )); CACHE=$(( RAM_MB * 3 / 4 )); MAINT=$(( RAM_MB / 16 ))
(( MAINT > 1024 )) && MAINT=1024
# Target the running cluster, not just the newest version directory (they can
# differ after a distro major-version upgrade leaves both installed).
PG_VER=""
if command -v pg_lsclusters >/dev/null 2>&1; then
  PG_VER=$(pg_lsclusters --no-header 2>/dev/null | awk '$4 ~ /online/ {print $1; exit}')
fi
if [[ -z "$PG_VER" ]]; then
  # shellcheck disable=SC2012  # version dirs are numeric (e.g. 16); ls + version-sort is fine
  PG_VER=$(ls /etc/postgresql | sort -V | tail -1)
fi
PG_CONF="/etc/postgresql/${PG_VER}/main/conf.d/dcrdata.conf"
PG_CONF_NEW="$(mktemp)"
cat > "$PG_CONF_NEW" <<EOF
# dcrdata tuning (generated by deploy.sh for ${RAM_MB} MB RAM).
synchronous_commit = off
max_connections = 32
shared_buffers = ${SHARED}MB
effective_cache_size = ${CACHE}MB
maintenance_work_mem = ${MAINT}MB
work_mem = 28MB
wal_buffers = 16MB
max_wal_size = 2GB
min_wal_size = 1GB
checkpoint_completion_target = 0.9
random_page_cost = 1.1
effective_io_concurrency = 200
EOF
# Only install and restart when the tuning actually changed: a needless
# restart on every re-run drops dcrdata's connections mid-sync.
if [[ -f "$PG_CONF" ]] && cmp -s "$PG_CONF_NEW" "$PG_CONF"; then
  rm -f "$PG_CONF_NEW"
  ok "PostgreSQL tuning unchanged (cluster ${PG_VER})"
else
  install -m 644 "$PG_CONF_NEW" "$PG_CONF"
  rm -f "$PG_CONF_NEW"
  systemctl restart postgresql
  ok "PostgreSQL tuned (shared_buffers=${SHARED}MB, effective_cache_size=${CACHE}MB)"
fi

# ---- 5. dcrd --------------------------------------------------------------

if [[ $SKIP_DCRD -eq 1 ]]; then
  # Explicit flags win; a bare re-run inherits the server/user recorded in the
  # state file, the password from the previous run's dcrdata.conf, and the
  # already-copied cert.
  DCRD_SERV="${EXT_DCRDSERV:-$PREV_DCRDSERV}"
  DCRD_RPCUSER="${EXT_DCRDUSER:-$PREV_DCRDUSER}"
  DCRD_RPCPASS="$EXT_DCRDPASS"
  if [[ -z "$DCRD_RPCPASS" && -f "${APPDATA}/dcrdata.conf" ]]; then
    DCRD_RPCPASS=$(sed -n 's/^dcrdpass=//p' "${APPDATA}/dcrdata.conf" | head -1)
  fi
  [[ -n "$DCRD_SERV" && -n "$DCRD_RPCUSER" && -n "$DCRD_RPCPASS" ]] \
    || die "external dcrd parameters incomplete; pass --dcrdserv/--dcrduser/--dcrdpass"
  log "Using existing dcrd at ${DCRD_SERV}"
  if [[ -n "$EXT_DCRDCERT" ]]; then
    install -m 644 -o "$DATA_USER" -g "$DATA_USER" "$EXT_DCRDCERT" "$DCRD_CERT_DST"
    ok "external dcrd cert copied to ${DCRD_CERT_DST}"
  else
    [[ -f "$DCRD_CERT_DST" ]] || die "no dcrd cert at ${DCRD_CERT_DST}; pass --dcrdcert"
    ok "reusing external dcrd cert at ${DCRD_CERT_DST}"
  fi
else
  if ! id "$NODE_USER" >/dev/null 2>&1; then
    log "Creating service user '${NODE_USER}'"
    useradd --system --create-home --home-dir "$NODE_HOME" --shell /usr/sbin/nologin "$NODE_USER"
  fi

  log "Installing dcrd (${DCRD_VERSION})"
  DCRD_VER_BEFORE=$(/usr/local/bin/dcrd --version 2>/dev/null | head -1 || true)
  GOBIN=/usr/local/bin GOTOOLCHAIN=local "$GO" install "github.com/decred/dcrd@${DCRD_VERSION}"
  DCRD_VER_AFTER=$(/usr/local/bin/dcrd --version 2>/dev/null | head -1 || true)
  ok "dcrd installed (${DCRD_VER_AFTER:-$DCRD_VERSION})"

  # dcrctl is an optional admin CLI. It was moved out of the dcrd module into its
  # own module (decred.org/dcrctl), versioned independently of dcrd — so it is
  # installed at @latest, not @${DCRD_VERSION}. Best-effort: a hiccup fetching it
  # must not abort the deploy, since dcrd and dcrdata don't need it to run.
  log "Installing dcrctl (optional admin CLI)"
  if GOBIN=/usr/local/bin GOTOOLCHAIN=local "$GO" install "decred.org/dcrctl@latest"; then
    ok "dcrctl installed"
  else
    warn "dcrctl install failed (optional); continuing without it"
  fi

  # Reuse the existing RPC password on re-runs so dcrdata's config stays valid.
  # (sed, not awk -F=: the password itself may contain '=' characters.)
  install -d -o "$NODE_USER" -g "$NODE_USER" "$NODE_DCRD_DIR"
  if [[ -f "${NODE_DCRD_DIR}/dcrd.conf" ]]; then
    DCRD_RPCPASS=$(sed -n 's/^rpcpass=//p' "${NODE_DCRD_DIR}/dcrd.conf" | head -1)
  fi
  DCRD_RPCUSER="dcrd"
  DCRD_RPCPASS="${DCRD_RPCPASS:-$(openssl rand -hex 24)}"
  DCRD_SERV="127.0.0.1:${RPC_PORT}"

  DCRD_CONF_NEW="$(mktemp)"
  cat > "$DCRD_CONF_NEW" <<EOF
[Application Options]
rpcuser=${DCRD_RPCUSER}
rpcpass=${DCRD_RPCPASS}
rpclisten=127.0.0.1:${RPC_PORT}
txindex=1
$( [[ $TESTNET -eq 1 ]] && echo "testnet=1" )
EOF
  DCRD_NEEDS_RESTART=0
  if [[ ! -f "${NODE_DCRD_DIR}/dcrd.conf" ]] || ! cmp -s "$DCRD_CONF_NEW" "${NODE_DCRD_DIR}/dcrd.conf"; then
    DCRD_NEEDS_RESTART=1
  fi
  install -m 600 -o "$NODE_USER" -g "$NODE_USER" "$DCRD_CONF_NEW" "${NODE_DCRD_DIR}/dcrd.conf"
  rm -f "$DCRD_CONF_NEW"
  chown -R "$NODE_USER:$NODE_USER" "$NODE_DCRD_DIR"

  cat > /etc/systemd/system/dcrd.service <<EOF
[Unit]
Description=dcrd Decred full node
After=network-online.target
Wants=network-online.target

[Service]
User=${NODE_USER}
Group=${NODE_USER}
ExecStart=/usr/local/bin/dcrd
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=${NODE_HOME}

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable dcrd >/dev/null 2>&1 || true
  # Restart only when the config or binary actually changed — a node mid-sync
  # should not be bounced by a no-op re-run, but stale-config/stale-binary
  # nodes previously kept running the old settings forever, so the "updater"
  # never really updated dcrd.
  if [[ -n "$DCRD_VER_BEFORE" && "$DCRD_VER_BEFORE" != "$DCRD_VER_AFTER" ]]; then
    DCRD_NEEDS_RESTART=1
  fi
  if ! systemctl is-active --quiet dcrd; then
    systemctl start dcrd
  elif [[ $DCRD_NEEDS_RESTART -eq 1 ]]; then
    log "dcrd config/binary changed; restarting dcrd"
    systemctl restart dcrd
  fi
  ok "dcrd service running (${NETWORK}, RPC 127.0.0.1:${RPC_PORT})"

  # dcrd writes rpc.cert on first start; wait for it, then expose to dcrdata.
  log "Waiting for dcrd to generate its RPC certificate"
  for _ in $(seq 1 30); do
    [[ -f "${NODE_DCRD_DIR}/rpc.cert" ]] && break
    sleep 1
  done
  [[ -f "${NODE_DCRD_DIR}/rpc.cert" ]] || die "dcrd rpc.cert not generated; check 'journalctl -u dcrd'"
  install -m 644 -o "$DATA_USER" -g "$DATA_USER" "${NODE_DCRD_DIR}/rpc.cert" "$DCRD_CERT_DST"
  ok "dcrd RPC cert published to ${DCRD_CERT_DST}"
fi

# ---- 6. Fetch & build dcrdata ---------------------------------------------

if [[ -d "${APP_DIR}/.git" ]]; then
  ACTION="upgrade"
  # Honor an explicit --repo change on upgrades; previously the flag was
  # silently ignored and the old origin kept being deployed.
  CUR_ORIGIN=$(sudo -u "$DATA_USER" git -C "$APP_DIR" remote get-url origin)
  if [[ $REPO_SET -eq 1 && "$CUR_ORIGIN" != "$REPO_URL" ]]; then
    log "Switching deploy repo: ${CUR_ORIGIN} -> ${REPO_URL}"
    sudo -u "$DATA_USER" git -C "$APP_DIR" remote set-url origin "$REPO_URL"
  fi
  log "Updating checkout (force-sync to remote; local edits to tracked files discarded)"
  sudo -u "$DATA_USER" git -C "$APP_DIR" fetch --depth 1 origin
  sudo -u "$DATA_USER" git -C "$APP_DIR" reset --hard '@{u}'
else
  ACTION="install"
  log "Cloning ${REPO_URL}"
  git clone --depth 1 "$REPO_URL" "$APP_DIR"
  chown -R "$DATA_USER:$DATA_USER" "$DATA_HOME"
fi

# Back end: build to a temp file, swapped into place atomically only on success
# so a broken build never takes down the running service. -buildvcs=false: the
# build runs as root over a checkout owned by the service user, which Go's VCS
# stamping would reject as "dubiously owned".
log "Building dcrdata"
( cd "$APP_CMD" && GOTOOLCHAIN=local "$GO" build -buildvcs=false -o "${APP_CMD}/dcrdata.new" . )
chown "$DATA_USER:$DATA_USER" "${APP_CMD}/dcrdata.new"
ok "dcrdata built"

# The front end is plain CSS + native ES modules served straight from
# cmd/dcrdata/public — no bundler, no Node.js, nothing to build.

# ---- 7. Configure dcrdata -------------------------------------------------

log "Writing dcrdata config"
install -d -o "$DATA_USER" -g "$DATA_USER" "$APPDATA"
cat > "${APPDATA}/dcrdata.conf" <<EOF
[Application Options]
$( [[ $TESTNET -eq 1 ]] && echo "testnet=1" )

# dcrd RPC
dcrduser=${DCRD_RPCUSER}
dcrdpass=${DCRD_RPCPASS}
dcrdserv=${DCRD_SERV}
dcrdcert=${DCRD_CERT_DST}

# PostgreSQL (peer auth over the local socket — no password)
pghost=/run/postgresql
pguser=${DATA_USER}
pgdbname=${DATA_USER}

# Web server — loopback; Caddy fronts it
apilisten=${LISTEN}
apiproto=http

# Behind a reverse proxy
userealip=true
trustproxy=true
$( [[ $HTTP_ONLY -eq 0 ]] && echo "allowedhost=${DOMAIN}" )
EOF
chown "$DATA_USER:$DATA_USER" "${APPDATA}/dcrdata.conf"
chmod 600 "${APPDATA}/dcrdata.conf"
ok "config written to ${APPDATA}/dcrdata.conf"

# ---- 8. dcrdata systemd service -------------------------------------------

log "Writing dcrdata systemd unit"
cat > /etc/systemd/system/dcrdata.service <<EOF
[Unit]
Description=dcrdata Decred block explorer
After=network-online.target postgresql.service dcrd.service
Wants=network-online.target

[Service]
User=${DATA_USER}
Group=${DATA_USER}
WorkingDirectory=${APP_CMD}
ExecStart=${APP_CMD}/dcrdata --appdata=${APPDATA}
Restart=on-failure
RestartSec=10
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=${DATA_HOME}

[Install]
WantedBy=multi-user.target
EOF

# Atomically swap in the freshly built binary, then (re)start.
mv -f "${APP_CMD}/dcrdata.new" "${APP_CMD}/dcrdata"
systemctl daemon-reload
systemctl enable dcrdata >/dev/null 2>&1 || true
systemctl restart dcrdata
ok "dcrdata service running"

# ---- 9. Caddy ------------------------------------------------------------

if command -v caddy >/dev/null 2>&1; then
  ok "Caddy already installed ($(caddy version | head -1))"
else
  log "Installing Caddy"
  # --yes: a prior partial run may have left the keyring file behind, and
  # gpg --dearmor refuses to overwrite without it, breaking re-runs.
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
    | gpg --dearmor --yes -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
    > /etc/apt/sources.list.d/caddy-stable.list
  apt-get update -qq
  apt-get install -y -qq caddy >/dev/null
  ok "Caddy installed"
fi

log "Writing Caddyfile"
SITE="$DOMAIN"; [[ $HTTP_ONLY -eq 1 ]] && SITE=":80"
CADDYFILE=/etc/caddy/Caddyfile
CADDY_MARKER="# Managed by dcrdata deploy.sh — remove this line to take manual control."
CADDY_NEW="$(mktemp)"
# Tabs are required for Caddyfile block indentation. The header_up lines
# matter: dcrdata runs with userealip=true, and its RealIP middleware prefers
# True-Client-IP and X-Real-IP over X-Forwarded-For — Caddy manages only the
# X-Forwarded-* headers, so without these a client could spoof its address.
cat > "$CADDY_NEW" <<EOF
${CADDY_MARKER}
${SITE} {
	encode zstd gzip
	reverse_proxy ${LISTEN} {
		header_up -True-Client-IP
		header_up X-Real-IP {remote_host}
	}

	@assets path /css/* /js/* /fonts/* /images/*
	header @assets Cache-Control "public, max-age=604800"

	header {
		Referrer-Policy "same-origin"
		X-Content-Type-Options "nosniff"
		X-Frame-Options "SAMEORIGIN"
$( [[ $HTTP_ONLY -eq 0 ]] && printf '\t\tStrict-Transport-Security "max-age=31536000; includeSubDomains"' )
		-Server
	}
}
EOF
# Overwrite only files this script owns (first line = managed marker). An
# existing file without the marker may carry operator edits — the Cloudflare
# or origin-cert changes DEPLOY.md describes, or a pre-marker script write —
# and telling those apart is not possible, so never overwrite it: warn with
# the important directives instead. Delete the file and re-run (or add the
# marker line yourself) to hand it back to the script.
if [[ -f "$CADDYFILE" ]] && ! grep -qF "$CADDY_MARKER" "$CADDYFILE"; then
  rm -f "$CADDY_NEW"
  warn "existing ${CADDYFILE} is not managed by this script; leaving it untouched."
  warn "ensure its reverse_proxy block strips client IP spoofing:"
  warn "    header_up -True-Client-IP"
  warn "    header_up X-Real-IP {remote_host}"
  warn "delete ${CADDYFILE} and re-run to regenerate a managed one."
elif [[ -f "$CADDYFILE" ]] && cmp -s "$CADDY_NEW" "$CADDYFILE"; then
  rm -f "$CADDY_NEW"
  ok "Caddyfile unchanged"
else
  install -m 644 "$CADDY_NEW" "$CADDYFILE"
  rm -f "$CADDY_NEW"
  caddy validate --config "$CADDYFILE" >/dev/null
  systemctl reload caddy
  ok "Caddy configured for ${SITE}"
fi

# ---- Record deployment state ----------------------------------------------

# Later re-runs inherit these choices and refuse incompatible ones; see the
# state checks after argument parsing. No secrets are stored here.
cat > "$STATE_FILE" <<EOF
# Recorded by dcrdata deploy.sh — used by re-runs to inherit these options and
# refuse network/topology changes that would desync configs from data.
# Delete this file (after wiping the data) to reconfigure from scratch.
NETWORK=${NETWORK}
SKIP_DCRD=${SKIP_DCRD}
DOMAIN=${DOMAIN}
HTTP_ONLY=${HTTP_ONLY}
REPO_URL=${REPO_URL}
LISTEN=${LISTEN}
EXT_DCRDSERV=${DCRD_SERV}
EXT_DCRDUSER=${DCRD_RPCUSER}
EOF
chmod 644 "$STATE_FILE"

# ---- Done -----------------------------------------------------------------

echo
if [[ "$ACTION" == "upgrade" ]]; then
  ok "Upgrade complete — dcrdata rebuilt and restarted."
else
  ok "Deployment complete (${NETWORK})."
fi
if [[ $HTTP_ONLY -eq 1 ]]; then
  IP="$(curl -fsSL https://api.ipify.org 2>/dev/null || echo '<server-ip>')"
  echo "   ${C_DIM}Site:${C_OFF}  http://${IP}/"
else
  echo "   ${C_DIM}Site:${C_OFF}  https://${DOMAIN}/   ${C_DIM}(TLS provisions on first request)${C_OFF}"
fi
cat <<EOF
   ${C_DIM}dcrd:${C_OFF}     journalctl -u dcrd -f
   ${C_DIM}dcrdata:${C_OFF}  journalctl -u dcrdata -f
   ${C_DIM}caddy:${C_OFF}    journalctl -u caddy -f

First run syncs dcrd and then builds dcrdata's PostgreSQL index from genesis —
expect many hours. The site serves a "syncing" status page until it completes.
EOF
