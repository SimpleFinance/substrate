# ==============================================================================
# Single Director instance (runs k8s apiserver and other centralized bits)
# ==============================================================================
module "director-0" {
  source        = "./director"
  director_name = "director-0"

  # use the 4th IP address in the substrate_zone_subnet subnet
  # (the first 3 addresses are reserved by AWS VPC)
  internal_ip = "${cidrhost(var.substrate_zone_subnet, 4)}"

  # a bunch of basic variables that we need to pass down from the top level
  # namespace so it ends up available in the director module namespace
  ami_id = "${module.bake.ami_id}"

  instance_type                = "${var.director_instance_type}"
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
  default_calico_pool          = "${var.pool_subnet_0}"
  border_internal_ip           = "${module.border-0.internal_ip}"
}

# an output to make `mkzone ssh director` work
output "director_ip" {
  value = "${module.director-0.internal_ip}"
}
