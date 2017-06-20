#!/bin/bash
###############################################################################
# set up a little helper to log higher-level events at INFO level
###############################################################################
status() {
  echo "$1" 2>&1 |
    tee -a /var/log/substrate-base-ami-provision.log 2>&1 |
    systemd-cat -t substrate-base-ami-provision -p info
}

dl-install() {
  set -e
  outpath="/usr/local/bin/$1"
  status "Installing '$1' to $outpath"
  mkdir -p /usr/local/bin
  curl -o "$outpath" -L "$2"
  bash -c "echo '${3}  $outpath' | sha256sum --check"
  chmod +rx "$outpath"
  return 0
}
