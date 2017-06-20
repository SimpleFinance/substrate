#!/bin/bash
# Launches initial Substrate node components.
#
# See substrate-node.service for more context.
substrate-kubelet-configure.sh

for _ in {1..50}; do
  kubeadm join --token="${K8STOKEN}" "${DIRECTOR}" && break
  sleep 15
done
