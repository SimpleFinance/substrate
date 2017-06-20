# Mechanics

## Custom Terraform Providers
We wrote a few custom providers to help build the base AMI. These are just (small) Go programs that implement the [Terraform provider interface](https://www.terraform.io/docs/plugins/provider.html). See [this blog post](https://hashicorp.com/blog/terraform-custom-providers.html) for a helpful walkthrough of the interface.

#### Archive (`terraform-provider-archive`)
This is a custom provider that builds a compressed tar from files in a directory (`.tgz`). See [the README](../terraform-provider-archive/README.md) for more info. We use this to bundle up some scripts and configuration when we bake our base AMI, as well as for bundling up spinup scripts.

#### Bakery (`terraform-provider-bakery`)
This is a custom provider that does most of the work to bake the base AMI:
 1. Launches an EC2 instance using some source AMI (Ubuntu 15.10).
 2. Runs user data, which configures the instance by installing packages, etc... This script finishes by shutting down the instance.
 3. When the instance has shut down, takes an EBS snapshot which can be passed to the `aws_ami` resource to create a custom AMI.

[The README](../terraform-provider-bakery/README.md) has more info.

## Systemd Units
The on-instance configuration we create on the base AMI includes a number of systemd init scripts ("units"). These allow the instances in the cluster to boot into a working configuration automatically at first launch (or on reboot/recovery).

#### Docker Runtime (`docker.service`)
This starts Docker on every node (worker and director), setting all the custom parameters we want for Substrate's use case.

#### Substrate Node Environment (`substrate-node-env.service`)
This runs relatively early in the boot process. It's job is to read EC2 metadata and user data passed in from Terraform and create a file `/etc/substrate/node.env`, which contains all the node-specific configuration variables we'll need in other units.

#### Substrate Hostname (`substrate-hostname.service`)
This sets the system hostname to our preferred FQDN (see _System hostnames_ in [`design.md`](design.md)).

#### Calico Per-Node Daemon (`calico-node.service`)
This launches the Project Calico [per-node daemon](https://github.com/projectcalico/calico-docker), which joins the node into the zone's Calico mesh.

#### Kubernetes Per-Node Daemon (`substrate-kubelet.service`)
This launches [`kubelet`](http://kubernetes.io/docs/admin/kubelet/), the Kubernetes per-node daemon, which joins the node into the Kubernetes cluster as well as runs special "static pods" to bootstrap the Kubernetes API itself.

#### Substrate Director (`substrate-director.service`)
This runs very last, and only on director nodes (using systemd's `ConditionHost` feature). It launches the Project Calico etcd store and policy agent daemon, the Kubernetes etcd store, API server, and controller manager. After the Kubernetes API is up and running, it launches a core set of "initial" resources into the cluster, including [kube-proxy](http://kubernetes.io/docs/admin/kube-proxy/) (running in a [DaemonSet](http://kubernetes.io/docs/admin/daemons/) on every node), and the Kubernetes web dashboard [kube-ui](https://github.com/kubernetes/kube-ui).

## DNS
There are several types of DNS names related to Substrate zones and environments:

#### Zone-local Resolution of Kubernetes DNS / SkyDNS (`*.cluster.local`)
This level is served by a Kubernetes addon called [SkyDNS](http://kubernetes.io/docs/admin/dns/). These are names for objects within Kubernetes, such as service virtual IPs. This is the primary level of DNS useful from within applications running on Substrate.

#### Zone-local Resolution of Instance DNS (`*.zone.local`)
This level is served by a DNS server running on the Border instances (in authoritative mode). These are internal names for the EC2 instances hosting the Substrate zone. Each instance is resolvable in several ways:
 - Instance ID and role (e.g., `i-0123456789abcdef0.border-0.zone.local`). This is the canonical hostname described in [`design.md`](design.md).
 - Instance ID alone (e.g., `i-0123456789abcdef0.zone.local`).
 - Role alone (e.g., `border-0.zone.local`). When multiple instances are serving the same role, this will be a round robin record resolving to all of them.
 - Role without numbers (e.g., `border.zone.local`). This is a special for the `director` and `border` nodes which have a numeric suffix like `border-0`. They are also resolvable without that suffix in a round robin fashion.

#### Zone-local Resolution of External DNS (`*`)
Code running in a zone has the ability to resolve external domain names subject to some security filtering. This level is also served by the DNS server running on the Border instances (in recursive/forwarding cache mode). These are names for external services that we need to resolve within the cluster, such as `ec2.us-west-2.amazonaws.com`.

#### External Resolution of Zone Components (`*.zoneXX.environment-domain.com`)
Internet-connected clients should be able to resolve the names of public facing zone instances, such as the border and load balancer instances. These names are served from a Route53 Hosted Zone that is created as part of the Substrate zone. It is authoritative for a subdomain `zoneXX` of the environment domain. For example, zone 0 of the `example.com` environment would have a Route53 Hosted Zone authoritative for `zone00.example.com`. This Hosted Zone would have `A` records such as `border.zone00.example.com`.

It's important that this subdomain is resolves correctly _before_ the zone boots for the first time. This allows the zone components to use `TXT` records to create valid SSL/TLS certificates. For this same reason, it's important that this level of DNS is delegated to a Route53 Hosted Zone that the zone can safely have full control over without risking giving a single zone control over the top level environment DNS records.

We ensure that this resolution is set up before the zone boots in `substrate zone create`. If DNS is not set up correctly, `substrate zone create` will attempt to set it up or prompt the user with instructions for configuring the environment DNS to delegate the subdomain (see `pseudocode.md` for details).

#### External Resolution of Environment Domains (`*.environment-domain.com`)
The top level DNS hosting for an environment may or may not be managed by Substrate. To work with Substrate, it needs to serve two types of records:

 1. `NS` records to delegate each `zoneXX.environment-domain.com` subdomain to the appropriate per-zone nameservers.

 2. `CNAME` records to direct user-facing domains to the load balancers in the currently active zone. This should ideally be a wildcard record (non-standard but supported by Route53). This is the mechanism we use to promote a zone to active duty, by resolving queries for, e.g., `status.example.com` to point at the proper load balancer pool in the active zone.

Since `NS` records are aggressively cached (often longer than their TTLs), we prefer to pre-delegate zone subdomains to Route53 Reusable Delegation sets that are pre-configured in the AWS accounts where we would like to host zones. This means creating `NS` records ahead of time according to some per-environment scheme. For example, in our `example.com` environment, we might pre-delegate odd-numbered zones to a "Prod A" AWS account and even-numbered zones to a "Prod B" account. This concern does not apply when the root domain DNS is hosted in the same account as the zones, as should often be the case for development environments. In this case the environment's root Hosted Zone and the zone's Hosted Zone can both use the same Delegation Set, eliminating any TTL problems.

To simplify setup, Substrate assumes that all zones in a particular AWS account will share the same Route53 Reusable Delegation Set, which will be created automatically at first use and tagged with a `substrate-*` prefix. This Delegation Set is not cleaned up on zone destruction but Delegation Sets are free in AWS.
