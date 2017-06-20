# substrate node ami bake module

variable "userdata" {}

variable "substrate_environment" {}

variable "substrate_version" {}

variable "calico_etcd_port" {}

variable "kubernetes_api_port" {}

variable "cluster_dns_server" {}

variable "aws_region" {}

variable "aws_availability_zone" {}

variable "cluster_dns" {}

variable "zone_name" {}

variable "zone_prefix" {}

variable "ubuntu_ami" {}

variable "iam_profile_name" {}

# create a virtual network with the specified CIDR
resource "aws_vpc" "base_ami_builder" {
  cidr_block           = "10.0.0.0/28"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.zone_name}"
    "substrate:role"        = "ami-builder-vpc"
    "Name"                  = "${var.zone_prefix}-base-ami-builder-vpc"
  }
}

# create a security group for the builder, it needs to reach the internet
resource "aws_security_group" "base_ami_builder" {
  name_prefix = "${var.zone_prefix}-base-ami-builder-security-group-"
  vpc_id      = "${aws_vpc.base_ami_builder.id}"

  egress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"

    cidr_blocks = [
      "0.0.0.0/0",
    ]
  }

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.zone_name}"
    "substrate:role"        = "base-ami-builder-security-group"
    "Name"                  = "${var.zone_prefix}-base-ami-builder-security-group"
  }

  lifecycle {
    create_before_destroy = true
  }
}

output "builder_sg_id" {
  value = "${aws_security_group.base_ami_builder.id}"
}

output "builder_vpc_id" {
  value = "${aws_vpc.base_ami_builder.id}"
}

# create some DHCP settings for everything spun up in the VPC
resource "aws_vpc_dhcp_options" "base_ami_builder" {
  domain_name_servers = ["AmazonProvidedDNS"]

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.zone_name}"
    "substrate:role"        = "ami-builder-dhcp-options"
    "Name"                  = "${var.zone_prefix}-base-ami-builder-dhcp-options"
  }
}

# associate our DHCP settings with the VPC
resource "aws_vpc_dhcp_options_association" "base_ami_builder" {
  vpc_id          = "${aws_vpc.base_ami_builder.id}"
  dhcp_options_id = "${aws_vpc_dhcp_options.base_ami_builder.id}"
}

# set up an "internet gateway" in our VPC which will allow instances with
# public IPs to connect to the internet (if security groups and ACLs permit)
resource "aws_internet_gateway" "base_ami_builder" {
  vpc_id = "${aws_vpc.base_ami_builder.id}"

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.zone_name}"
    "substrate:role"        = "base_ami_builder-internet-gateway"
    "Name"                  = "${var.zone_prefix}-base-ami-builder-internet-gateway"
  }
}

# set up a route table that routes internet traffic through our gateway
# (it also implicitly routes internal traffic across the private "fabric")
resource "aws_route_table" "base_ami_builder" {
  vpc_id = "${aws_vpc.base_ami_builder.id}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.base_ami_builder.id}"
  }

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.zone_name}"
    "substrate:role"        = "ami-builder-route-table"
    "Name"                  = "${var.zone_prefix}-base-ami-builder-route-table"
  }
}

# associate our route table with the VPC
resource "aws_main_route_table_association" "base_ami_builder" {
  vpc_id         = "${aws_vpc.base_ami_builder.id}"
  route_table_id = "${aws_route_table.base_ami_builder.id}"
}

# create a single large subnet that takes up the entire VPC
resource "aws_subnet" "base_ami_builder" {
  vpc_id                  = "${aws_vpc.base_ami_builder.id}"
  cidr_block              = "${aws_vpc.base_ami_builder.cidr_block}"
  availability_zone       = "${var.aws_availability_zone}"
  map_public_ip_on_launch = false

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.zone_name}"
    "substrate:role"        = "ami-builder-subnet"
    "Name"                  = "${var.zone_prefix}-base-ami-builder-subnet"
  }

  # add some dependencies here so none of our instance get created before
  # DHCP and routing are set up.
  depends_on = [
    "aws_main_route_table_association.base_ami_builder",
    "aws_vpc_dhcp_options_association.base_ami_builder",
  ]
}

output "builder_subnet_id" {
  value = "${aws_subnet.base_ami_builder.id}"
}

resource "bakery_ebs_snapshot" "base" {
  region            = "${var.aws_region}"
  builder_subnet_id = "${aws_subnet.base_ami_builder.id}"

  builder_iam_instance_profile = "${var.iam_profile_name}"

  builder_security_groups = [
    "${aws_security_group.base_ami_builder.id}",
  ]

  builder_availability_zone = "${var.aws_availability_zone}"
  builder_source_ami        = "${var.ubuntu_ami}"

  builder_instance_tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.zone_name}"
    "substrate:role"        = "base-ami-snapshot"
    "Name"                  = "${var.zone_prefix}-base-ami-snapshot"
  }

  tags {
    "Name"                  = "baker"
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.zone_name}"
    "substrate:role"        = "base-ami-snapshot"
    "Name"                  = "${var.zone_prefix}-base-ami-snapshot"
  }

  builder_user_data = "${var.userdata}"

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [
    "aws_internet_gateway.base_ami_builder",
  ]
}

resource "aws_ami" "base" {
  name                = "${var.zone_prefix}-base-ami-${bakery_ebs_snapshot.base.id}"
  description         = "Substrate Base AMI ${var.substrate_version} for ${var.zone_name}"
  virtualization_type = "hvm"
  architecture        = "x86_64"
  root_device_name    = "/dev/xvda"

  ebs_block_device {
    device_name           = "/dev/xvda"
    snapshot_id           = "${bakery_ebs_snapshot.base.id}"
    volume_size           = 8
    volume_type           = "gp2"
    delete_on_termination = true
  }

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.zone_name}"
    "substrate:role"        = "base-ami"
    "Name"                  = "${var.zone_prefix}-base-ami"
  }

  lifecycle {
    create_before_destroy = true
  }
}

output "ami_id" {
  value = "${aws_ami.base.id}"
}
