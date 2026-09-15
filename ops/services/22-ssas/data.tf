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

data "aws_acm_certificate" "ssas" {
  domain = local.ssas_domain
}

data "aws_ssm_parameter" "params_ssas" {
  for_each        = toset(local.config.parameter_names)
  name            = "/bcda/${module.platform.parent_env}/sensitive/ssas/${each.value}"
  with_decryption = true
}
