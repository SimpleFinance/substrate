variable "border_name" {}

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

variable "kubernetes_api_port" {}

variable "substrate_environment_subnet" {}

variable "substrate_zone_subnet" {}

variable "director_internal_ip" {}

variable "default_calico_pool" {}

variable "calico_etcd_port" {}

variable "cloudwatch_logs_group_arn" {}

resource "aws_security_group" "border" {
  name_prefix = "${var.zone_prefix}-${var.border_name}-security-group-"
  vpc_id      = "${var.vpc_id}"

  # Allow HTTPS egress to the internet
  egress {
    from_port   = "443"
    to_port     = "443"
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # egress to director disco port
  egress {
    from_port   = "6443"
    to_port     = "6443"
    protocol    = "tcp"
    cidr_blocks = ["${var.director_internal_ip}/32"]
  }

  egress {
    from_port   = "9898"
    to_port     = "9898"
    protocol    = "tcp"
    cidr_blocks = ["${var.director_internal_ip}/32"]
  }

  # Allow DNS egress to the internet
  egress {
    from_port   = "53"
    to_port     = "53"
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = "53"
    to_port     = "53"
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow NTP egress to the internet
  egress {
    from_port   = "123"
    to_port     = "123"
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow SSH ingress from known admin IPs
  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"

    cidr_blocks = [
      # TODO: this was ported from a version that hardcoded some internal Simple ranges
      "0.0.0.0/0",
    ]
  }

  # Allow ingress from the rest of the environment
  ingress {
    from_port   = "0"
    to_port     = "0"
    protocol    = "-1"
    cidr_blocks = ["${var.substrate_environment_subnet}"]
  }

  # Allow SSH egress to the rest of the zone
  egress {
    from_port   = "22"
    to_port     = "22"
    protocol    = "tcp"
    cidr_blocks = ["${var.substrate_zone_subnet}"]
  }

  # Allow egress to the director
  egress {
    from_port   = "${var.kubernetes_api_port}"
    to_port     = "${var.kubernetes_api_port}"
    protocol    = "tcp"
    cidr_blocks = ["${var.director_internal_ip}/32"]
  }

  egress {
    from_port   = "${var.calico_etcd_port}"
    to_port     = "${var.calico_etcd_port}"
    protocol    = "tcp"
    cidr_blocks = ["${var.director_internal_ip}/32"]
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
    protocol    = "tcp"
    to_port     = 65535
    from_port   = 0
    cidr_blocks = ["${var.default_calico_pool}"]
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

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "${var.border_name}-security-group"
    "Name"                  = "${var.zone_prefix}-${var.border_name}-security-group"
  }

  lifecycle {
    create_before_destroy = true
  }
}

# ==============================================================================
# An IAM role for this border instance
# ==============================================================================
resource "aws_iam_role" "border" {
  name = "${var.zone_prefix}-${var.border_name}"
  path = "/"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "border_profile" {
  name  = "${var.zone_prefix}-${var.border_name}"
  roles = ["${aws_iam_role.border.name}"]
}

resource "aws_iam_role_policy" "border_policy" {
  name = "${var.zone_prefix}-${var.border_name}"
  role = "${aws_iam_role.border.id}"

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams"
        ],
        "Resource": [
          "${var.cloudwatch_logs_group_arn}",
          "${var.cloudwatch_logs_group_arn}:log-stream:*"
        ]
      },
      {
        "Effect":"Allow",
        "Action":"ec2:DescribeInstances",
        "Resource": "*"
      }
    ]
}
EOF
}

resource "aws_instance" "border" {
  ami           = "${var.ami_id}"
  instance_type = "${var.instance_type}"
  key_name      = "${var.admin_key_name}"

  associate_public_ip_address = true

  subnet_id  = "${var.subnet_id}"
  private_ip = "${var.internal_ip}"

  source_dest_check = false

  vpc_security_group_ids = [
    "${aws_security_group.border.id}",
  ]

  monitoring    = false
  ebs_optimized = false

  root_block_device {
    volume_type = "gp2"
  }

  iam_instance_profile = "${aws_iam_instance_profile.border_profile.name}"

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "${var.border_name}"
    "Name"                  = "${var.zone_prefix}-${var.border_name}"
  }

  user_data = <<EOF
{
  "substrate": {
    "role": ${jsonencode(var.border_name)},
    "director": ${jsonencode(var.director_internal_ip)},
    "border": ${jsonencode(var.internal_ip)}
  }
}
EOF
}

resource "aws_eip" "border" {
  instance = "${aws_instance.border.id}"
  vpc      = true
}

output "public_ip" {
  value = "${aws_eip.border.public_ip}"
}

output "internal_ip" {
  value = "${var.internal_ip}"
}
