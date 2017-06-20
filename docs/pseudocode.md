# Substrate pseudocode
This document contains psuedocode descriptions of the high level CLI actions in Substrate. It's meant to help describe in detail what happens when you run a particular command, without getting too much into the weeds.

### `substrate zone create`
 1. Substrate makes sure it will be able to write the zone manifest file.

 2. Substrate creates a new zone manifest structure containing all the configuration metadata about the zone.

 3. Substrate calculates the zone subdomain based on the zone index and the environment name. For example, zone 0 in the `foo.bar.example.com` environment would have a zone subdomain of `zone00.foo.bar.example.com`.

 4. Substrate gets the Route53 Reusable Delegation Set (DS) that is used for all zones in the current AWS account. If no such DS exists, Substrate creates one. This gives a set of expected nameservers that will be used to serve the zone subdomain.

 5. Substrate looks for the first suffix of the zone subdomain that already has working public DNS records. For example, it looks up `foo.bar.example.com`, then `bar.example.com`, then `example.com`. It stops when it finds a suffix that has valid `NS` records based on a normal recursive DNS lookup.

 6. Substrate does a DNS lookup for `NS` record `zone00.foo.bar.example.com` against the authoritative nameservers for the working suffix domain. This gives the set of actual nameservers that are configured to host the zone subdomain (often an empty set, if it is not preconfigured).

 7. If the actual nameservers match the expected nameservers of the DS in step 3, then zone DNS is configured and ready to go. If not, Substrate tries to help the user create the correct `NS` records to set up the zone subdomain:
   - Substrate searches the Route53 Hosted Zones in the current account to see if the parent (suffix) domain is already hosted in the same account. If it is, Substrate can create the NS record that delegates the subdomain automatically.
   - If no matching Hosted Zone is found, then the environment domain must be hosted somewhere else. The user is prompted to manually create the correct `NS` records that will delegate the zone subdomain to the DS for the current account. After doing this manual step, the user can re-run `substrate zone create`.

 8. Substrate extracts a complete working Terraform environment to a temporary working directory. This includes pinned versions of all Terraform binaries as well as the Terraform configuration (`.tf`) to create the zone.

 9. Substrate creates a `.tfvars` file that translates most of the zone metadata into Terraform variables that we can reference from our `.tf` configuration files.

 10. Substrate runs `terraform get -update` to install the modules we use in our configuration.

 11. Substrate runs `terraform plan`, which causes Terraform to load our configuration, generate the dependency graph of all of our resources, an formulate a plan for the order of operations, which it prints to the terminal and writes to a `.tfplan` file. At this point Substrate pauses to prompt the user unless `--no-prompt` was passed.

 12. Substrate starts a background thread to watch streaming log events that will be generated during the base AMI bake. These events will be printed to the console until the zone has finished creating.

 13. Substrate runs `terraform apply`, passing in the `.tfplan` file from step 11. Terraform creates all the AWS resources related to the zone and outputs a `.tfstate` file. One of these steps is launching the AMI bakery instance, after which it launches the border, director, and worker nodes.

 14. Substrate collects the `.tfstate` file, bundles it it with the rest of the zone metadata, and writes this structure (as JSON) to the output zone manifest.

#### `substrate zone create` (AMI Bakery Node Boot)
WIP

#### `substrate zone create` (Director Node Boot)
WIP

#### `substrate zone create` (Border Node Boot)
WIP

#### `substrate zone create` (Worker Node Boot)
WIP

### `substrate zone destroy`
WIP
