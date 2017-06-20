variable "director_name" {}

variable "internal_ip" {}

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

variable "substrate_zone_subnet" {}

variable "kubernetes_api_port" {}

variable "default_calico_pool" {}

variable "border_internal_ip" {}

variable "etcd_internal_port" {
  default     = "2380"
  description = "TCP port used for peer to peer etcd communication"
}

variable "etcd_client_port" {
  default     = "2379"
  description = "TCP port used for clients to connect to etcd"
}

resource "aws_security_group" "director" {
  name_prefix = "${var.zone_prefix}-${var.director_name}-security-group-"
  vpc_id      = "${var.vpc_id}"

  # inbound kubernetes API traffic from the entire environment
  ingress {
    from_port   = "${var.kubernetes_api_port}"
    to_port     = "${var.kubernetes_api_port}"
    protocol    = "tcp"
    cidr_blocks = ["${var.substrate_environment_subnet}"]
  }

  # ETCD Authority for calico
  ingress {
    from_port   = "4002"
    to_port     = "4002"
    protocol    = "tcp"
    cidr_blocks = ["${var.substrate_zone_subnet}"]
  }

  # ssh to border
  ingress {
    from_port   = "22"
    to_port     = "22"
    protocol    = "tcp"
    cidr_blocks = ["${var.border_internal_ip}/32"]
  }

  # k8s discovery port
  ingress {
    from_port   = "9898"
    to_port     = "9898"
    protocol    = "tcp"
    cidr_blocks = ["${var.substrate_zone_subnet}"]
  }

  ingress {
    from_port   = "6443"
    to_port     = "6443"
    protocol    = "tcp"
    cidr_blocks = ["${var.substrate_zone_subnet}"]
  }

  # For Calico BGP
  ingress {
    from_port   = "179"
    to_port     = "179"
    protocol    = "tcp"
    cidr_blocks = ["${var.substrate_zone_subnet}"]
  }

  # ship logs to border
  egress {
    from_port   = "19532"
    to_port     = "19532"
    protocol    = "tcp"
    cidr_blocks = ["${var.border_internal_ip}/32"]
  }

  # Allow the director to hit anything internally
  egress {
    from_port   = "0"
    to_port     = "0"
    protocol    = "-1"
    cidr_blocks = ["${var.substrate_environment_subnet}"]
  }

  # Calico pool subnet
  # allow pods to call talk to each other
  egress {
    protocol  = "tcp"
    to_port   = 65535
    from_port = 0

    cidr_blocks = [
      "${var.default_calico_pool}",
    ]
  }

  ingress {
    protocol  = "tcp"
    to_port   = 65535
    from_port = 0

    cidr_blocks = [
      "${var.default_calico_pool}",
    ]
  }

  egress {
    protocol    = "udp"
    to_port     = 65535
    from_port   = 0
    cidr_blocks = ["${var.default_calico_pool}"]
  }

  ingress {
    protocol    = "udp"
    to_port     = 65535
    from_port   = 0
    cidr_blocks = ["${var.default_calico_pool}"]
  }

  # node_exporter http
  ingress {
    from_port   = "9100"
    to_port     = "9100"
    protocol    = "tcp"
    cidr_blocks = ["${var.substrate_zone_subnet}"]
  }

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "${var.director_name}-security-group"
    "Name"                  = "${var.zone_prefix}-${var.director_name}-security-group"
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_instance" "director" {
  ami           = "${var.ami_id}"
  instance_type = "${var.instance_type}"
  key_name      = "${var.admin_key_name}"

  source_dest_check = false

  associate_public_ip_address = false

  subnet_id  = "${var.subnet_id}"
  private_ip = "${var.internal_ip}"

  vpc_security_group_ids = [
    "${aws_security_group.director.id}",
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
    "substrate:role"        = "${var.director_name}"
    "Name"                  = "${var.zone_prefix}-${var.director_name}"
  }

  user_data = <<EOF
{
    "substrate": {
        "role": ${jsonencode(var.director_name)},
        "director": ${jsonencode(var.internal_ip)},
        "border": ${jsonencode(var.border_internal_ip)}
    }
}
EOF
}

output "internal_ip" {
  value = "${var.internal_ip}"
}

output "sg_id" {
  value = "${aws_security_group.director.id}"
}
