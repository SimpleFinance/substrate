# Substrate design
This setup is designed to allow new environments to be provisioned easily with an automated CLI tool (`substrate`). A regular (non-ops) engineer should be able to set up a dev environment for their team in under 30 minutes, using only a non-production AWS account (with no possible impact to production systems).

Likewise, an infrastructure engineer should be able to provision new production infrastructure in an arbitrary AWS region by running the same tool, then "plugging in" to the global CI/CD system once the new zone is running and validated.

Finally, the infrastructure should provide a manageable API we can use to build platform-level components such as our planned CI/CD dashboard.

## Start with commodity components
One major goal of this design is to use as many off the shelf components as possible. We'd also like to try to follow widely-used conventions where we can, since that tends to make new open source tools easier to fit into our environment.

Some of the major tools used by this repo are:
 - **_[Ubuntu](http://www.ubuntu.com/)_**: we don't need too much out of our instance base AMI, but Ubuntu should be on the ball shipping security updates more than smaller distros.

 - **_[Terraform](https://terraform.io)_**: a tool for provisioning infrastructure resources (instances, networks, security groups). Works very similarly to AWS CloudFormation, but also supports cross-provider templates (e.g., AWS + DynECT DNS).

 - **_[Docker](https://www.docker.com/)_**: a tool for working with Linux containers (process groups with isolation via kernel namespaces).

 - **_[Kubernetes (k8s)](http://kubernetes.io/)_**: a cluster scheduling engine for managing containers running across a group of instances. Provides an API we can use to deploy services, while managing service availability, providing service discovery, autoscaling, and other cluster management facilities. Kubernetes is managed by Google and is a spiritual successor to Google's internal [Borg](http://research.google.com/pubs/archive/43438.pdf) scheduler.

 - **_[Kubernetes Cluster Federation (aka Ubernetes)](https://github.com/kubernetes/kubernetes/blob/master/docs/proposals/federation.md)_**: a layer which connects multiple Kubernetes clusters running in different regions/accounts/providers. It provides an API similar to the local Kubernetes API, but with extra parameters to handle multi-region concerns.

## Naming things
These are the proposed names for the core concepts of Substrate:

 - **_Environment_**: a "universe" of services. Each environment maps to a primary domain name which is also the name we use for the environment. There is one special environment `example.com` which hosts our production services and data. Several environments might share a parent domain even though they are separate environments. For example we might have a `staging.qa.example.com` environment and `load-test.qa.example.com` that are each running QA-related workloads but each contain a complete, independent workload.

 - **_Account_**: a hosting provider account. In our current infrastructure, this is an AWS account. We use multiple AWS accounts to host various development, QA, and production workloads. Accounts have a many-to-many relationship with environments. A single account might host several dev/QA environments, and the single production `example.com` environment will be hosted across multiple accounts.

 - **_Zone_**: a unit of infrastructure serving part of a single environment. Each zone is provisioned under a single AWS account into a single AWS Availability Zone. Zones are mostly homogeneous, meaning they run relatively uniform workloads (e.g., there is not a separate zone for frontend vs. backend applications). Zones are numbered within each environment and are refered to by name using a subdomain `zoneXX` of the environment to which they belong, e.g., `zone01.example.com` is zone number 1 in the `example.com` environment.

 - **_Corp_**: a special pseudo-environment that hosts things that are not environment-specific. Systems in corp are treated as special cases and will mostly be managed similarly to our current infrastructure. Examples of corp systems are corporate identity providers, VPN servers, source code repositories, and CI/CD services. Corp also serves as a hub to help all the zones discover and route to each other (see [`networking.md`](networking.md)).

Within a Substrate zone, we attach metadata as AWS tags to all the components we create:

 - **`substrate:environment`**: the domain of the environment to which this resource belongs (e.g., `example.com`).

 - **`substrate:zone`**: the domain of the zone to which this resource belongs (e.g., `zone01.example.com`).

 - **`substrate:role`**: the logical role of this resource within the zone (e.g., `worker`, `director`).

 - **`substrate:version`**: the version of Substrate that created the resource (e.g., `v1.0.0`).

Wherever possible, we try to use multidimensional key/value style labels to identify instances (e.g., role, instance ID, region, verstion). However, some components of Substrate expect these identifying items to be flattened down to a simpler namespace:

 - **_System hostnames_**: instances running Substrate have hostnames of `$instanceID.$role.zone.local`. For example, a worker node might have hostname `i-01234567.worker.zone.local`. We don't intend to use this hostname as a primary identifier for the instance, but it should end up in system logs and SSH prompts.

 - **_AWS `Name` tags and object names_**: in addition to the tags under the `substrate:` prefix, we also set the tag `Name` to include the Substrate role and zone name in the format `substrate-$zoneName-$role`, converting `.` to `-` to form a valid AWS resource name. For example, a worker instance in zone 1 of the `example.com` environment would be tagged with `"Name": "substrate-zone01-example-com-worker"`. This tag is only set for convenience when using the AWS Management Console with the default column layout. We should not rely on this tag for any purpose other than manual access via the AWS web UI. It's different from the hostname because we can't easily include the instance ID (since the tags are specified before that is known). It also applies to resources other than EC2 instances, so a 1:1 mapping with hostname doesn't make total sense anyway. We also use this value for the name of AWS resources that have a name attribute but do not support tagging (e.g., IAM roles).

## Layers of immutability
Immutability makes systems easier to reason about. Substrate has three immutable layers. Each is immutable in the sense that we do not modify it in place. Instead, we change it by creating a new copy, shifting load, then destroying the original:

 1. _Containers_: our CI/CD system will produce container images (and metadata) as artifacts. To deploy a new version of an application, we'll end up spinning up the new version in fresh containers, then terminating the old version.

 2. _Instances_: instances will contain a full snapshot of the base workload. To update system packages, we'll bake a fresh AMI, then spin up fresh instances running the new AMI, migrate containers to the new hosts (via k8s API), then shut down old (now idle) instances.

 3. _Zones_: when we want to change some core structure of our zone (e.g., moving to new instance types, security group tweaks, changing AWS regions) we'll spin up a fresh zone using the new structure, plug it into the corp network, register it with the Ubernetes layer, then migrate applications from the old zone to the new zone, leaving the old zone empty and ready to destroy.

## Control plane vs. data plane
In many network devices, there is a concept of a [_control plane_ vs. a _data plane_](http://sdntutorials.com/difference-between-control-plane-and-data-plane/). The control plane runs higher level processes that do not have hard latency constraints (e.g., routing protocols). The data plane is the part of the system that forwards packets and must operate at high speeds with low latency.

In this architecture, Corp acts as the control plane. It is not involved at all in handling individual user requests, but configures and deploys to the zones, which act as the data plane. Since Corp is not directly involved in handling user requests, it has much lower availability requirements. Our goal should be that Corp is something we can bring down for maintenance for several hours without any impact to customers (but possibly with some impact to internal users).

## Isolation
We want to isolate different zones from each other as much as possible. This means development and production workloads will run in different AWS accounts. We also want the ability to shard a single production environment across two AWS accounts. This is meant to reduce the potential blast radius of any accidental or malicious use of a single engineer's AWS credentials.

There are some singleton resources in each environment that cannot be sharded across zones/accounts. Domain registration and DNS hosting for the environment domain can only exist in one place. For the production environment, domain registration and root DNS hosting for `example.com` will happen in a third AWS account dedicated purely to that purpose. Administrative access to this account should be rare.

Development/QA environments do not need as much isolation for security/availability purposes. We will support running multiple development environments under the same AWS account, and hosting development domain registration and DNS under that same account for convenience.

## Providing a clean upgrade path
Another reason to split the infrastructure into zones is to provide a clean blue/green-style upgrade path. We always want to be able start with an environment running Substrate version X, spin up new zones running Substrate version Y, migrate databases and request traffic to the new zones, and then shut down the version X zones. This should be possible even between major version upgrades to Substrate.

Our desire for a clean upgrade path also pushes us towards keeping as much of the Substrate implementation as possible at the zone level, with as few components at the environment level as possible. Any object/component that is global for an environment (e.g., DNS) must be compatible across all potential future Substrate versions.
