variable "worker_pool_name" {}

variable "instance_count" {}

variable "has_public_ip" {}

variable "ami_id" {}

variable "instance_type" {}

variable "vpc_id" {}

variable "subnet_id" {}

variable "admin_key_name" {}

variable "substrate_environment" {}

variable "substrate_version" {}

variable "substrate_zone" {}

variable "zone_prefix" {}

variable "substrate_environment_subnet" {}

variable "director_internal_ip" {}

variable "default_calico_pool" {}

variable "border_internal_ip" {}

resource "aws_security_group" "workers" {
  name_prefix = "${var.zone_prefix}-${var.worker_pool_name}-security-group-"
  vpc_id      = "${var.vpc_id}"

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "${var.worker_pool_name}-security-group"
    "Name"                  = "${var.zone_prefix}-${var.worker_pool_name}-security-group"
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group_rule" "egress_to_director_discovery_port" {
  security_group_id = "${aws_security_group.workers.id}"
  type              = "egress"
  protocol          = "tcp"
  to_port           = 6443
  from_port         = 6443
  cidr_blocks       = ["${var.director_internal_ip}/32"]
}

resource "aws_security_group_rule" "egress_to_director_discovery_port_2" {
  security_group_id = "${aws_security_group.workers.id}"
  type              = "egress"
  protocol          = "tcp"
  to_port           = 9898
  from_port         = 9898
  cidr_blocks       = ["${var.director_internal_ip}/32"]
}

resource "aws_security_group_rule" "egress_to_default_pool" {
  security_group_id = "${aws_security_group.workers.id}"
  type              = "egress"
  protocol          = "tcp"
  to_port           = 65535
  from_port         = 0
  cidr_blocks       = ["${var.default_calico_pool}"]
}

resource "aws_security_group_rule" "ingress_to_default_pool" {
  security_group_id = "${aws_security_group.workers.id}"
  type              = "ingress"
  protocol          = "tcp"
  to_port           = 65535
  from_port         = 0
  cidr_blocks       = ["${var.default_calico_pool}"]
}

resource "aws_security_group_rule" "egress_to_default_pool_udp" {
  security_group_id = "${aws_security_group.workers.id}"
  type              = "egress"
  protocol          = "udp"
  to_port           = 65535
  from_port         = 0
  cidr_blocks       = ["${var.default_calico_pool}"]
}

resource "aws_security_group_rule" "ingress_to_default_pool_udp" {
  security_group_id = "${aws_security_group.workers.id}"
  type              = "ingress"
  protocol          = "udp"
  to_port           = 65535
  from_port         = 0
  cidr_blocks       = ["${var.default_calico_pool}"]
}

resource "aws_security_group_rule" "ingress_from_environment" {
  security_group_id = "${aws_security_group.workers.id}"
  type              = "ingress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["${var.substrate_environment_subnet}"]
}

resource "aws_security_group_rule" "egress_to_environment" {
  security_group_id = "${aws_security_group.workers.id}"
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["${var.substrate_environment_subnet}"]
}

resource "aws_instance" "workers" {
  count         = "${var.instance_count}"
  ami           = "${var.ami_id}"
  instance_type = "${var.instance_type}"
  key_name      = "${var.admin_key_name}"

  source_dest_check = false

  associate_public_ip_address = "${var.has_public_ip}"

  subnet_id = "${var.subnet_id}"

  security_groups = [
    "${aws_security_group.workers.id}",
  ]

  monitoring    = false
  ebs_optimized = false

  root_block_device {
    volume_type = "gp2"
  }

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "${var.worker_pool_name}"
    "Name"                  = "${var.zone_prefix}-${var.worker_pool_name}"
  }

  user_data = <<EOF
{
    "substrate": {
        "role": ${jsonencode(var.worker_pool_name)},
        "director": ${jsonencode(var.director_internal_ip)},
        "border": ${jsonencode(var.border_internal_ip)}
    }

}
EOF

  lifecycle {
    create_before_destroy = true
  }
}

output "security_group_id" {
  value = "${aws_security_group.workers.id}"
}

output "public_ips" {
  value = ["${aws_instance.workers.*.public_ip}"]
}
