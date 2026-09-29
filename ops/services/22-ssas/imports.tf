#############################################
# To be removed post-migration to bcda-app. #
# Handles imports for moving state from ops #
# repo to app repo.                         #
#############################################

################
# Data lookups #
################

data "aws_lb" "ssas_alb_import" {
  name = "bcda-ssas-${module.platform.env}"
}

data "aws_lb_listener" "ssas_alb_public_import" {
  load_balancer_arn = data.aws_lb.ssas_alb_import.arn
  port              = local.config.ports.https_port
}

data "aws_lb_listener" "ssas_alb_admin_import" {
  load_balancer_arn = data.aws_lb.ssas_alb_import.arn
  port              = local.config.ports.admin_port
}

data "aws_lb_target_group" "ecs_ssas_public_import" {
  name = "bcda-${module.platform.env}-ecs-ssas-public"
}

data "aws_lb_target_group" "ecs_ssas_admin_import" {
  name = "bcda-${module.platform.env}-ecs-ssas-admin"
}

data "aws_cloudwatch_log_group" "ecs_ssas_app_import" {
  name = "/aws/ecs/fargate/${data.aws_ecs_cluster.this.cluster_name}/ssas"
}

data "aws_cloudwatch_log_group" "ecs_ssas_datadog_import" {
  name = "/aws/ecs/fargate/${data.aws_ecs_cluster.this.cluster_name}/ssas/datadog-agent"
}

data "aws_ecs_task_definition" "ssas_import" {
  task_definition = "bcda-${module.platform.env}-ssas"
}

data "aws_iam_role" "ssas_execution_import" {
  name = "bcda-${module.platform.env}-ssas-execution"
}

data "aws_iam_role" "ssas_task_role_import" {
  name = "bcda-${module.platform.env}-ssas-task-role"
}

###########
# Imports #
###########

# --- ALB --- #

import {
  to = aws_lb.ssas_alb
  id = data.aws_lb.ssas_alb_import.arn
}

import {
  to = aws_lb_listener.ssas_alb_public
  id = data.aws_lb_listener.ssas_alb_public_import.arn
}

import {
  to = aws_lb_listener.ssas_alb_admin
  id = data.aws_lb_listener.ssas_alb_admin_import.arn
}

import {
  to = aws_lb_target_group.ecs_ssas_public
  id = data.aws_lb_target_group.ecs_ssas_public_import.arn
}

import {
  to = aws_lb_target_group.ecs_ssas_admin
  id = data.aws_lb_target_group.ecs_ssas_admin_import.arn
}

# --- ECS --- #

import {
  to = module.ecs_ssas.aws_ecs_service.this
  id = "${data.aws_ecs_cluster.this.cluster_name}/bcda-${module.platform.env}-ssas"
}

import {
  to = module.ecs_ssas.aws_ecs_task_definition.this
  id = data.aws_ecs_task_definition.ssas_import.arn
}

# --- Autoscaling --- #

import {
  to = aws_appautoscaling_target.ecs_ssas_cpu_target
  id = "ecs/service/${data.aws_ecs_cluster.this.cluster_name}/bcda-${module.platform.env}-ssas/ecs:service:DesiredCount"
}

import {
  to = aws_appautoscaling_policy.ecs_ssas_cpu_policy
  id = "ecs/service/${data.aws_ecs_cluster.this.cluster_name}/bcda-${module.platform.env}-ssas/ecs:service:DesiredCount/bcda-${module.platform.env}-ssas-cpu-auto-scaling"
}

# --- CloudWatch Log Groups --- #

import {
  to = module.ecs_ssas.aws_cloudwatch_log_group.app
  id = data.aws_cloudwatch_log_group.ecs_ssas_app_import.name
}

import {
  to = module.ecs_ssas.aws_cloudwatch_log_group.datadog[0]
  id = data.aws_cloudwatch_log_group.ecs_ssas_datadog_import.name
}

# --- ECS Alarms --- #

import {
  to = module.ssas_ecs_alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-ssas-cpu-critical"]
  id = "bcda-${module.platform.env}-ssas-cpu-critical"
}

import {
  to = module.ssas_ecs_alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-ssas-cpu-warn"]
  id = "bcda-${module.platform.env}-ssas-cpu-warn"
}

import {
  to = module.ssas_ecs_alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-ssas-memory-critical"]
  id = "bcda-${module.platform.env}-ssas-memory-critical"
}

import {
  to = module.ssas_ecs_alarms.aws_cloudwatch_metric_alarm.ecs_alarms["bcda-${module.platform.env}-ssas-memory-warn"]
  id = "bcda-${module.platform.env}-ssas-memory-warn"
}

# --- IAM --- #

import {
  to = module.ecs_ssas.aws_iam_role.execution[0]
  id = "bcda-${module.platform.env}-ssas-execution"
}

import {
  to = module.ecs_ssas.aws_iam_role.task
  id = "bcda-${module.platform.env}-ssas-task-role"
}

import {
  to = module.ecs_ssas.aws_iam_role_policy.execution[0]
  id = "bcda-${module.platform.env}-ssas-execution:bcda-${module.platform.env}-ssas-execution"
}

import {
  to = module.ecs_ssas.aws_iam_role_policy.task
  id = "bcda-${module.platform.env}-ssas-task-role:bcda-${module.platform.env}-ssas-task-policy"
}
