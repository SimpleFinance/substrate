#!/bin/bash
set -e

# shellcheck disable=SC1091
source /etc/substrate/zone.env

# shellcheck disable=SC1091
source ./common.sh

# shellcheck disable=SC2039

doPull() {
  puller_pids=()
  while read -r image; do
    status "pulling $image container to local cache..."
    docker pull "$image" &
    # shellcheck disable=SC2039
    puller_pids+=("$! ")
  done <"$1"
  # shellcheck disable=SC2039
  wait "${puller_pids[@]}"
}

doPull cached-images.txt
