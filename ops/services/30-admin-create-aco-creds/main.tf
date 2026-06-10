locals {
  app               = "bcda"
  env               = terraform.workspace
  service           = "admin-create-aco-creds"
  full_name         = "${local.app}-${local.env}-${local.service}"
  db_sg_name        = "bcda-${local.env}-db"
  memory_size       = 256
  creds_bucket_name = "bcda-${local.env}-aco-creds-*"
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  name = "alias/bcda-${local.env}-app-config-kms"
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

data "aws_iam_policy_document" "creds_bucket" {
  statement {
    actions   = ["s3:PutObject"]
    resources = ["arn:aws:s3:::${local.creds_bucket_name}"]
  }
}

data "aws_iam_policy_document" "kms_access" {
  statement {
    actions = ["kms:ListAliases"]
    // must be *, see: https://docs.aws.amazon.com/kms/latest/developerguide/alias-access.html#alias-access-view
    resources = ["*"]
  }
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=941672f97adfd8a19ce6533313302c4c74bac7a8"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/30-admin-create-aco-creds"
  service     = local.service
  ssm_root_map = {
    bene-prefs = "/bcda/${local.env}/${local.service}/"
  }
}

module "admin_create_aco_creds_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=945fbd644cc8d239bdf3f3a3a7241fb6066a0f55"

  platform     = module.platform
  architecture = "arm64"

  name        = local.service
  description = "Finds and Creates ACO Credentials for passed in ACO ID and IP addresses"

  handler                = "bootstrap"
  runtime                = "provided.al2023"
  liveness_check_enabled = false

  memory_size = local.memory_size

  function_role_inline_policies = { assume-bucket-role = data.aws_iam_policy_document.creds_bucket.json }

  environment_variables = {
    ENV      = local.env
    APP_NAME = "${local.app}-${local.env}-admin-create-aco-creds"
  }

  ssm_parameter_paths = [
    "/slack/token/workflow-alerts",
    "/bcda/${local.env}/sensitive/api/DATABASE_URL",
    "/bcda/${local.env}/sensitive/api/SSAS_URL",
    "/bcda/${local.env}/sensitive/api/BCDA_SSAS_CLIENT_ID",
    "/bcda/${local.env}/sensitive/api/BCDA_SSAS_SECRET",
    "/bcda/${local.env}/sensitive/api/BCDA_CA_FILE.pem",
    "/bcda/${local.env}/sensitive/aco_creds_bucket"
  ]

  extra_kms_key_arns = [data.aws_kms_alias.bcda_app_config_kms_key.target_key_arn]

  github_actions_repos = ["CMSgov/bcda-app"]
}

# Add a rule to the database security group to allow access from the function
data "aws_security_group" "db" {
  name = local.db_sg_name
}

resource "aws_security_group_rule" "function_access" {
  type        = "ingress"
  from_port   = 5432
  to_port     = 5432
  protocol    = "tcp"
  description = "admin-create-aco-creds function access"

  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = module.admin_create_aco_creds_function.security_group_id
}

import {
  to = module.admin_create_aco_creds_function.aws_cloudwatch_log_group.function
  id = "/aws/lambda/${local.full_name}"
}
