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

  is_prod         = contains(["prod", "sandbox"], local.parent_env)
  alarm_topic_arn = local.is_prod ? data.aws_sns_topic.cloudwatch_critical_alarms_topic : data.aws_sns_topic.cloudwatch_alarms_topic

  app_sg_name = "bcda-api-${module.platform.env}"
  alb_sg_name = "${local.config.alb_sg_name_prefix}-${module.platform.env}"

  web_domain_map = {
    for env in local.established_envs :
    env => env == "prod" ? "api.bcda.cms.gov" : "${env}.bcda.cms.gov"
  }

  web_domain = lookup(local.web_domain_map, module.platform.env, "${module.platform.env}.bcda.cms.gov")
}

module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = "bcda"
  env         = terraform.workspace
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/${basename(abspath(path.module))}"
  service     = local.service
}

##############
# Networking #
##############

resource "aws_lb" "alb_01" {
  name                             = "bcda-api-${module.platform.env}-01"
  internal                         = false
  load_balancer_type               = "application"
  enable_deletion_protection       = true
  drop_invalid_header_fields       = true
  enable_cross_zone_load_balancing = true
  idle_timeout                     = local.config.health_check.idle_timeout

  security_groups = [
    data.aws_security_group.sg_alb_01.id,
    data.aws_security_group.zscaler_public.id,
  ]

  subnets = toset(keys(module.platform.public_subnets))

  access_logs {
    bucket  = "cms-cloud-${module.platform.aws_caller_identity.account_id}-${module.platform.primary_region.name}"
    enabled = true
  }

  tags = {
    Name = "bcda-${module.platform.env}-app-01"
  }
}

resource "aws_lb_target_group" "ecs_api_https" {
  name                 = "bcda-${module.platform.env}-ecs-api-https"
  port                 = local.config.ports.app
  protocol             = local.config.health_check.protocol
  vpc_id               = module.platform.vpc_id
  deregistration_delay = local.config.health_check.deregistration_delay
  target_type          = "ip"

  health_check {
    interval            = local.config.health_check.interval
    path                = local.config.health_check.path
    port                = local.config.ports.app
    protocol            = local.config.health_check.protocol
    healthy_threshold   = local.config.health_check.healthy_threshold
    unhealthy_threshold = local.config.health_check.unhealthy_threshold
    timeout             = local.config.health_check.timeout
  }
}

resource "aws_lb_listener" "alb_https_01" {
  load_balancer_arn = aws_lb.alb_01.arn
  port              = local.config.ports.https
  protocol          = local.config.health_check.protocol
  ssl_policy        = local.config.ssl_policy
  certificate_arn   = data.aws_acm_certificate.web.arn

  default_action {
    type = "forward"

    forward {
      target_group {
        arn = aws_lb_target_group.ecs_api_https.arn
      }
    }
  }
}

###########
# Service #
###########

module "ecs_api" {
  source                            = "github.com/CMSgov/cdap/terraform/modules/service?ref=e8af7a286d7e7637e41de27adf00af2d0d58f4e7"
  service_name_override             = local.service
  platform                          = module.platform
  cluster_arn                       = data.aws_ecs_cluster.this.arn
  image                             = "${data.aws_ecr_repository.ecr_api.repository_url}:${var.image_tag}"
  cpu                               = local.config.ecs.cpu
  memory                            = local.config.ecs.mem
  desired_count                     = local.config.scaling.min
  security_groups                   = [data.aws_security_group.app_sg.id]
  cpu_architecture                  = "ARM64"
  enable_datadog_synthetics_ingress = true
  additional_task_role_policies     = { bootstrap_policy = tostring(aws_iam_policy.api_task.arn) }
  port_mappings                     = [{ containerPort = local.config.ports.app }]

  container_environment = [
    { name = "LOG_TO_STD_OUT", value = "true" }
  ]

  container_secrets = [
    for param in data.aws_ssm_parameter.params_api : {
      name      = element(split("/", param.name), -1)
      valueFrom = param.arn
    }
  ]

  load_balancers = [{
    target_group_arn = aws_lb_target_group.ecs_api_https.arn
    container_name   = local.service
    container_port   = local.config.ports.app
  }]

  mount_points = [
    {
      containerPath = "/var/efs"
      sourceVolume  = "efs"
    },
    {
      containerPath = "/etc/sv/api/env"
      sourceVolume  = "api_config"
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
          access_point_id = data.aws_ssm_parameter.efs_access_point_api.value
        }
      }
    },
    { name = "api_config" }
  ]
}

###############
# Autoscaling #
###############

resource "aws_appautoscaling_target" "ecs_api_cpu_target" {
  min_capacity       = local.config.scaling.min
  max_capacity       = local.config.scaling.max
  resource_id        = "service/${data.aws_ecs_cluster.this.cluster_name}/${module.ecs_api.service.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "ecs_ssas_cpu_policy" {
  name               = "bcda-${module.platform.env}-api-cpu-auto-scaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs_api_cpu_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_api_cpu_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_api_cpu_target.service_namespace

  target_tracking_scaling_policy_configuration {
    target_value       = local.config.scaling.target_value
    scale_in_cooldown  = local.config.scaling.scale_in_cooldown
    scale_out_cooldown = local.config.scaling.scale_out_cooldown
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}

##########
# Alarms #
##########

module "api-ecs-alarms" {
  source = "../modules/ecs_alarms"

  service_name           = module.ecs_api.service.name
  cluster_name           = data.aws_ecs_cluster.this.cluster_name
  alarm_notification_arn = data.aws_sns_topic.cloudwatch_alarms_topic.arn
  ok_notification_arn    = data.aws_sns_topic.cloudwatch_alarms_topic.arn
  alarms = [
    {
      alarm_name        = "${module.ecs_api.service.name}-cpu-critical"
      alarm_description = "CRITICAL - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_api.service.name}-cpu-warn"
      alarm_description = "WARN - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    },
    {
      alarm_name        = "${module.ecs_api.service.name}-memory-critical"
      alarm_description = "CRITICAL - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_api.service.name}-memory-warn"
      alarm_description = "WARN - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    }
  ]
}

module "cloudwatch_alarms_elb_http" {
  source = "../modules/elb_http_alarms"

  app                         = "bcda"
  env                         = module.platform.env
  vpc_name                    = module.platform.vpc_id
  cloudwatch_notification_arn = local.alarm_topic_arn
  load_balancer_name          = aws_lb.alb_01.name

  alarm_backend_4xx_enable      = local.config.alarms.backend_4xx
  alarm_backend_5xx_enable      = local.config.alarms.backend_5xx
  alarm_elb_5xx_enable          = local.config.alarms.elb_5xx
  alarm_elb_high_latency_enable = local.config.alarms.elb_high_latency

  alarm_elb_no_backend_treat_missing_data   = "notBreaching"
  alarm_elb_high_latency_treat_missing_data = "notBreaching"
}
