# create a virtual network with the specified CIDR
resource "aws_vpc" "main" {
  cidr_block           = "${var.substrate_zone_subnet}"
  enable_dns_support   = false
  enable_dns_hostnames = false

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "main-vpc"
    "Name"                  = "${var.zone_prefix}-main-vpc"
  }
}

# create some DHCP settings for everything spun up in the VPC
resource "aws_vpc_dhcp_options" "main" {
  domain_name         = "zone.local"
  domain_name_servers = ["${module.border-0.internal_ip}"]

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "main-dhcp-options"
    "Name"                  = "${var.zone_prefix}-main-dhcp-options"
  }
}

# associate our DHCP settings with the VPC
resource "aws_vpc_dhcp_options_association" "main" {
  vpc_id          = "${aws_vpc.main.id}"
  dhcp_options_id = "${aws_vpc_dhcp_options.main.id}"
}

# set up an "internet gateway" in our VPC which will allow instances with
# public IPs to connect to the internet (if security groups and ACLs permit)
resource "aws_internet_gateway" "main" {
  vpc_id = "${aws_vpc.main.id}"

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "main-internet-gateway"
    "Name"                  = "${var.zone_prefix}-main-internet-gateway"
  }
}

# set up a route table that routes internet traffic through our gateway
# (it also implicitly routes internal traffic across the private "fabric")
resource "aws_route_table" "main" {
  vpc_id = "${aws_vpc.main.id}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.main.id}"
  }

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "main-route-table"
    "Name"                  = "${var.zone_prefix}-main-route-table"
  }
}

# associate our route table with the VPC
resource "aws_main_route_table_association" "main" {
  vpc_id         = "${aws_vpc.main.id}"
  route_table_id = "${aws_route_table.main.id}"
}

# create a single large subnet that takes up the entire VPC
resource "aws_subnet" "main" {
  vpc_id                  = "${aws_vpc.main.id}"
  cidr_block              = "${aws_vpc.main.cidr_block}"
  availability_zone       = "${var.aws_availability_zone}"
  map_public_ip_on_launch = false

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "main-subnet"
    "Name"                  = "${var.zone_prefix}-main-subnet"
  }

  # add some dependencies here so none of our instance get created before
  # DHCP and routing are set up.
  depends_on = [
    "aws_main_route_table_association.main",
    "aws_vpc_dhcp_options_association.main",
  ]
}
