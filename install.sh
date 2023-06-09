#!/bin/bash

#install to path
firstdir=$(echo $PATH | cut -d':' -f1)

# check if filehaunt binary exists
if [ ! -f filehaunt ]; then
    echo "[-] filehaunt binary not found"
    go build
    if [ $? -ne 0 ]; then
        echo "[-] failed to build filehaunt binary"
        exit 1
    else
        echo "[+] filehaunt binary built"
        cp filehaunt $firstdir
        mv filehaunt /opt/filehaunt
    fi
else 
    echo "[+] filehaunt binary found"
    cp filehaunt $firstdir
    mv filehaunt /opt/filehaunt
fi

printf "[Unit]
Description=Service for filehaunt file verification
ConditionPathExists=/opt/filehaunt
After=network.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/opt/filehaunt/filehaunt -verify
Restart=always
RestartSec=60

[Install]
WantedBy=multi-user.target" > /etc/systemd/system/filehaunt.service

systemctl start filehaunt.service
systemctl enable filehaunt.service