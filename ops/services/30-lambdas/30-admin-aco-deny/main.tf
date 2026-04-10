locals {
  full_name   = "${var.app}-${var.env}-admin-aco-deny"
  db_sg_name  = "bcda-${var.env}-db"
  memory_size = 256
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  name = "alias/bcda-${var.env}-app-config-kms"
}

module "admin_aco_deny_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=2f21bef1de2d6fd1326e7106699250f610f4c66c"

  app = var.app
  env = var.env

  name        = local.full_name
  description = "Denies access to BCDA for supplied ACO IDs"

  handler = "bootstrap"
  runtime = "provided.al2"

  memory_size = local.memory_size

  environment_variables = {
    ENV      = var.env
    APP_NAME = "${var.app}-${var.env}-admin-aco-deny"
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
  description = "admin-aco-deny function access"

  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = module.admin_aco_deny_function.security_group_id
}
