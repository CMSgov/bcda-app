locals {
  defaults        = yamldecode(file("${path.module}/config/defaults.yml"))
  env_config_path = "${path.module}/config/${module.platform.parent_env}.yml"
  env_config      = fileexists(local.env_config_path) ? yamldecode(file(local.env_config_path)) : {}

  config = {
    for key in distinct(concat(keys(local.defaults), keys(local.env_config))) :
    key => try(
      # Attempt map merge (works if both values are map/object-typed)
      merge(
        lookup(local.defaults, key, {}),
        lookup(local.env_config, key, {})
      ),
      # Fallback to scalar: env wins, then default
      lookup(local.env_config, key, lookup(local.defaults, key, null))
    )
  }

  default_tags = module.platform.default_tags
  service      = replace(basename(abspath(path.module)), "/^[0-9]+-/", "")

  efs_access_point_id = data.aws_efs_access_points.main.access_point_ids[0]
  alb_sg_name         = "${local.config.alb_sg_name_prefix}-${module.platform.env}"

  # Autoscaling lookups for dynamic policy
  worker_steps = [
    for step in local.config.worker.scaling_steps : {
      scaling_adjustment          = step.scaling_adjustment_offset == "max" ? local.config.scaling.max : local.config.scaling.min + step.scaling_adjustment_offset
      metric_interval_lower_bound = lookup(step, "metric_interval_lower_bound", null)
      metric_interval_upper_bound = lookup(step, "metric_interval_upper_bound", null)
    }
  ]

  # Alarm lookups
  worker_alarms = [
    for alarm in local.config.alarms : merge(alarm, {
      alarm_name = "${module.ecs_worker.service.name}-${alarm.alarm_name_suffix}"
    })
  ]
}

module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = "bcda"
  env         = terraform.workspace
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/${basename(abspath(path.module))}"
  service     = local.service
}

###########
# Service #
###########

module "ecs_worker" {
  source                        = "github.com/CMSgov/cdap/terraform/modules/service?ref=e8af7a286d7e7637e41de27adf00af2d0d58f4e7"
  service_name_override         = "worker"
  platform                      = module.platform
  cluster_arn                   = data.aws_ecs_cluster.this.arn
  image                         = "${data.aws_ecr_repository.ecr_worker.repository_url}:${var.image_tag}"
  cpu                           = local.config.ecs.cpu
  memory                        = local.config.ecs.mem
  desired_count                 = local.config.scaling.min
  security_groups               = [data.aws_security_group.worker_sg.id]
  additional_task_role_policies = { bootstrap_policy = tostring(aws_iam_policy.instance.arn) }
  cpu_architecture              = "ARM64"

  container_environment = [
    { name = "FHIR_TEMP_DIR", value = "/home/bcda/FHIR_TEMP_DIR" }
  ]

  container_secrets = [
    for param in data.aws_ssm_parameter.params_worker : {
      name      = element(split("/", param.name), -1)
      valueFrom = param.arn
    }
  ]

  health_check = {
    command     = ["CMD-SHELL", "bcdaworker health >> /proc/1/fd/1 2>&1  || exit 1"],
    interval    = local.config.health_check.interval,
    retries     = local.config.health_check.unhealthy_threshold,
    startPeriod = local.config.health_check.start_period,
    timeout     = local.config.health_check.timeout
  }

  mount_points = [
    {
      containerPath = "/var/efs",
      sourceVolume  = "efs"
    },
    {
      containerPath = "/etc/sv/worker/env"
      sourceVolume  = "worker_config"
    },
    {
      containerPath = "/home/bcda/FHIR_TEMP_DIR",
      sourceVolume  = "fhir_tmp"
    }
  ]

  volumes = [
    {
      name = "efs"
      efs_volume_configuration = {
        file_system_id     = data.aws_efs_file_system.main.id
        root_directory     = "/"
        transit_encryption = "ENABLED"
        authorization_config = {
          access_point_id = local.efs_access_point_id
        }
      }
    },
    { name = "worker_config" },
    { name = "fhir_tmp" }
  ]
}

###############
# Autoscaling #
###############

resource "aws_appautoscaling_target" "ecs_worker_target" {
  min_capacity       = local.config.scaling.min
  max_capacity       = local.config.scaling.max
  resource_id        = "service/${data.ecs_cluster.this.name}/${module.ecs_worker.service.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "ecs_worker_policy" {
  name               = "bcda-${module.platform.env}-worker-job-count-scaleup"
  policy_type        = "StepScaling"
  resource_id        = aws_appautoscaling_target.ecs_worker_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_worker_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_worker_target.service_namespace

  step_scaling_policy_configuration {
    adjustment_type         = "ExactCapacity"
    cooldown                = local.config.scaling.cooldown
    metric_aggregation_type = "Maximum"

    dynamic "step_adjustment" {
      for_each = local.worker_steps
      content {
        scaling_adjustment          = step_adjustment.value.scaling_adjustment
        metric_interval_lower_bound = lookup(step_adjustment.value, "metric_interval_lower_bound", null)
        metric_interval_upper_bound = lookup(step_adjustment.value, "metric_interval_upper_bound", null)
      }
    }
  }
}

##########
# Alarms #
##########

resource "aws_cloudwatch_metric_alarm" "ecs_worker_job_count" {
  alarm_name          = "bcda-${module.platform.env}-worker-job-count"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "JobQueueCount"
  namespace           = "BCDA"
  period              = "120"
  statistic           = "Average"
  threshold           = "0"

  dimensions = {
    Environment = module.platform.env
  }

  alarm_description = "Queued job count for workers"
  alarm_actions     = [aws_appautoscaling_policy.ecs_worker_policy.arn]
}

module "worker_ecs_alarms" {
  source = "../../modules/ecs_alarms"

  service_name = module.ecs_worker.service.name
  cluster_name = data.aws_ecs_cluster.this.cluster_name

  alarm_notification_arn = data.aws_sns_topic.cloudwatch_alarms_topic.arn
  ok_notification_arn    = data.aws_sns_topic.cloudwatch_alarms_topic.arn

  alarms = local.worker_alarms
}
