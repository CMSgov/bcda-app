data "aws_vpc" "main" {
  id = module.platform.vpc_id
}

data "aws_ecs_cluster" "this" {
  cluster_name = "bcda-${module.platform.env}"
}

data "aws_ecr_repository" "ecr_worker" {
  name = "bcda-worker"
}

data "aws_security_group" "worker_sg" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["bcda-worker-${module.platform.env}"]
  }
}

data "aws_security_group" "sg_alb_01" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = [local.alb_sg_name]
  }
}

data "aws_security_group" "db" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["bcda-${module.platform.env}-db"]
  }
}

data "aws_kms_key" "app_config_kms_key" {
  key_id = "alias/bcda-${module.platform.env}-app-config-kms"
}


data "aws_sns_topic" "cloudwatch_alarms_topic" {
  name = "bcda-${module.platform.env}-cloudwatch-alarms"
}

data "aws_ssm_parameter" "params_worker" {
  for_each        = toset(local.config.parameter_names)
  name            = "/bcda/${module.platform.env}/sensitive/worker/${each.value}"
  with_decryption = true
}

data "aws_efs_file_system" "main" {
  tags = {
    Name = "bcda-${module.platform.env}-efs"
  }
}

data "aws_ssm_parameter" "efs_access_point_worker" {
  name = "/bcda/${module.platform.env}/infra/sensitive/efs_access_point_worker"
}
