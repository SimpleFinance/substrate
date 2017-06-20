#!/bin/bash
set -ex

###############################################################################
# install journald-cloudwatch-logs, a log shipper for journald
###############################################################################

# shellcheck disable=SC1091
source ./common.sh

# shellcheck disable=SC1091
source /etc/substrate/zone.env

JCL_VERSION="v0.0.6a"
JCL_SHA256="bdfc78839c2c5ddc101973c07c2efbba6a0c83097e0aa631531ad063db5b914a"

status "installing journald-cloudwatch-logs ${JCL_VERSION}..."
mkdir -p /usr/local/bin
curl -o /usr/local/bin/jcl -L \
  https://github.com/SubstrateProject/journald-cloudwatch-logs/releases/download/${JCL_VERSION}/journald-cloudwatch-logs
bash -c "echo '${JCL_SHA256}  /usr/local/bin/jcl' | sha256sum --check"
chmod +rx /usr/local/bin/jcl
mkdir -p /var/log/journal/

cat >/etc/substrate/jcl.conf <<EOF
# ship logs to cloudwatch
log_group = "${SUBSTRATE_ZONE_CLOUDWATCH_LOGS_GROUP}"
journal_dir = "/var/log/journal/remote/"
state_file = "/var/log/journal/remote/jcl-state"
log_priority = "debug"
EOF

cat >/etc/substrate/jcl-local.conf <<EOF
# ship logs to cloudwatch
log_group = "${SUBSTRATE_ZONE_CLOUDWATCH_LOGS_GROUP}"
state_file = "/var/log/journal/jcl-state"
log_priority = "debug"
EOF

###############################################################################
# start up a temporary log shipping process to ship the AMI build logs
###############################################################################
nohup /usr/local/bin/jcl /etc/substrate/jcl-local.conf >/dev/null 2>/dev/null </dev/null &
