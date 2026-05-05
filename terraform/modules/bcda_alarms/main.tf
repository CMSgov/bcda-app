resource "aws_cloudwatch_metric_alarm" "excessive-job-count" {
  alarm_name          = "bcda-${var.env}-excessive-job-count"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "1"
  metric_name         = "JobQueueCount"
  namespace           = "BCDA"
  period              = "30"
  statistic           = "Maximum"
  threshold           = "10000"

  dimensions = {
    Environment = var.env
  }

  alarm_description = "Excessive Job Count Alarm"
  alarm_actions     = [var.cloudwatch_notification_arn]
}

