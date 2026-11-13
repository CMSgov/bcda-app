data "aws_vpc" "main" {
  id = module.platform.vpc_id
}

data "aws_ecs_cluster" "this" {
  cluster_name = "bcda-${module.platform.env}"
}

data "aws_ecr_repository" "ecr_api" {
  name = "bcda-api"
}

data "aws_efs_file_system" "main" {
  tags = {
    Name = "bcda-${module.platform.env}-efs"
  }
}

data "aws_ssm_parameter" "efs_access_point_api" {
  name = "/bcda/${module.platform.env}/infra/sensitive/efs_access_point_api"
}

data "aws_kms_key" "app_config_kms_key" {
  key_id = "alias/bcda-${module.platform.env}-app-config-kms"
}

data "aws_sns_topic" "cloudwatch_alarms_topic" {
  name = "bcda-${module.platform.env}-cloudwatch-alarms"
}

data "aws_sns_topic" "cloudwatch_critical_alarms_topic" {
  count = local.is_prod ? 1 : 0
  name  = "bcda-${module.platform.env}-cloudwatch-critical-alarms"
}

data "aws_acm_certificate" "web" {
  domain      = local.web_domain
  statuses    = ["ISSUED"]
  most_recent = true
}

data "aws_security_group" "app_sg" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["bcda-api-${module.platform.env}"]
  }
}

data "aws_security_group" "sg_alb_01" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = [local.alb_sg_name]
  }
}

data "aws_security_group" "ssas_alb" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["ssas-alb"]
  }
}

data "aws_security_group" "worker_sg" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["bcda-worker-${module.platform.env}"]
  }
}

data "aws_security_group" "db" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["bcda-${module.platform.env}-db"]
  }
}

data "aws_security_group" "zscaler_public" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["zscaler-public"]
  }
}

data "aws_ssm_parameter" "params_api" {
  for_each        = toset(local.config.parameter_names)
  name            = "/bcda/${module.platform.env}/sensitive/api/${each.value}"
  with_decryption = true
}
