locals {
  app        = "bcda"
  service    = "attribution-import"
  full_name  = "${local.app}-${var.env}-attribution-import"
  db_sg_name = "${local.app}-${var.env}-db"
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
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/20-attribution-import"
  service     = local.service
  ssm_root_map = {
    attribution-import = "/bcda/${var.env}/${local.service}/"
  }
}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/aws/lambda/${local.full_name}"
  retention_in_days = 180

  tags = {
    Name = "/aws/lambda/${local.full_name}"
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

# ---------------------------------------------------------------------------
# Managed policies
# ---------------------------------------------------------------------------
data "aws_iam_policy_document" "default_function" {
  statement {
    sid = "SsmSqsLogsEc2"
    actions = [
      "ssm:GetParameters",
      "ssm:GetParameter",
      "sqs:ReceiveMessage",
      "sqs:GetQueueAttributes",
      "sqs:DeleteMessage",
      "logs:PutLogEvents",
      "logs:CreateLogStream",
      "logs:CreateLogGroup",
    ]
    resources = ["*"]
  }
  statement {
    sid = "KmsEncryptDecrypt"
    actions = [
      "kms:GenerateDataKey",
      "kms:Encrypt",
      "kms:Decrypt",
    ]
    resources = [
      module.platform.kms_alias_primary.arn,
      module.platform.kms_alias_secondary.arn
    ]
  }
}

data "aws_iam_policy_document" "attribution-import_bucket_rw" {

  statement {
    sid    = "ListBucket"
    effect = "Allow"

    actions = [
      "s3:ListBucket",
      "s3:GetBucketLocation",
    ]

    resources = [
      module.attribution-import_file_bucket.arn,
    ]
  }

  statement {
    sid    = "ReadWriteObjects"
    effect = "Allow"

    actions = [
      # Read
      "s3:GetObject",
      "s3:GetObjectVersion",
      "s3:GetObjectTagging",
      "s3:PutObject",
      "s3:PutObjectTagging",
      "s3:DeleteObject",
      "s3:DeleteObjectVersion",
      "s3:AbortMultipartUpload",
      "s3:ListMultipartUploadParts",
    ]
    resources = [
      "${module.attribution-import_file_bucket.arn}/*", "arn:aws:logs:*:*:*"
    ]
  }
}

module "attribution_import_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=2874c72ccd4c4821e5e3f77ccf61cf77ed05169f"

  app = local.app
  env = var.env

  name        = local.full_name
  description = "Ingests the most recent attribution from BFD"

  handler = "bootstrap"
  runtime = "provided.al2023"

  memory_size  = 2048
  architecture = "arm64"

  function_role_inline_policies = {
    assume-bucket-role = data.aws_iam_policy_document.assume_bucket_role.json
  }

  environment_variables = {
    ENV      = var.env
    APP_NAME = "${local.app}-${var.env}-attribution-import"
  }
  extra_kms_key_arns = [module.platform.kms_alias_primary.target_key_arn, module.platform.kms_alias_secondary.target_key_arn]
}

resource "aws_iam_role_policy" "attribution-import_bucket_rw" {
  name   = "attribution-import-bucket-rw"
  role   = "bcda-${var.env}-attribution-import-function"
  policy = data.aws_iam_policy_document.attribution-import_bucket_rw.json
}

resource "aws_iam_role_policy" "logging" {
  name   = "attribution-import-logging"
  role   = "${local.full_name}-function"
  policy = data.aws_iam_policy_document.default_function.json
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
  name          = "${local.full_name}-file"
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

data "aws_security_group" "db" {
  name = local.db_sg_name
}

resource "aws_security_group_rule" "function_access" {
  type        = "ingress"
  from_port   = 5432
  to_port     = 5432
  protocol    = "tcp"
  description = "${local.full_name} function access"

  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = module.attribution_import_function.security_group_id
}

resource "aws_s3_bucket_notification" "this" {
  bucket = module.attribution-import_file_bucket.id

  topic {
    topic_arn = aws_sns_topic.this.arn
    events    = ["s3:ObjectCreated:*"]
  }
}
