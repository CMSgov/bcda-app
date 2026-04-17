locals {
  app         = "bcda"
  full_name   = "${local.app}-${var.env}-admin-create-group"
  db_sg_name  = "bcda-${var.env}-db"
  memory_size = 2048
  service     = "admin-create-group"
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  name = "alias/bcda-${var.env}-app-config-kms"
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = var.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/10-config"
  service     = local.service
}

module "admin_create_group_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=f4c14d47cc20e7f6de9112d7155af1213c9bca5a"

  app = local.app
  env = var.env

  name        = local.full_name
  description = "Creates a group for the supplied CMS ID."

  handler = "bootstrap"
  runtime = "provided.al2023"

  memory_size = local.memory_size

  environment_variables = {
    ENV      = var.env
    APP_NAME = "${local.app}-${var.env}-admin-create-group"
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
  description = "admin-create-group function access"

  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = module.admin_create_group_function.security_group_id
}
