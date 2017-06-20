#!/bin/sh
# Generates /etc/substrate/node.env.
#
# See substrate-node-env.service for more context.

# grab some EC2 metadata
INTERNAL_IP=$(ec2metadata --local-ipv4)
PUBLIC_IP=$(ec2metadata --public-ipv4)
INSTANCE_ID=$(ec2metadata --instance-id)

# pull out some variables from the EC2 user data we set in terraform
ROLE=$(ec2metadata --user-data | jq -r .substrate.role)
DIRECTOR=$(ec2metadata --user-data | jq -r .substrate.director)
BORDER=$(ec2metadata --user-data | jq -r .substrate.border)

# choose the fully qualified domain name for this node
FQDN="$INSTANCE_ID.$ROLE.zone.local"

# the Calico etcd instance is also on the director
ETCD_AUTHORITY="$DIRECTOR:$CALICO_ETCD_PORT"

# dump everything out to the target .env file
cat >/etc/substrate/node.env <<EOF
ROLE=$ROLE
DIRECTOR=$DIRECTOR
BORDER=$BORDER
FQDN=$FQDN
ETCD_AUTHORITY=$ETCD_AUTHORITY
INSTANCE_ID=$INSTANCE_ID
INTERNAL_IP=$INTERNAL_IP
DEFAULT_IPV4=$INTERNAL_IP
PUBLIC_IP=$PUBLIC_IP
BORDER=$BORDER
EOF
