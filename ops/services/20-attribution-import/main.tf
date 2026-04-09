locals {
  service            = "attribution-import"
  full_name          = "${local.app}-${var.env}-attribution-import"
  db_sg_name         = "${local.app}-${var.env}-db"
  extra_kms_key_arns = local.app == "bcda" ? [data.aws_kms_alias.bcda_app_config_kms_key[0].target_key_arn] : []
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  count = local.app == "bcda" ? 1 : 0
  name  = "alias/bcda-${var.env}-app-config-kms"
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = "bcda"
  env         = var.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/10-config"
  service     = local.service
  ssm_root_map = {
    attribution-import = "/bcda/${var.env}/${local.service}/"
  }
}


data "aws_iam_policy_document" "assume_bucket_role" {
  statement {
    actions = ["sts:AssumeRole"]
    resources = [
      module.platform.ssm.attribution-import.misp-eft-role_arn.value
    ]
  }
}

module "attribution_import_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=f4c14d47cc20e7f6de9112d7155af1213c9bca5a"

  app = local.app
  env = var.env

  name        = local.full_name
  description = "Ingests the most recent attribution from BFD"

  handler = "bootstrap"
  runtime = "provided.al2023"

  memory_size = 2048

  function_role_inline_policies = {
    assume-bucket-role = data.aws_iam_policy_document.assume_bucket_role.json
  }

  environment_variables = {
    ENV      = var.env
    APP_NAME = "${local.app}-${var.env}-attribution-import"
  }
  extra_kms_key_arns = local.extra_kms_key_arns
}

# Set up queue for receiving messages when a file is added to the bucket
module "attribution_import_queue" {
  source = "github.com/CMSgov/cdap//terraform/modules/queue?ref=f4c14d47cc20e7f6de9112d7155af1213c9bca5a"

  app = local.app
  env = var.env

  name = local.full_name

  function_name    = module.attribution_import_function.name
  policy_documents = [data.aws_iam_policy_document.sns_send_message.json]
}

data "aws_iam_policy_document" "sns_send_message" {
  statement {
    sid     = "SnsSendMessage"
    actions = ["sqs:SendMessage"]

    principals {
      type        = "Service"
      identifiers = ["sns.amazonaws.com"]
    }

    resources = [module.attribution_import_queue.arn]

    condition {
      test     = "ArnEquals"
      variable = "aws:SourceArn"
      values   = [aws_sns_topic.this.arn]
    }
  }
}

module "attribution-import_file_bucket" {
  source = "github.com/CMSgov/cdap//terraform/modules/bucket?ref=787224b"

  app           = local.app
  env           = var.env
  name          = "${local.app}-${var.env}-${local.service}-file"
  ssm_parameter = "/${local.app}/${var.env}/${local.service}/nonsensitive/file_bucket_name"
}

data "aws_iam_policy_document" "topic" {
  statement {
    principals {
      type        = "Service"
      identifiers = ["s3.amazonaws.com"]
    }

    actions   = ["SNS:Publish"]
    resources = [aws_sns_topic.this.arn]

    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values   = [module.attribution-import_file_bucket.arn]
    }
  }
}

resource "aws_sns_topic" "this" {
  name              = "${local.full_name}-topic"
  kms_master_key_id = "alias/bcda-${var.env}"
}

resource "aws_sns_topic_policy" "this" {
  arn = aws_sns_topic.this.arn

  policy = data.aws_iam_policy_document.topic.json
}

resource "aws_sns_topic_subscription" "this" {
  endpoint  = module.attribution_import_queue.arn
  protocol  = "sqs"
  topic_arn = aws_sns_topic.this.arn
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
  description = "attribution-import function access"

  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = module.attribution_import_function.security_group_id
}
