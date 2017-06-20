# Substrate Changes

## v1.x (unreleased)

## v1.0.1

- Stop using our custom `terraform-provider-ubuntu` plugin, since Terraform can now provide this functionality natively.
- Expose `node_exporter` http on director.
- Bump up to Terraform 0.8.3
- More version information from provisioning script and cli utils
- Adapt to use `kubeadm` 1.6alpha line
- Bump up to version 1.5.1 of Kubernetes

## v1.0.0

- Increase the number of worker instances from 2 to 4.
- Fix a bug where the border Squid proxy wouldn't stay running after zone spinup.
- Install and configure Prometheus `node_exporter` on each of our instances.
- Add `quay.io` to our list of whitelisted container sources.

## v1.0.0-rc3 (2016-12-01)

- Fixed a bug where `journal-remote` logs collected on the border node were not being cleaned automatically. This led to the border node running out of disk over long periods of time. This is a bug in systemd/`journal-remote`, which we've worked around by cleaning up manually with a timer unit (aka cronjob).

## v1.0.0-rc2 (2016-11-29)

- Disabled log forwarding from `journald` (which is good about not using up the entire disk) to `syslog` (which is not). This avoids an issue where logs could fill the root volume and prevent Kubernetes from functioning properly on a node.
- Lowered the logging verbosity of `kubelet` to reduce pressure on our logging system.

## v1.0.0-rc1 (2016-11-28)

- Our first v1.0.0 release candidate! We've made a ton of progress since our v0.0.x versions -- too much to reasonably list here. Suffice to say we built most of Substrate in the last year.

## v0.0.6

- Added a real CLI and a reworked build process.

## v0.0.5

- Begin work on new development flow, reintroduce `master` branch

## v0.0.4

- Remove proxy/squid setup
- Move kubernetes master to a single instance etcd setup
- Create a kubernetes worker ASG and launch configuration
- Remove Route53 setup for now

## v0.0.3
 - Upgrade to Terraform 0.6.14.
 - Upgrade to Kubernetes 1.2

## v0.0.2
 - Made a test change to a naming thing to see if it gets picked up.

## v0.0.1
 - Initial version of substrate with versioning/CI set up.
