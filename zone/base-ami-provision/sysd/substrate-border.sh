#!/bin/bash
# Launches initial Substrate border components.
#
# See substrate-border.service for more context.

###############################################################################
# first load a bunch of static pods directly into kubelet, which will get etcd
# and the Kubernetes API server up and running
###############################################################################
mkdir -p /etc/substrate/rendered-manifests/
for template in /etc/substrate/manifests/border-static-pods/*.yaml; do
  rendered="/etc/substrate/rendered-manifests/$(basename "${template}")"
  echo "loading static pod $rendered from $template..."
  envsubst <"${template}" >"${rendered}"
done

# bootstrap our proxy
kubelet --runonce --pod-manifest-path=/etc/substrate/rendered-manifests/border-squid.yaml
k8sdir=/etc/kubernetes/manifests
mkdir -p ${k8sdir} &&
  cp /etc/substrate/rendered-manifests/border-dns.yaml ${k8sdir}/border-dns.yaml

substrate-kubelet-configure.sh
for _ in {1..50}; do
  kubeadm join --skip-preflight-checks --token="${K8STOKEN}" "${DIRECTOR}" && break
  sleep 15
done

cp /etc/substrate/rendered-manifests/border-squid.yaml ${k8sdir}/border-squid.yaml
