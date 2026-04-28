locals {
  app        = "bcda"
  service    = "api-waf-sync"
  full_name  = "${local.app}-${var.env}-api-waf-sync"
  db_sg_name = "${local.app}-${var.env}-db"
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  name = "alias/bcda-${var.env}-app-config-kms"
}

data "aws_kms_alias" "environment_key" {
  name = "alias/${local.app}-${var.env}"
}

data "aws_rds_cluster" "this" {
  cluster_identifier = "${local.app}-${var.env}-aurora"
}

data "aws_security_groups" "db" {
  tags = {
    Name = "bcda-${var.env}-db"
  }
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = var.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/10-config"
  service     = local.service
}

module "api_waf_sync_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=2874c72ccd4c4821e5e3f77ccf61cf77ed05169f"

  app = local.app
  env = var.env
  architecture = "arm64"

  name        = local.full_name
  description = "Synchronizes the IP whitelist in ${local.app} with the WAF IP Set"

  handler = "bootstrap"
  runtime = "provided.al2023"

  function_role_inline_policies = {
    waf-access = data.aws_iam_policy_document.aws_waf_access.json
  }

  schedule_expression = "cron(0/10 * * * ? *)"

  environment_variables = {
    ENV      = var.env
    APP_NAME = "${local.app}-${var.env}-${local.service}"
    DB_HOST  = "postgres://${data.aws_rds_cluster.this.endpoint}:${data.aws_rds_cluster.this.port}/bcda"
  }

  extra_kms_key_arns = concat(
    [
      data.aws_kms_alias.environment_key.target_key_arn,
      data.aws_kms_alias.bcda_app_config_kms_key.target_key_arn
    ],
  )
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
