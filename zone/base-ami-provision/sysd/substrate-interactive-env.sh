#!/bin/sh
# Generates /etc/substrate/interactive.env, used to setup env vars for interactive
# services
#
# This is pulled into the bashrc during provision

# Gets the auth token for this calico tool
# https://github.com/projectcalico/k8s-policy/blob/v0.1.3/policy_tool/README.md
TOKEN_NAME=$(kubectl describe serviceaccount/default | grep "Tokens:" | awk '{print $2}')
K8S_TOKEN=$(kubectl get secret "$TOKEN_NAME" -o yaml | grep "token: " | awk '{print $2}')

# dump everything out to the target .env file
cat >/etc/substrate/interactive.env <<EOF
KUBE_AUTH_TOKEN=$K8S_TOKEN
EOF
