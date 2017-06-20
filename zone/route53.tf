resource "aws_route53_zone" "public" {
  name              = "${var.substrate_zone}.${var.substrate_environment_domain}"
  comment           = "public zone for ${var.substrate_zone}"
  delegation_set_id = "${var.delegation_set_id}"

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "public-dns"
    "Name"                  = "${var.zone_prefix}-public-dns"
  }
}
