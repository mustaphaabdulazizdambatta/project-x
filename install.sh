#!/bin/bash
set -e

SERVICE_FILE=/etc/systemd/system/x-tymus.service
PROJECT_DIR=/root/project-x

echo "[*] Stopping any running x-tymus instances..."
systemctl stop x-tymus 2>/dev/null || true
pkill -x x-tymus 2>/dev/null || true
sleep 2

echo "[*] Writing service file to $SERVICE_FILE..."
cat > "$SERVICE_FILE" << 'EOF'
[Unit]
Description=x-tymus phishing framework
After=network.target
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
User=root
WorkingDirectory=/root/project-x
ExecStart=/root/project-x/build/x-tymus -p /root/project-x/phishlets -c /root/.x-tymus
Restart=always
RestartSec=5s
KillMode=control-group
KillSignal=SIGTERM
TimeoutStopSec=15
LimitNOFILE=65536
StandardOutput=append:/var/log/x-tymus.log
StandardError=append:/var/log/x-tymus.log

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
