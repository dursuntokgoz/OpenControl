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

echo "=== nginx restart via whitelist (fixture for phase 2 acceptance) ==="
# The agent currently exposes only read-only ops; restarting services through
# the agent arrives with the ServiceControl op set. Here we assert that nginx
# itself works so phase 2 has a live service to manage.
systemctl restart nginx.service
nginx -v
curl -fsS http://127.0.0.1/ | grep -q "nginx" || {
    echo "nginx default page missing"
    exit 1
}

echo "ITEST_OK"
