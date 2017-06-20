#!/bin/bash
set -e

# shellcheck disable=SC1091
source ./common.sh

status "install jcl"
./jcl-bootstrap.sh 2>&1 |
  tee -a /var/log/substrate-base-ami-provision.log 2>&1 |
  systemd-cat -t substrate-base-ami-provision -p debug

status "RUN provision"
./provision.sh 2>&1 |
  tee -a /var/log/substrate-base-ami-provision.log 2>&1 |
  systemd-cat -t substrate-base-ami-provision -p debug
