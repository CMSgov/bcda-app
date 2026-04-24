locals {
  service           = "admin-create-aco-creds"
  full_name         = "${var.app}-${var.env}-${local.service}"
  db_sg_name        = "bcda-${var.env}-db"
  memory_size       = 256
  creds_bucket_name = "bcda-${var.env}-aco-creds-*"
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  name = "alias/bcda-${var.env}-app-config-kms"
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
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = var.app
  env         = var.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/10-config"
  service     = local.service
  ssm_root_map = {
    bene-prefs = "/bcda/${var.env}/${local.service}/"
  }
}

module "admin_create_aco_creds_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=2874c72ccd4c4821e5e3f77ccf61cf77ed05169f"

  app = var.app
  env = var.env
  architecture = "arm64"

  name        = local.full_name
  description = "Finds and Creates ACO Credentials for passed in ACO ID and IP addresses"

  handler = "bootstrap"
  runtime = "provided.al2"

  memory_size = local.memory_size

  function_role_inline_policies = { assume-bucket-role = data.aws_iam_policy_document.creds_bucket.json }

  environment_variables = {
    ENV      = var.env
    APP_NAME = "${var.app}-${var.env}-admin-create-aco-creds"
  }

  extra_kms_key_arns = [data.aws_kms_alias.bcda_app_config_kms_key.target_key_arn]
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
