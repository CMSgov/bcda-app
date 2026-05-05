# CloudWatch Events Module
data "aws_region" "current" {}

# Example : Use for Cron/Rate jobs for AWS Lambda
resource "aws_cloudwatch_event_rule" "cw_rule" {
  name                = var.name
  description         = var.description
  schedule_expression = var.schedule
}

resource "aws_cloudwatch_event_target" "cw_event_target" {
  rule      = aws_cloudwatch_event_rule.cw_rule.name
  target_id = var.target_id
  arn       = var.arn
}
