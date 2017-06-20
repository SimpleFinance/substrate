resource "aws_cloudwatch_log_group" "zone_system_logs" {
  name              = "${var.substrate_cloudwatch_logs_group_system_logs}"
  retention_in_days = 1
}
