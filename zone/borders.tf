# ==============================================================================
# Single Border instance (runs egress proxy, DNS, and NTP)
# ==============================================================================
module "border-0" {
  source      = "./border"
  border_name = "border-0"

  # use the 5th IP address in the substrate_zone_subnet subnet
  # (the first 3 addresses are reserved by AWS VPC, the 4th is director-0)
  internal_ip = "${cidrhost(var.substrate_zone_subnet, 5)}"

  # a bunch of basic variables that we need to pass down from the top level
  # namespace so it ends up available in the director module namespace
  ami_id = "${module.bake.ami_id}"

  instance_type                = "${var.border_instance_type}"
  vpc_id                       = "${aws_vpc.main.id}"
  subnet_id                    = "${aws_subnet.main.id}"
  admin_key_name               = "${aws_key_pair.admin.key_name}"
  substrate_environment        = "${var.substrate_environment}"
  substrate_version            = "${var.substrate_version}"
  substrate_zone               = "${var.substrate_zone}"
  zone_prefix                  = "${var.zone_prefix}"
  substrate_environment_subnet = "${var.substrate_environment_subnet}"
  substrate_zone_subnet        = "${var.substrate_zone_subnet}"
  kubernetes_api_port          = "${var.kubernetes_api_port}"
  director_internal_ip         = "${module.director-0.internal_ip}"
  default_calico_pool          = "${var.pool_subnet_0}"
  calico_etcd_port             = "${var.calico_etcd_port}"
  cloudwatch_logs_group_arn    = "${aws_cloudwatch_log_group.zone_system_logs.arn}"
}

resource "aws_route53_record" "border-0" {
  zone_id = "${aws_route53_zone.public.zone_id}"
  name    = "border.${aws_route53_zone.public.name}"
  type    = "A"
  ttl     = "30"
  records = ["${module.border-0.public_ip}"]
}

# an output to make `mkzone ssh director` work
output "border_eip" {
  value = "${module.border-0.public_ip}"
}

output "border_dns" {
  value = "${aws_route53_record.border-0.name}"
}
