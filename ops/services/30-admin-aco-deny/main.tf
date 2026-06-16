locals {
  app         = "bcda"
  env         = terraform.workspace
  full_name   = "${local.app}-${local.env}-admin-aco-deny"
  db_sg_name  = "bcda-${local.env}-db"
  memory_size = 256
  service     = "admin-aco-deny"
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  name = "alias/bcda-${local.env}-app-config-kms"
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/30-admin-aco-deny"
  service     = local.service
}

module "admin_aco_deny_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=945fbd644cc8d239bdf3f3a3a7241fb6066a0f55"

  platform     = module.platform
  architecture = "arm64"

  name        = local.service
  description = "Denies access to BCDA for supplied ACO IDs"

  handler                = "bootstrap"
  runtime                = "provided.al2023"
  liveness_check_enabled = false

  memory_size = local.memory_size

  environment_variables = {
    ENV      = local.env
    APP_NAME = "${local.app}-${local.env}-admin-aco-deny"
  }

  ssm_parameter_paths = [
    "/bcda/${local.env}/sensitive/api/DATABASE_URL",
    "/slack/token/workflow-alerts"
  ]

  extra_kms_key_arns = [data.aws_kms_alias.bcda_app_config_kms_key.target_key_arn]

  github_actions_repos = ["CMSgov/bcda-app"]
}
