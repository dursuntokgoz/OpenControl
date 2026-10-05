#!/bin/sh
# Runs inside the integration container after systemd is ready.
set -eu

echo "=== service states ==="
systemctl is-active --no-legend panel-api.service
systemctl is-active --no-legend panel-agent.service

echo "=== healthz ==="
curl -fsS http://127.0.0.1:8080/healthz
echo

echo "=== agent ping ==="
SERVERPANEL_CONFIG=/etc/serverpanel/config.yaml panelctl agent-ping

echo "=== doctor ==="
panelctl doctor -config /etc/serverpanel/config.yaml

echo "=== service list via agent ==="
SERVERPANEL_CONFIG=/etc/serverpanel/config.yaml panelctl service list

echo "=== nginx restart via agent (phase 2 acceptance) ==="
SERVERPANEL_CONFIG=/etc/serverpanel/config.yaml panelctl service restart nginx.service
sleep 1
systemctl is-active --no-legend nginx.service
curl -fsS http://127.0.0.1/ | grep -q "nginx" || {
    echo "nginx default page missing after agent restart"
    exit 1
}

echo "ITEST_OK"
