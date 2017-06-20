#!/bin/bash
# Launches initial Substrate director components.
#
# See substrate-director.service for more context.
substrate-kubelet-configure.sh

# @@ make idempotent
# @@ parameterize version pin
kubeadm init --use-kubernetes-version="v1.5.1" --token="${K8STOKEN}" #--cloud-provider=aws

for template in /etc/substrate/manifests/director-static-pods/*.yaml; do
  rendered="/etc/kubernetes/manifests/$(basename "${template}")"
  echo "loading static pod $rendered from $template..."
  envsubst <"${template}" >"${rendered}"
done

template=/etc/substrate/manifests/networking/calico.yaml
rendered=/etc/substrate/manifests/calico.yaml
envsubst <"${template}" >"${rendered}"
kubectl apply -f "${rendered}"
