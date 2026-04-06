#!/bin/bash
set -e

SERVICE_FILE=/etc/systemd/system/x-tymus.service
PROJECT_DIR=/root/project-x

echo "[*] Stopping any running x-tymus instances..."
pkill -x x-tymus 2>/dev/null || true
sleep 2

echo "[*] Writing service file to $SERVICE_FILE..."
cat > "$SERVICE_FILE" << 'EOF'
[Unit]
Description=x-tymus phishing framework
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/root/project-x
ExecStartPre=/bin/sh -c 'pkill -x x-tymus || true'
ExecStartPre=/bin/sleep 2
ExecStart=/root/project-x/build/x-tymus -p /root/project-x/phishlets -c /root/.x-tymus
KillMode=control-group
KillSignal=SIGTERM
TimeoutStopSec=15
Restart=on-failure
RestartSec=10s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

echo "[*] Reloading systemd..."
systemctl daemon-reload

echo "[*] Enabling and starting x-tymus service..."
systemctl enable x-tymus
systemctl restart x-tymus

sleep 2
echo ""
systemctl status x-tymus --no-pager
