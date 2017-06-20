#!/bin/bash

set -e

# shellcheck disable=SC1091
source ./common.sh

#VERSION="v0.22.0"
SHA="f843d0fcecfa61e203c668371138f9f16533df7942f19343ad8e4eeb47504a8f"
#URL="https://github.com/projectcalico/calico-containers/releases/download/${VERSION}/calicoctl"
URL="https://github.com/n-marton/calico-containers/releases/download/v0.22.0k8s/calicoctl"
FINAL_NAME="calicoctl"

dl-install $FINAL_NAME $URL $SHA
