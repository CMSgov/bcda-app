#############################################
# To be removed post-migration to bcda-app. #
# Handles imports for moving state from ops #
# repo to app repo.                         #
#############################################

################
# Data lookups #
################

# ECS Service
import {
  to = module.ecs_worker.aws_ecs_service.this
  id = "${data.terraform_remote_state.cluster.outputs.cluster_name}/bcda-${module.platform.env}-worker"
}

# IAM Execution Role
import {
  to = module.ecs_worker.aws_iam_role.execution[0]
  id = "bcda-${module.platform.env}-worker-execution"
}

# IAM Task Role
import {
  to = module.ecs_worker.aws_iam_role.task
  id = "bcda-${module.platform.env}-worker-task-role"
}

# IAM Role Policies
import {
  to = module.ecs_worker.aws_iam_role_policy.execution[0]
  id = "bcda-${module.platform.env}-worker-execution:bcda-${module.platform.env}-worker-execution"
}

import {
  to = module.ecs_worker.aws_iam_role_policy.task
  id = "bcda-${module.platform.env}-worker-task-role:bcda-${module.platform.env}-worker-task-policy"
}

# IAM Role Policy Attachment
import {
  to = module.ecs_worker.aws_iam_role_policy_attachment.task_additional["bootstrap_policy"]
  id = "bcda-${module.platform.env}-worker-task-role/${data.aws_iam_policy.worker_task_import.arn}"
}

# Autoscaling Target
import {
  to = aws_appautoscaling_target.ecs_worker_target
  id = "ecs/service/bcda-${module.platform.env}/bcda-${module.platform.env}-worker/ecs:service:DesiredCount"
}

# Autoscaling Policy
import {
  to = aws_appautoscaling_policy.ecs_worker_policy
  id = "bcda-${module.platform.env}-worker-job-count-scaleup/ecs/service/bcda-${module.platform.env}/bcda-${module.platform.env}-worker/ecs:service:DesiredCount"
}

# CloudWatch Metric Alarm
import {
  to = aws_cloudwatch_metric_alarm.ecs_worker_job_count
  id = "bcda-${module.platform.env}-worker-job-count"
}

# Log Groups
import {
  to = module.ecs_worker.aws_cloudwatch_log_group.app
  id = "/aws/ecs/fargate/bcda-${module.platform.env}/worker"
}

import {
  to = module.ecs_worker.aws_cloudwatch_log_group.datadog[0]
  id = "/aws/ecs/fargate/bcda-${module.platform.env}/worker/datadog-agent"
}

# SSM image tag parameter
import {
  to = module.ecs_worker.aws_ssm_parameter.image_tag
  id = "/bcda/${module.platform.env}/nonsensitive/worker/image-tag"
}

# ECS Alarms
import {
  to = module.worker-ecs-alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${var.env}-worker-cpu-critical"]
  id = "bcda-${module.platform.env}-worker-cpu-critical"
}

import {
  to = module.worker-ecs-alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${var.env}-worker-cpu-warn"]
  id = "bcda-${module.platform.env}-worker-cpu-warn"
}

import {
  to = module.worker-ecs-alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${var.env}-worker-memory-critical"]
  id = "bcda-${module.platform.env}-worker-memory-critical"
}

import {
  to = module.worker-ecs-alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${var.env}-worker-memory-warn"]
  id = "bcda-${module.platform.env}-worker-memory-warn"
}

# Data lookup for policy attachment import
data "aws_iam_policy" "worker_task_import" {
  name = "bcda-${module.platform.env}-worker-task"
}
