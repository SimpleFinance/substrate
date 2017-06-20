#!/bin/bash

cat >/etc/systemd/journal-upload.conf <<EOF
[Upload]
URL=http://$BORDER:19532
# TODO: handle certs
# ServerKeyFile=/etc/ssl/private/journal-upload.pem
# ServerCertificateFile=/etc/ssl/certs/journal-upload.pem
# TrustedCertificateFile=/etc/ssl/ca/trusted.pem
EOF

systemctl enable systemd-journal-upload.service
systemctl restart systemd-journal-upload.service
