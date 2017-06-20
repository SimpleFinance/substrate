resource "aws_key_pair" "admin" {
  key_name   = "${var.zone_prefix}-admin-ssh"
  public_key = "${file(var.ssh_public_key)}"
}
