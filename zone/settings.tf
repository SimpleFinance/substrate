variable "substrate_version" {
  description = "version of infrastructure/bootstrap (used in tags)"
}

variable "zone_prefix" {
  description = "prefix for all named objects in this zone"
}

variable "substrate_environment" {
  description = "name for this zone's environment (used in tags and as part of the DNS names)"
}

variable "substrate_environment_domain" {
  description = "the primary domain under which this zone's environment lives."
}

variable "substrate_environment_subnet" {
  description = "the private IPv4 subnet for all the hosts in this zone's environment"
}

variable "substrate_zone" {
  description = "name for this zone (used in tags and as part of the DNS names)"
}

variable "substrate_zone_subnet" {
  description = "the private IPv4 subnet for all the hosts in this zone"
}

variable "substrate_cloudwatch_logs_group_system_logs" {
  description = "the name of the CloudWatch Logs group for this zone's system logs"
}

variable "substrate_cloudwatch_logs_group_vpc_flow_logs" {
  description = "the name of the CloudWatch Logs group for this zone's VPC flow logs"
}

variable "aws_region" {
  description = "AWS region in which resources will be created"
}

variable "aws_availability_zone" {
  description = "AWS availability zone in which resources will be created"
}

# aws_account_id is a sanity check to make sure we're running in the AWS account we expect
# (other AWS settings, e.g., AWS_ACCESS_KEY_ID, are read from env vars)
variable "aws_account_id" {
  description = "ID of AWS account in which resources will be created"
}

variable "delegation_set_id" {
  description = "Route53 Reusable Delegation Set ID to use for the Route53 Hosted Zone for this zone"
}

provider "aws" {
  region              = "${var.aws_region}"
  allowed_account_ids = ["${var.aws_account_id}"]
}

variable "pool_subnet_0" {
  description = "calico's default subnet pool"
  default     = "192.168.0.0/16"
}

variable "cluster_dns" {
  description = "fixed kubernetes service address for cluster dns server"
  default     = "10.100.0.10"
}

# this is the path to the public key we want to configure as the default admin SSH key
variable "ssh_public_key" {
  description = "path to an SSH public key that will be given admin access (e.g., ~/.ssh/you.pub)"
}

variable "instance_type" {
  description = "Type of worker instances to launch"
  default     = "t2.medium"
}

variable "director_instance_type" {
  description = "Type of director instances to launch"
  default     = "t2.medium"
}

variable "border_instance_type" {
  description = "Type of border instances to launch"
  default     = "t2.medium"
}

variable "calico_etcd_port" {
  description = "Port for clients of calico etcd authority"
  default     = 4002
}

variable "kubernetes_api_port" {
  default     = "443"
  description = "TCP port used for clients to connect to kubernetes"
}
