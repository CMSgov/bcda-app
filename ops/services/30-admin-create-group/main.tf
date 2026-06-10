locals {
  app         = "bcda"
  env         = terraform.workspace
  full_name   = "${local.app}-${local.env}-admin-create-group"
  db_sg_name  = "bcda-${local.env}-db"
  memory_size = 2048
  service     = "admin-create-group"
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  name = "alias/bcda-${local.env}-app-config-kms"
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/30-admin-create-group"
  service     = local.service
}

module "admin_create_group_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=945fbd644cc8d239bdf3f3a3a7241fb6066a0f55"

  platform     = module.platform
  architecture = "arm64"

  name        = local.service
  description = "Creates a group for the supplied CMS ID."

  handler                = "bootstrap"
  runtime                = "provided.al2023"
  liveness_check_enabled = false

  memory_size = local.memory_size

  environment_variables = {
    ENV      = local.env
    APP_NAME = "${local.app}-${local.env}-admin-create-group"
  }

  ssm_parameter_paths = [
    "/slack/token/workflow-alerts",
    "/bcda/${local.env}/sensitive/api/DATABASE_URL",
    "/bcda/${local.env}/sensitive/api/SSAS_URL",
    "/bcda/${local.env}/sensitive/api/BCDA_SSAS_CLIENT_ID",
    "/bcda/${local.env}/sensitive/api/BCDA_SSAS_SECRET",
    "/bcda/${local.env}/sensitive/api/BCDA_CA_FILE.pem"
  ]

  extra_kms_key_arns = [data.aws_kms_alias.bcda_app_config_kms_key.target_key_arn]

  github_actions_repos = ["CMSgov/bcda-app"]
}

# TODO: Delete after deploying BCDA-10031
resource "aws_security_group_rule" "function_access" {
  count       = 0
  type        = "ingress"
  from_port   = 5432
  to_port     = 5432
  protocol    = "tcp"
  description = "admin-create-group function access"

  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = module.admin_create_group_function.security_group_id
}

import {
  to = module.admin_create_group_function.aws_cloudwatch_log_group.function
  id = "/aws/lambda/${local.full_name}"
}
