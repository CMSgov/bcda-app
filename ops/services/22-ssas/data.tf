data "aws_vpc" "main" {
  id = module.platform.vpc_id
}

data "aws_ecs_cluster" "this" {
  cluster_name = "bcda-${module.platform.env}"
}

data "aws_ecr_repository" "ecr_ssas" {
  name = "bcda-ssas"
}

data "aws_security_group" "app_sg" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["bcda-api-${module.platform.env}"]
  }
}

data "aws_security_group" "ssas_alb" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["ssas-alb"]
  }
}

data "aws_acm_certificate" "ssas" {
  domain = local.ssas_domain
}

data "aws_kms_key" "app_config_kms_key" {
  key_id = "alias/bcda-${module.platform.env}-app-config-kms"
}

data "aws_sns_topic" "cloudwatch_alarms_topic" {
  name = "bcda-${module.platform.env}-cloudwatch-alarms"
}

data "aws_ssm_parameter" "params_ssas" {
  for_each        = toset(local.config.parameter_names)
  name            = "/bcda/${module.platform.parent_env}/sensitive/ssas/${each.value}"
  with_decryption = true
}

data "aws_ssm_parameter" "config_bucket" {
  name = "/bcda/${module.platform.env}/sensitive/ssas/CONFIG_BUCKET"
}

data "aws_ssm_parameter" "ssas_aco_ms_admin_cidr_blocks" {
  count = local.is_prod ? 1 : 0
  name  = "/bcda/${module.platform.parent_env}/infra/sensitive/ssas_aco_ms_admin_cidr_blocks"
}

data "aws_ssm_parameter" "ssas_4i_admin_cidr_blocks" {
  count = local.is_prod ? 1 : 0
  name  = "/bcda/${module.platform.parent_env}/infra/sensitive/ssas_4i_admin_cidr_blocks"
}

data "aws_ssm_parameter" "ssas_4i_public_cidr_blocks" {
  count = local.is_prod ? 1 : 0
  name  = "/bcda/${module.platform.parent_env}/infra/sensitive/ssas_4i_public_cidr_blocks"
}

data "aws_ssm_parameter" "ssas_ihp_cidr_blocks" {
  count = local.is_prod ? 1 : 0
  name  = "/bcda/${module.platform.parent_env}/infra/sensitive/ssas_ihp_cidr_blocks"
}
