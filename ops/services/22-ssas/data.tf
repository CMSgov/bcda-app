data "aws_vpc" "main" {
  id = module.platform.vpc_id
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
    values = "ssas-alb"
  }
}

data "aws_acm_certificate" "ssas" {
  domain = local.ssas_domain
}

data "aws_ssm_parameter" "params_ssas" {
  for_each        = toset(local.config.parameter_names)
  name            = "/bcda/${module.platform.parent_env}/sensitive/ssas/${each.value}"
  with_decryption = true
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
