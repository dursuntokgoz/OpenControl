#!/bin/sh
# ServerPanel installer — POSIX sh, idempotent.
#
# Usage: install.sh [artifact-dir]   (artifact dir defaults to /opt/serverpanel-pkg)
#
# Expected artifact layout:
#   bin/panel-api  bin/panel-agent  bin/panelctl
#   web/           systemd/*.service
set -eu

SRC="${1:-/opt/serverpanel-pkg}"
PREFIX=/usr/local/bin
CONF_DIR=/etc/serverpanel
STATE_DIR=/var/lib/serverpanel
WEB_DIR=/usr/share/serverpanel/web
RUN_DIR=/run/serverpanel

[ "$(id -u)" = "0" ] || { echo "install.sh must run as root"; exit 1; }
[ -x "$SRC/bin/panel-api" ] || { echo "missing $SRC/bin/panel-api"; exit 1; }

echo "[install] creating panel user"
if ! id panel >/dev/null 2>&1; then
    useradd --system --home-dir "$STATE_DIR" --shell /usr/sbin/nologin panel
fi

echo "[install] directories"
mkdir -p "$CONF_DIR" "$STATE_DIR" "$WEB_DIR" "$RUN_DIR"
chown panel:panel "$STATE_DIR"
chmod 750 "$STATE_DIR"

echo "[install] binaries"
install -m 0755 "$SRC/bin/panel-api"   "$PREFIX/panel-api"
install -m 0755 "$SRC/bin/panel-agent" "$PREFIX/panel-agent"
install -m 0755 "$SRC/bin/panelctl"    "$PREFIX/panelctl"

echo "[install] web assets"
rm -rf "$WEB_DIR.new"
mkdir -p "$WEB_DIR.new"
cp -r "$SRC/web/." "$WEB_DIR.new/"
rm -rf "$WEB_DIR.old"
[ -e "$WEB_DIR" ] && mv "$WEB_DIR" "$WEB_DIR.old"
mv "$WEB_DIR.new" "$WEB_DIR"
rm -rf "$WEB_DIR.old"

echo "[install] configuration"
if [ ! -f "$CONF_DIR/config.yaml" ]; then
    TOKEN="$(head -c 32 /dev/urandom | base64 | tr -d '=+/' | head -c 43)"
    SESSION_SECRET="$(head -c 32 /dev/urandom | base64 | tr -d '=+/' | head -c 43)"
    cat > "$CONF_DIR/config.yaml" <<EOF
http:
  listen: 127.0.0.1:8080
  webDist: $WEB_DIR
database:
  driver: sqlite
  sqlitePath: $STATE_DIR/panel.db
agent:
  socketPath: $RUN_DIR/agent.sock
  token: $TOKEN
sessionSecret: $SESSION_SECRET
log:
  format: text
EOF
    chmod 640 "$CONF_DIR/config.yaml"
    chown root:panel "$CONF_DIR/config.yaml"
else
    echo "[install] keeping existing config"
fi

echo "[install] systemd units"
for unit in panel-api.service panel-agent.service; do
    if [ -f "/etc/systemd/system/$unit" ]; then
        systemctl stop "$unit" 2>/dev/null || true
    fi
done
install -m 0644 "$SRC/systemd/panel-api.service"  /etc/systemd/system/
install -m 0644 "$SRC/systemd/panel-agent.service" /etc/systemd/system/

echo "[install] enabling services"
systemctl daemon-reload
systemctl enable panel-agent.service panel-api.service
systemctl restart panel-agent.service
systemctl restart panel-api.service

echo "[install] status"
systemctl --no-pager --lines=0 status panel-agent.service || true
systemctl --no-pager --lines=0 status panel-api.service || true
echo "[install] done"
