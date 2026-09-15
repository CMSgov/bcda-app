data "aws_sqs_queue" "alarm_to_slack" {
  name = "cdap-${local.cdap_env}-alarm-to-slack"
}

resource "aws_sns_topic" "cloudwatch_alarms_topic" {
  display_name      = local.cloudwatch_alarms_topic_name
  name              = local.cloudwatch_alarms_topic_name
  kms_master_key_id = module.platform.kms_alias_primary.target_key_arn
}

resource "aws_sns_topic" "cloudwatch_critical_alarms_topic" {
  count             = local.is_prod ? 1 : 0
  display_name      = local.cloudwatch_critical_alarms_topic_name
  name              = local.cloudwatch_critical_alarms_topic_name
  kms_master_key_id = module.platform.kms_alias_primary.target_key_arn
}

resource "aws_sns_topic_subscription" "alarm_to_slack" {
  topic_arn = aws_sns_topic.cloudwatch_alarms_topic.arn
  protocol  = "sqs"
  endpoint  = data.aws_sqs_queue.alarm_to_slack.arn
}

resource "aws_cloudwatch_metric_alarm" "excessive-job-count" {
  alarm_name          = "bcda-${local.env}-excessive-job-count"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "1"
  metric_name         = "JobQueueCount"
  namespace           = "BCDA"
  period              = "30"
  statistic           = "Maximum"
  threshold           = "10000"

  dimensions = {
    Environment = local.env
  }

  alarm_description = "Excessive Job Count Alarm"
  alarm_actions     = local.is_prod ? [aws_sns_topic.cloudwatch_critical_alarms_topic[0].arn] : [aws_sns_topic.cloudwatch_alarms_topic.arn]
}
