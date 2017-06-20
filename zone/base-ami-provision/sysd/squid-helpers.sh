#!/bin/bash

wl=/etc/squid/whitelist.txt

REGISTRY_URLS="        \
.docker.io             \
.cloudfront.net        \
.gcr.io                \
storage.googleapis.com \
"

scid() {
  docker ps -q --filter='ancestor=substrate/squid'
}

squid_hup() {
  # shellcheck disable=SC2046
  docker exec -t "$(scid)" kill -1 "$(docker exec -t $(scid) pgrep squid | tr -d '\r')"
}

whitelist_url() {
  # Appends a url to the whitelist
  docker exec -t "$(scid)" /bin/sh -c "echo $1 >> $wl"
}

squid_show_whitelist() {
  docker exec -t "$(scid)" /bin/sh -c "cat $wl"
}

whitelist_registries() {
  for url in $REGISTRY_URLS; do
    whitelist_url "$url"
  done
  squid_hup
}

squid_reset() {
  docker kill "$(scid)"
}

squid_turn_on_access_logging() {
  docker exec "$(scid)" \
    sed -i "s/access_log none/\#access_log none/" /etc/squid/squid.conf
  squid_hup
}

squid_tail_access_log() {
  docker exec -it "$(scid)" tail -f /var/log/squid/access.log
}
