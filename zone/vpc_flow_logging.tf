resource "aws_cloudwatch_log_group" "zone_vpc_flow_logs" {
  name              = "${var.substrate_cloudwatch_logs_group_vpc_flow_logs}"
  retention_in_days = 1
}

# add flow log to capture netflow/ipfix metadata on all traffic
resource "aws_flow_log" "zone_vpc_flow_logs" {
  log_group_name = "${aws_cloudwatch_log_group.zone_vpc_flow_logs.name}"
  iam_role_arn   = "${aws_iam_role.vpc_flow_logs_role.arn}"
  vpc_id         = "${aws_vpc.main.id}"
  traffic_type   = "ALL"
}

resource "aws_iam_role" "vpc_flow_logs_role" {
  name = "${var.zone_prefix}-vpc-flow-logging"

  assume_role_policy = <<EOF
{
	"Version":"2012-10-17",
	"Statement": [
	{
		"Sid": "",
		"Effect": "Allow",
		"Principal": {
		"Service": "vpc-flow-logs.amazonaws.com"
	},
	"Action":"sts:AssumeRole"
	}]
}
EOF
}

resource "aws_iam_role_policy" "vpc_flow_logs_policy" {
  name = "${var.zone_prefix}-vpc-flow-logging"
  role = "${aws_iam_role.vpc_flow_logs_role.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "logs:DescribeLogGroups",
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogStreams"
      ],
      "Effect": "Allow",
      "Resource": [
        "${aws_cloudwatch_log_group.zone_vpc_flow_logs.arn}",
        "${aws_cloudwatch_log_group.zone_vpc_flow_logs.arn}:log-stream:*"

      ]
    }
  ]
}
EOF
}
