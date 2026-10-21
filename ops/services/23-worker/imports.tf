#############################################
# To be removed post-migration to bcda-app. #
# Handles imports for moving state from ops #
# repo to app repo.                         #
#############################################

################
# Data lookups #
################

data "aws_iam_role" "worker_execution_import" {
  name = "bcda-${module.platform.env}-worker-execution"
}

data "aws_iam_role" "worker_task_role_import" {
  name = "bcda-${module.platform.env}-worker-task-role"
}

data "aws_ecs_task_definition" "worker_import" {
  task_definition = "bcda-${module.platform.env}-worker"
}

###########
# Imports #
###########

import {
  to = module.ecs_worker.aws_ssm_parameter.image_tag
  id = "/bcda/${module.platform.env}/nonsensitive/worker/image-tag"
}

# --- ECS --- #

import {
  to = module.ecs_worker.aws_ecs_service.this
  id = "${data.aws_ecs_cluster.this.cluster_name}/bcda-${module.platform.env}-worker"
}

import {
  to = module.ecs_worker.aws_ecs_task_definition.this
  id = data.aws_ecs_task_definition.worker_import.arn
}

# --- IAM --- #

import {
  to = module.ecs_worker.aws_iam_role.execution[0]
  id = "bcda-${module.platform.env}-worker-execution"
}

import {
  to = module.ecs_worker.aws_iam_role.task
  id = "bcda-${module.platform.env}-worker-task-role"
}

import {
  to = module.ecs_worker.aws_iam_role_policy.execution[0]
  id = "bcda-${module.platform.env}-worker-execution:bcda-${module.platform.env}-worker-execution"
}

import {
  to = module.ecs_worker.aws_iam_role_policy.task
  id = "bcda-${module.platform.env}-worker-task-role:bcda-${module.platform.env}-worker-task-policy"
}

# --- Autoscaling --- #

import {
  to = aws_appautoscaling_target.ecs_worker_target
  id = "ecs/service/${data.aws_ecs_cluster.this.cluster_name}/bcda-${module.platform.env}-worker/ecs:service:DesiredCount"
}

import {
  to = aws_appautoscaling_policy.ecs_worker_policy
  id = "ecs/service/${data.aws_ecs_cluster.this.cluster_name}/bcda-${module.platform.env}-worker/ecs:service:DesiredCount/bcda-${module.platform.env}-worker-job-count-scaleup"
}

# --- Cloudwatch --- #

import {
  to = aws_cloudwatch_metric_alarm.ecs_worker_job_count
  id = "bcda-${module.platform.env}-worker-job-count"
}

import {
  to = module.ecs_worker.aws_cloudwatch_log_group.app
  id = "/aws/ecs/fargate/bcda-${module.platform.env}/worker"
}

import {
  to = module.ecs_worker.aws_cloudwatch_log_group.datadog[0]
  id = "/aws/ecs/fargate/bcda-${module.platform.env}/worker/datadog-agent"
}

# --- ECS Alarms --- #

import {
  to = module.worker_ecs_alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-worker-cpu-critical"]
  id = "bcda-${module.platform.env}-worker-cpu-critical"
}

import {
  to = module.worker_ecs_alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-worker-cpu-warn"]
  id = "bcda-${module.platform.env}-worker-cpu-warn"
}

import {
  to = module.worker_ecs_alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-worker-memory-critical"]
  id = "bcda-${module.platform.env}-worker-memory-critical"
}

import {
  to = module.worker_ecs_alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-worker-memory-warn"]
  id = "bcda-${module.platform.env}-worker-memory-warn"
}
