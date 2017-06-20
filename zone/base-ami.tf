resource "random_id" "token" {
  byte_length = 8
}

resource "random_id" "token_prefix" {
  byte_length = 6
}

# this is written out to *before* provision.sh runs /etc/substrate/zone.env
data "template_file" "zone_env" {
  template = "${file("${path.module}/zone.env")}"

  vars {
    token                 = "substr.${random_id.token.hex}"
    substrate_environment = "${var.substrate_environment}"
    substrate_version     = "${var.substrate_version}"
    substrate_zone        = "${var.substrate_zone}"
    calico_etcd_port      = "${var.calico_etcd_port}"
    kubernetes_api_port   = "${var.kubernetes_api_port}"
    cluster_dns_server    = "${var.cluster_dns}"
    aws_region            = "${var.aws_region}"
    aws_availability_zone = "${var.aws_availability_zone}"
    cloudwatch_logs_group = "${aws_cloudwatch_log_group.zone_system_logs.name}"
  }
}

data "tarball_file" "provisioner" {
  gzip_level         = 9
  directory          = "zone/base-ami-provision"
  override_timestamp = "2015-12-31T18:34:13Z"
  override_owner     = 0
  override_group     = 0
}

# a random ID to use in our S3 prefix to make it hard to brute force and find the provision tarball in S3
resource "random_id" "provision_prefix" {
  byte_length = 16
}

# set up a bucket to hold generated provisioning tarball
resource "aws_s3_bucket" "provisioner" {
  bucket        = "${var.zone_prefix}-base-ami-provision"
  acl           = "private"
  force_destroy = true

  tags {
    "substrate:environment" = "${var.substrate_environment}"
    "substrate:version"     = "${var.substrate_version}"
    "substrate:zone"        = "${var.substrate_zone}"
    "substrate:role"        = "base-ami-provision"
    "Name"                  = "${var.zone_prefix}-base-ami-provision"
  }

  depends_on = [
    "data.tarball_file.provisioner",
  ]

  policy = <<EOF
{
  "Version":"2012-10-17",
  "Statement":[
    {
      "Effect":"Allow",
      "Principal": "*",
      "Action":["s3:GetObject"],
      "Resource":["arn:aws:s3:::${var.zone_prefix}-base-ami-provision/*"]
    }
  ]
}
EOF
}

resource "aws_s3_bucket_object" "provisioner" {
  bucket  = "${aws_s3_bucket.provisioner.bucket}"
  key     = "provision-${var.substrate_version}-${random_id.provision_prefix.hex}-${data.tarball_file.provisioner.sha256}.tgz.b64"
  content = "${data.tarball_file.provisioner.contents_base64}"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_iam_role" "provisioner" {
  name = "${var.zone_prefix}-base-ami-provision"
  path = "/"

  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Principal": {
          "Service": "ec2.amazonaws.com"
        },
        "Action": "sts:AssumeRole"
      }
    ]
  }
EOF
}

resource "aws_iam_role_policy" "provisioner" {
  name = "${var.zone_prefix}-base-ami-provision"
  role = "${aws_iam_role.provisioner.id}"

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
        "${aws_cloudwatch_log_group.zone_system_logs.arn}",
        "${aws_cloudwatch_log_group.zone_system_logs.arn}:log-stream:*"

      ]
    }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "provisioner" {
  name  = "${var.zone_prefix}-base-ami-provision"
  roles = ["${aws_iam_role.provisioner.name}"]

  provisioner "local-exec" {
    command = "/bin/sleep 30"
  }
}

data "template_file" "builder_user_data" {
  depends_on = ["aws_s3_bucket_object.provisioner"]
  template   = "${file("${path.module}/builder_user_data.yaml")}"

  vars {
    payload_s3_uri = "https://${aws_s3_bucket_object.provisioner.bucket}.s3.amazonaws.com/provision-${var.substrate_version}-${random_id.provision_prefix.hex}-${data.tarball_file.provisioner.sha256}.tgz.b64"

    # ${aws_s3_bucket_object.provisioner.key}
    payload_checksum = "${data.tarball_file.provisioner.sha256} /tmp/provision.tgz"
    zone_env         = "${base64encode(data.template_file.zone_env.rendered)}"
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  owners = ["099720109477"] # Canonical
}

module "bake" {
  source                = "./node-ami"
  userdata              = "${data.template_file.builder_user_data.rendered}"
  substrate_environment = "${var.substrate_environment}"
  substrate_version     = "${var.substrate_version}"
  calico_etcd_port      = "${var.calico_etcd_port}"
  kubernetes_api_port   = "${var.kubernetes_api_port}"
  cluster_dns_server    = "${var.cluster_dns}"
  aws_region            = "${var.aws_region}"
  aws_availability_zone = "${var.aws_availability_zone}"
  cluster_dns           = "${var.cluster_dns}"
  zone_name             = "${var.substrate_zone}"
  zone_prefix           = "${var.zone_prefix}"
  ubuntu_ami            = "${data.aws_ami.ubuntu.id}"
  iam_profile_name      = "${aws_iam_instance_profile.provisioner.name}"
}

output "ami_node" {
  value = "${module.bake.ami_id}"
}

output "ami_ubuntu" {
  value = "${data.aws_ami.ubuntu.id}"
}
