#!/usr/bin/env python
import boto3
import time
import os
import logging
import re

logging.basicConfig()
LOG = logging.getLogger("border-dns-watch-ec2")


def build_hosts_list(ec2, substrate_zone):
    instances = ec2.instances.filter(Filters=[
        {'Name': 'instance-state-name', 'Values': ('running', )},
        {'Name': 'tag:substrate:zone', 'Values': (substrate_zone, )}
    ])

    for instance in sorted(instances, key=lambda i: i.id):
        # skip if the instance doesn't have an IP assigned for some reason
        if instance.private_ip_address is None:
            continue

        # pull out the substrate:role tag or skip if there isn't one
        role = None
        for tag in instance.tags:
            if tag["Key"] == "substrate:role":
                role = tag["Value"]
                break
        if role is None:
            continue

        # allow a lookup by full instance ID + role
        # (like "i-0ef8d0a59f62d5fb1.worker-lb-corp.zone.local")
        yield "%s\t%s.%s.zone.local" % (
            instance.private_ip_address, instance.id, role)

        # allow a lookup by instance ID alone
        # (like "i-0ef8d0a59f62d5fb1.zone.local")
        yield "%s\t%s.zone.local" % (
            instance.private_ip_address, instance.id)

        # allow a lookup by role alone (like "worker-lb-corp.zone.local")
        # these will end up as round robin entries in dnsmasq
        yield "%s\t%s.zone.local" % (
            instance.private_ip_address, role)

        # if the role has a number at the end (like "director-0"), allow lookup
        # without the number. This will make, e.g., "director.zone.local" a
        # round robin for all the director instances.
        match = re.match(r"(.*)-\d+", role)
        if match is not None:
            yield "%s\t%s.zone.local" % (
                instance.private_ip_address, match.group(1))


def main():
    try:
        substrate_zone = os.environ["SUBSTRATE_ZONE"]
    except KeyError:
        raise RuntimeError("expected $SUBSTRATE_ZONE in environment!")

    try:
        ec2 = boto3.resource('ec2')
    except Exception as e:
        raise RuntimeError("couldn't connect to EC2: %r" % (e, ))

    while True:
        last = None
        try:
            # grab the expected hosts file lines and flatten them
            hosts = list(build_hosts_list(ec2, substrate_zone))
            hostfile_contents = "\n".join(hosts) + "\n"

            # if nothing has changed since last time, assume we're in a stable
            # period and sleep for a longer duration
            if last == hostfile_contents:
                time.sleep(60)
                continue

            # otherwise write out the updated file and only pause for a moment,
            # since if we just made a change we may have more changes soon
            last = hostfile_contents
            LOG.info("rendering %s hosts entries...", len(hosts))
            with open("/etc/dnsmasq/extra-hosts/ec2", "wb") as hostfile:
                hostfile.write(hostfile_contents)
            time.sleep(10)

        except Exception as e:
            LOG.exception("hit error, backing off...")
            time.sleep(120)

if __name__ == "__main__":
    main()
