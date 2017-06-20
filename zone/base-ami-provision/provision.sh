#!/bin/bash
set -ex

###############################################################################
# source the "zone.env" to get context (zone name, substrate version, ...)
###############################################################################

# shellcheck disable=SC1091
source /etc/substrate/zone.env

# shellcheck disable=SC1091
source ./common.sh

########################
# add upstream k8s repo
#######################
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF

###############################################################################
# update any outdated system packages (the Ubuntu AMI is often outdated)
###############################################################################
status "installing the latest updates to existing packages..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get \
  -o Dpkg::Options::="--force-confnew" \
  --force-yes \
  -fuy \
  dist-upgrade

###############################################################################
# install some extra packages we need:
# - linux-image-extra-virtual (enables aufs to use as a docker storage backend)
# - jq (allows us to dissect EC2 user data from a shell script)
# - systemd-journal-remote (log shipping tools)
###############################################################################
status "installing new packages..."
apt-get install -qy \
  linux-image-extra-virtual \
  jq \
  awscli \
  systemd-journal-remote

###############################################################################

# set some linux kernel boot flags to make Docker happier

###############################################################################

status "setting boot flags in GRUB to enable memory/swap accounting..."
echo "GRUB_CMDLINE_LINUX=\"cgroup_enable=memory swapaccount=1\"" >/etc/default/grub.d/00-substrate-docker-tweaks.cfg
update-grub

###############################################################################
# stop docker from starting immediately when we install (which is broken)
###############################################################################
# TODO: figure out why this is broken and drop this
status "stopping Docker from autostarting (hack)..."
echo "exit 101" >/usr/sbin/policy-rc.d
chmod +x /usr/sbin/policy-rc.d

status "installing Docker & kube pkgs..."
apt-get install -qy docker-engine kubelet kubeadm kubectl kubernetes-cni

status "$(dpkg -l | grep 'kube\|docker')"

systemctl start docker

status "removing autostart hack..."
rm /usr/sbin/policy-rc.d

###############################################################################
# clear the apt cache since we won't need it anymore
###############################################################################

status "clearing the apt cache..."
apt-get clean

###############################################################################
# install the Prometheus node_exporter binary (it doesn't run nicely in Docker)
###############################################################################
PROMETHEUS_NODE_EXPORTER_VERSION=0.13.0
status "installing Prometheus node_exporter v${PROMETHEUS_NODE_EXPORTER_VERSION}..."
curl -sL \
  "https://github.com/prometheus/node_exporter/releases/download/v${PROMETHEUS_NODE_EXPORTER_VERSION}/node_exporter-${PROMETHEUS_NODE_EXPORTER_VERSION}.linux-amd64.tar.gz" |
  tar -xzO "node_exporter-${PROMETHEUS_NODE_EXPORTER_VERSION}.linux-amd64/node_exporter" \
    >/usr/local/bin/prometheus-node-exporter
chmod +x /usr/local/bin/prometheus-node-exporter

###############################################################################
# disable all the dynamic MOTD stuff because it's slow on instances that don't
# have outbound connectivity
###############################################################################

status "setting up MOTD..."
chmod -x /etc/update-motd.d/*
# set our own static MOTD instead
printf "\nSubstrate AMI %s baked %s\n\n" "$SUBSTRATE_VERSION" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >/etc/motd

###############################################################################
# drop the Ubuntu legal banner (we don't need a reminder on every login)
###############################################################################

status "dropping the Ubuntu legal banner..."
rm /etc/legal

###############################################################################
# customize bashrc so we can get a prompt with the full hostname
###############################################################################

status "customizing .bashrc to get a nice prompt..."
sed -i "s/\\\\h/\\\\H/g" /etc/skel/.bashrc
sed -i "s/\\\\h/\\\\H/g" /home/ubuntu/.bashrc

###############################################################################
# drop the annoying "To run a command as administrator [...]" prompt
###############################################################################
status "creating .sudo_as_admin_successful files to get rid of sudo prompt..."
touch /home/ubuntu/.sudo_as_admin_successful
touch /etc/skel/.sudo_as_admin_successful

###############################################################################
# load all our computed environment into interactive shells as well
###############################################################################

status "customizing .bashrc to get our zone.env, node.env and interactive.env loaded..."
for bashrc in /etc/skel/.bashrc /home/ubuntu/.bashrc; do
  {
    echo "export \$(xargs < /etc/substrate/zone.env)"
    echo "export \$(xargs < /etc/substrate/node.env)"
    echo "export \$(xargs < /etc/substrate/interactive.env)"
  } >>"$bashrc"
done

###############################################################################
# build Docker images for all the base zone components
###############################################################################
builder_pids=()
for subdir in ./prebaked-images/*; do
  img=$(basename "$subdir")
  status "building substrate/$img base container..."
  (docker build -t "substrate/$img" "$subdir") &
  builder_pids+=("$! ")
done
wait "${builder_pids[@]}"

###############################################################################
# stash all our manifests so we can load them up via systemd later
###############################################################################
status "stashing static Kubernetes manifests templates into /etc/substrate/manifests..."
mkdir -p /etc/substrate/manifests
cp -rvT ./manifests /etc/substrate/manifests

###############################################################################
# pull some other images that we need cached locally
###############################################################################
status "CACHE images for kubeadm, calico, etc"
time ./pull-images.sh

status "STOP docker service"
systemctl stop docker

###############################################################################
# set up some helpers that we use in our custom units
###############################################################################
status "installing helper scripts to /usr/local/bin/..."
for script in ./sysd/*.sh; do
  chmod +x "$script"
  cp -v "$script" /usr/local/bin/
done

###############################################################################
# create some custom systemd units to start all our components
###############################################################################
status "installing our custom systemd units..."
cp -v ./sysd/*.service ./sysd/*.timer /etc/systemd/system/
systemctl daemon-reload
for filename in ./sysd/*.service ./sysd/*.timer; do
  systemctl enable "$(basename "$filename")" || true
done

###############################################################################
# set up some directories for systemd-journal-remote
###############################################################################

status "setting up journal directories for journald-remote shipping..."
mkdir -p /var/log/journal/remote/
chmod 2755 /var/log/journal/remote
chown systemd-journal-remote:systemd-journal /var/log/journal/remote

###############################################################################
# reconfigure journald so it doesn't clog /var/log/syslog and /var/log/kern.log
###############################################################################

status "reconfiguring journald to not mirror to syslog..."
mkdir -p /etc/systemd/journald.conf.d/
cat <<EOF >/etc/systemd/journald.conf.d/00-substrate-tweaks.conf
[Journal]
# keep at most 500 MB of logs on disk locally
SystemMaxUse=500M

# do not forward logs to traditional syslog
ForwardToSyslog=no
EOF

###############################################################################
# install calicoctl
###############################################################################

./install-calicoctl.sh

###############################################################################

# add a pre-login banner for compliance purposes
###############################################################################

status "setting login banner to a scary warning..."
cp banner.txt /etc/issue.net
echo "Banner /etc/issue.net" >>/etc/ssh/sshd_config

status "DONE: $(date)"
