locals {
  app        = "bcda"
  env        = terraform.workspace
  service    = "api-waf-sync"
  full_name  = "${local.app}-${local.env}-api-waf-sync"
  db_sg_name = "${local.app}-${local.env}-db"
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  name = "alias/bcda-${local.env}-app-config-kms"
}

data "aws_kms_alias" "environment_key" {
  name = "alias/${local.app}-${local.env}"
}

data "aws_rds_cluster" "this" {
  cluster_identifier = "${local.app}-${local.env}-aurora"
}

data "aws_security_groups" "db" {
  tags = {
    Name = "bcda-${local.env}-db"
  }
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/30-api-waf-sync"
  service     = local.service
}

module "api_waf_sync_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=945fbd644cc8d239bdf3f3a3a7241fb6066a0f55"

  platform     = module.platform
  architecture = "arm64"

  name        = local.service
  description = "Synchronizes the IP whitelist in ${local.app} with the WAF IP Set"

  handler                = "bootstrap"
  runtime                = "provided.al2023"
  liveness_check_enabled = false

  function_role_inline_policies = {
    waf-access = data.aws_iam_policy_document.aws_waf_access.json
  }

  schedule_expression = "cron(0/10 * * * ? *)"

  environment_variables = {
    ENV      = local.env
    APP_NAME = "${local.app}-${local.env}-${local.service}"
    DB_HOST  = "postgres://${data.aws_rds_cluster.this.endpoint}:${data.aws_rds_cluster.this.port}/bcda"
  }

  ssm_parameter_paths = [
    "/bcda/${local.env}/sensitive/api/DATABASE_URL"
  ]

  extra_kms_key_arns = concat(
    [
      data.aws_kms_alias.environment_key.target_key_arn,
      data.aws_kms_alias.bcda_app_config_kms_key.target_key_arn
    ],
  )

  github_actions_repos = ["CMSgov/bcda-ssas-app"]

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
  description = "api-waf-sync function access"

  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = module.api_waf_sync_function.security_group_id
}

# Because we inline policies, we cannot just link to aws:policy/AWSWAFFullAccess
data "aws_iam_policy_document" "aws_waf_access" {
  statement {
    effect    = "Allow"
    resources = ["*"]

    actions = [
      "wafv2:ListIpSets",
      "wafv2:GetIpSet",
      "wafv2:UpdateIpSet",
    ]
  }
}

import {
  to = module.api_waf_sync_function.aws_cloudwatch_log_group.function
  id = "/aws/lambda/${local.full_name}"
}
