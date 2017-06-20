#!/bin/bash
# this needs to be run before kubeadm

mkdir -p /etc/systemd/system/kubelet.service.d
cat >/etc/systemd/system/kubelet.service.d/11-node-labels.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=--v=2 --node-labels=\"substrate.zone/role=${ROLE},substrate.zone/version=${SUBSTRATE_VERSION},substrate.zone/zone=${SUBSTRATE_ZONE},substrate.zone/environment=${SUBSTRATE_ENVIRONMENT}\""
EOF

systemctl daemon-reload
