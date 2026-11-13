#############################################
# To be removed post-migration to bcda-app. #
# Handles imports for moving state from ops #
# repo to app repo.                         #
#############################################

################
# Data lookups #
################

data "aws_ecs_task_definition" "api_import" {
  task_definition = "bcda-${module.platform.env}-api"
}

data "aws_iam_role" "api_execution_import" {
  name = "bcda-${module.platform.env}-api-execution"
}

data "aws_iam_role" "api_task_role_import" {
  name = "bcda-${module.platform.env}-api-task-role"
}

data "aws_lb" "alb_01_import" {
  name = "bcda-api-${module.platform.env}-01"
}

data "aws_lb_listener" "alb_https_01_import" {
  load_balancer_arn = data.aws_lb.alb_01_import.arn
  port              = 443
}

data "aws_lb_target_group" "ecs_api_https_import" {
  name = "bcda-${module.platform.env}-ecs-api-https"
}

###########
# Imports #
###########

import {
  to = module.ecs_api.aws_ssm_parameter.image_tag
  id = "/bcda/${module.platform.env}/nonsensitive/api/image-tag"
}

# --- ECS --- #

import {
  to = module.ecs_api.aws_ecs_service.this
  id = "${data.aws_ecs_cluster.this.cluster_name}/bcda-${module.platform.env}-api"
}

import {
  to = module.ecs_api.aws_ecs_task_definition.this
  id = data.aws_ecs_task_definition.api_import.arn
}

# --- IAM --- #

import {
  to = module.ecs_api.aws_iam_role.execution[0]
  id = "bcda-${module.platform.env}-api-execution"
}

import {
  to = module.ecs_api.aws_iam_role.task
  id = "bcda-${module.platform.env}-api-task-role"
}

import {
  to = module.ecs_api.aws_iam_role_policy.execution[0]
  id = "bcda-${module.platform.env}-api-execution:bcda-${module.platform.env}-api-execution"
}

import {
  to = module.ecs_api.aws_iam_role_policy.task
  id = "bcda-${module.platform.env}-api-task-role:bcda-${module.platform.env}-api-task-policy"
}

# --- Log Groups --- #

import {
  to = module.ecs_api.aws_cloudwatch_log_group.app
  id = "/aws/ecs/fargate/bcda-${module.platform.env}/api"
}

import {
  to = module.ecs_api.aws_cloudwatch_log_group.datadog[0]
  id = "/aws/ecs/fargate/bcda-${module.platform.env}/api/datadog-agent"
}

# --- ALB --- #

import {
  to = aws_lb.alb_01
  id = data.aws_lb.alb_01_import.arn
}

import {
  to = aws_lb_listener.alb_https_01
  id = data.aws_lb_listener.alb_https_01_import.arn
}

import {
  to = aws_lb_target_group.ecs_api_https
  id = data.aws_lb_target_group.ecs_api_https_import.arn
}

# --- Autoscaling --- #

import {
  to = aws_appautoscaling_target.ecs_api_cpu_target
  id = "ecs/service/${data.aws_ecs_cluster.this.cluster_name}/bcda-${module.platform.env}-api/ecs:service:DesiredCount"
}

import {
  to = aws_appautoscaling_policy.ecs_api_cpu_policy
  id = "ecs/service/${data.aws_ecs_cluster.this.cluster_name}/bcda-${module.platform.env}-api/ecs:service:DesiredCount/bcda-${module.platform.env}-api-cpu-auto-scaling"
}

# --- ECS Alarms --- #

import {
  to = module.api-ecs-alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-api-cpu-critical"]
  id = "bcda-${module.platform.env}-api-cpu-critical"
}

import {
  to = module.api-ecs-alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-api-cpu-warn"]
  id = "bcda-${module.platform.env}-api-cpu-warn"
}

import {
  to = module.api-ecs-alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-api-memory-critical"]
  id = "bcda-${module.platform.env}-api-memory-critical"
}

import {
  to = module.api-ecs-alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-api-memory-warn"]
  id = "bcda-${module.platform.env}-api-memory-warn"
}

import {
  to = module.cloudwatch_alarms_elb_http.aws_cloudwatch_metric_alarm.healthy_hosts[0]
  id = "bcda-api-${module.platform.env}-01-elb-no-backend"
}

import {
  to = module.cloudwatch_alarms_elb_http.aws_cloudwatch_metric_alarm.spillover_count[0]
  id = "elb-bcda-api-${module.platform.env}-01-spillover-count"
}

import {
  to = module.cloudwatch_alarms_elb_http.aws_cloudwatch_metric_alarm.surge_queue_exceeded[0]
  id = "elb-bcda-api-${module.platform.env}-01-surge-queue-length"
}
