

resource "aws_cloudwatch_metric_alarm" "ecs_alarms" {
  for_each = { for idx in var.alarms : idx.alarm_name => idx }

  alarm_name        = each.value.alarm_name
  alarm_description = "${each.value.alarm_description}: ${var.service_name}"

  metric_name = each.value.metric_name
  namespace   = "AWS/ECS"

  dimensions = {
    ClusterName = var.cluster_name
    ServiceName = var.service_name
  }

  statistic           = "Average"
  threshold           = each.value.threshold
  comparison_operator = "GreaterThanThreshold"

  period             = each.value.period
  evaluation_periods = each.value.eval_periods

  alarm_actions = [var.alarm_notification_arn]
  ok_actions    = [var.ok_notification_arn]

  treat_missing_data = "ignore"

}
