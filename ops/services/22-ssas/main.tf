locals {
  defaults   = yamldecode(file("config/defaults.yml"))
  env_config = yamldecode(file("config/${module.platform.parent_env}.yml"))

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
  is_prod      = contains(["prod", "sandbox"], local.parent_env)

  # Network
  ssas_domain = "ssas.${module.platform.env}.bcda.cms.gov"

  # CIDR Blocks
  app_cidr_block        = data.aws_vpc.main.cidr_block
  management_cidr_block = module.platform.platform_cidr
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

resource "aws_lb" "ssas_alb" {
  name               = "bcda-ssas-${module.platform.env}"
  internal           = true
  load_balancer_type = "application"
  idle_timeout       = 60

  security_groups = [
    data.aws_security_group.ssas_alb.id,
    module.platform.security_groups["remote-management"].id,
    module.platform.security_groups["zscaler-private"].id,
    module.platform.security_groups["zscaler-public"].id,
  ]

  subnets = module.platform.private_subnets[*].id

  access_logs {
    bucket  = "cms-cloud-${module.platform.account_id}-${module.platform.primary_region.name}"
    enabled = true
  }
}

resource "aws_lb_target_group" "ecs_ssas_admin" {
  name     = "bcda-${module.platform.env}-ssas-admin"
  port     = local.config.ports.ssas_admin_port
  protocol = "HTTPS"
  vpc_id   = module.platform.vpc_id

  health_check {
    path                = local.config.health_check.path
    interval            = local.config.health_check.interval
    timeout             = local.config.health_check.timeout
    healthy_threshold   = local.config.health_check.healthy_threshold
    unhealthy_threshold = local.config.health_check.unhealthy_threshold
  }
}

resource "aws_lb_target_group" "ecs_ssas_public" {
  name     = "bcda-${module.platform.env}-ssas-public"
  port     = local.config.ports.ssas_public_port
  protocol = "HTTPS"
  vpc_id   = module.platform.vpc_id

  health_check {
    path                = local.config.health_check.path
    interval            = local.config.health_check.interval
    timeout             = local.config.health_check.timeout
    healthy_threshold   = local.config.health_check.healthy_threshold
    unhealthy_threshold = local.config.health_check.unhealthy_threshold
  }
}

resource "aws_lb_listener" "ssas_alb_admin" {
  load_balancer_arn = aws_lb.ssas_alb.arn
  port              = local.config.ports.admin_port
  protocol          = "HTTPS"
  ssl_policy        = local.config.ssl_policy
  certificate_arn   = data.aws_acm_certificate.ssas.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ecs_ssas_admin.arn
  }
}

resource "aws_lb_listener" "ssas_alb_public" {
  load_balancer_arn = aws_lb.ssas_alb.arn
  port              = local.config.ports.https_port
  protocol          = "HTTPS"
  ssl_policy        = local.config.ssl_policy
  certificate_arn   = data.aws_acm_certificate.ssas.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.ecs_ssas_public.arn
  }
}

###########
# Service #
###########

module "ecs_ssas" {
  source                        = "github.com/CMSgov/cdap/terraform/modules/service?ref=e8af7a286d7e7637e41de27adf00af2d0d58f4e7"
  service_name_override         = local.service
  platform                      = module.platform
  cluster_arn                   = module.ecs_cluster.this.arn
  image                         = "${data.aws_ecr_repository.ecr_ssas.repository_url}:${var.image_tag}"
  cpu                           = local.config.ecs.cpu
  memory                        = local.config.ecs.mem
  desired_count                 = local.config.scaling.min
  port_mappings                 = [{ containerPort = local.config.ports.ssas_public_port }, { containerPort = local.config.ports.ssas_admin_port }]
  security_groups               = [aws_security_group.app_sg.id]
  additional_task_role_policies = { bootstrap_policy = tostring(aws_iam_policy.ssas_task.arn) }
  cpu_architecture              = "ARM64"

  container_environment = [
    { name = "LOG_TO_STD_OUT", value = "true" }
  ]

  container_secrets = [
    for param in data.aws_ssm_parameter.params_ssas : {
      name      = element(split("/", param.name), -1)
      valueFrom = param.arn
    }
  ]

  load_balancers = [
    {
      target_group_arn = aws_lb_target_group.ecs_ssas_public.arn
      container_name   = local.service
      container_port   = local.config.ports.ssas_public_port
    },
    {
      target_group_arn = aws_lb_target_group.ecs_ssas_admin.arn
      container_name   = local.service
      container_port   = local.config.ports.ssas_admin_port
    }
  ]

  mount_points = [
    {
      containerPath = "/etc/sv/ssas/env"
      sourceVolume  = "ssas_config"
    }
  ]

  volumes = [{ name = "ssas_config" }]
}

###############
# Autoscaling #
###############

resource "aws_appautoscaling_target" "ecs_ssas_cpu_target" {
  max_capacity       = local.config.scaling.max
  min_capacity       = local.config.scaling.min
  resource_id        = "service/${data.terraform_remote_state.cluster.outputs.cluster_name}/${module.ecs_ssas.service.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "ecs_ssas_cpu_policy" {
  name               = "bcda-${module.platform.env}-ssas-cpu-scaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs_ssas_cpu_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_ssas_cpu_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_ssas_cpu_target.service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
    target_value = 60
  }
}

##########
# Alarms #
##########

module "ssas_ecs_alarms" {
  source = "../../modules/ecs_alarms"

  service_name = module.ecs_ssas.service.name
  cluster_name = module.ecs_cluster.this.name

  alarm_notification_arn = aws_sns_topic.cloudwatch_alarms_topic.arn
  ok_notification_arn    = aws_sns_topic.cloudwatch_alarms_topic.arn

  alarms = [
    {
      alarm_name        = "${module.ecs_ssas.service.name}-cpu-critical"
      alarm_description = "CRITICAL - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_ssas.service.name}-cpu-warn"
      alarm_description = "WARN - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    },
    {
      alarm_name        = "${module.ecs_ssas.service.name}-memory-critical"
      alarm_description = "CRITICAL - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_ssas.service.name}-memory-warn"
      alarm_description = "WARN - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    }
  ]
}
