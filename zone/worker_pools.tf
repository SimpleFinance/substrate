# ==============================================================================
# Single default worker pool for all app workloads (for now).
# ==============================================================================
module "workers" {
  source           = "./worker_pool"
  worker_pool_name = "worker"
  instance_count   = 4
  has_public_ip    = true

  # a bunch of basic stuff that doesn't vary across the worker pools but needs
  # to be passed down into the module
  ami_id = "${module.bake.ami_id}"

  admin_key_name               = "${aws_key_pair.admin.key_name}"
  default_calico_pool          = "${var.pool_subnet_0}"
  director_internal_ip         = "${module.director-0.internal_ip}"
  substrate_environment_subnet = "${var.substrate_environment_subnet}"
  instance_type                = "${var.instance_type}"
  subnet_id                    = "${aws_subnet.main.id}"
  substrate_environment        = "${var.substrate_environment}"
  substrate_version            = "${var.substrate_version}"
  substrate_zone               = "${var.substrate_zone}"
  zone_prefix                  = "${var.zone_prefix}"
  vpc_id                       = "${aws_vpc.main.id}"
  border_internal_ip           = "${module.border-0.internal_ip}"
}

resource "aws_security_group_rule" "workers_ingress_tcp_80" {
  security_group_id = "${module.workers.security_group_id}"
  type              = "ingress"
  from_port         = 80
  to_port           = 80
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
}

resource "aws_security_group_rule" "workers_ingress_tcp_443" {
  security_group_id = "${module.workers.security_group_id}"
  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
}

resource "aws_route53_record" "workers" {
  zone_id = "${aws_route53_zone.public.zone_id}"
  name    = "worker.${aws_route53_zone.public.name}"
  type    = "A"
  ttl     = "30"
  records = ["${module.workers.public_ips}"]
}

output "worker_public_ips" {
  value = "${module.workers.public_ips}"
}

output "worker_dns" {
  value = "${aws_route53_record.workers.name}"
}
