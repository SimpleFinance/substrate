#!/bin/bash
#
# Set up envvars for border http proxy for docker
#

mkdir -p /etc/systemd/system/docker.service.d/

cat >/etc/systemd/system/docker.service.d/http-proxy.conf <<EOF
[Service]
Environment="HTTP_PROXY=http://${BORDER}:3128"
EOF

## Reload daemon
systemctl daemon-reload
