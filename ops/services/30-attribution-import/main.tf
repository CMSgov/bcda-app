locals {
  app        = "bcda"
  service    = "attribution-import"
  full_name  = "${local.app}-${var.env}-attribution-import"
  db_sg_name = "${local.app}-${var.env}-db"
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = "bcda"
  env         = var.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/30-attribution-import"
  service     = local.service
  ssm_root_map = {
    attribution-import = "/bcda/${var.env}/${local.service}/"
  }
}

data "aws_kms_alias" "bcda_app_config_kms_key" {
  name = "alias/bcda-${var.env}-app-config-kms"
}

data "aws_rds_cluster" "this" {
  cluster_identifier = "${local.app}-${var.env}-aurora"
}

resource "aws_kms_key" "attribution-import_bucket" {
  description             = "Custom KMS key for encrypting the attribution import file bucket"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_kms_alias" "attribution-import_bucket" {
  name          = "alias/bcda-${var.env}-attribution-import-bucket-key"
  target_key_id = aws_kms_key.attribution-import_bucket.key_id
}

module "attribution_import_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=8a6527c0689bb46ae0e74bd47e4087ab59cff1b0"

  architecture = "arm64"

  name        = local.service
  description = "Ingests the most recent attribution from EFT"

  handler = "bootstrap"
  runtime = "provided.al2023"

  memory_size = 2048

  platform = {
    app               = module.platform.app
    env               = var.env
    service           = local.service
    kms_alias_primary = { target_key_arn = module.platform.kms_alias_primary.target_key_arn }
    primary_region    = { name = module.platform.region_name }
    account_id        = module.platform.account_id
  }
  liveness_check_enabled = false

  additional_admin_role_arns = [module.platform.ssm.attribution-import.misp-eft-role_arn.value]
  github_actions_repos       = ["bcda-app:*"]

  environment_variables = {
    ENV      = var.env
    APP_NAME = "${local.app}-${var.env}-${local.service}"
  }

  ssm_parameter_paths = [
    "/bcda/${var.env}/sensitive/api/DATABASE_URL"
  ]

  extra_kms_key_arns = [
    module.platform.kms_alias_primary.target_key_arn,
    module.platform.kms_alias_secondary.target_key_arn,
    data.aws_kms_alias.bcda_app_config_kms_key.target_key_arn,
    aws_kms_key.attribution-import_bucket.arn
  ]
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
  source = "github.com/CMSgov/cdap//terraform/modules/bucket?ref=6ded520857376f46bb317dca898e5df6a9ecc93b"

  app           = local.app
  env           = var.env
  name          = "${local.full_name}-file"
  kms_key_arn   = join("", ["", aws_kms_key.attribution-import_bucket.arn])
  ssm_parameter = "/${local.app}/${var.env}/${local.service}/nonsensitive/file_bucket_name"
}

resource "aws_sns_topic" "this" {
  name              = "${local.full_name}-topic"
  kms_master_key_id = module.platform.kms_alias_primary.arn
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

resource "aws_sns_topic_policy" "this" {
  arn    = aws_sns_topic.this.arn
  policy = data.aws_iam_policy_document.topic.json
}

resource "aws_sns_topic_subscription" "this" {
  endpoint  = module.attribution_import_queue.arn
  protocol  = "sqs"
  topic_arn = aws_sns_topic.this.arn
}

resource "aws_s3_bucket_notification" "this" {
  bucket = module.attribution-import_file_bucket.id

  topic {
    topic_arn = aws_sns_topic.this.arn
    events    = ["s3:ObjectCreated:*"]
  }
}
