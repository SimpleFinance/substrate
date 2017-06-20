#!/bin/bash

while ! curl "http://${DIRECTOR}:${CALICO_ETCD_PORT}/health" >/dev/null; do
  echo "Attempt to connect to calico etcd authority"
  sleep 1
done

/usr/local/bin/calicoctl pool add 192.168.0.0/16 --nat-outgoing
